<template>
  <div class="flash-provision-wrap">
    <OnyxInfoCard v-if="!webSerialSupported" color="warning" headline="Web Serial not supported.">
      Use Chrome or Edge (desktop) to flash and provision the sensor via USB.
    </OnyxInfoCard>

    <template v-else>
      <!-- Idle: flash button kept in DOM via v-show so event listener persists across phases -->
      <div v-show="phase === 'idle'">
        <p>Connect your ESP32 sensor via USB, then click the button below.</p>
        <p class="hint">Requires Chrome or Edge browser.</p>
        <div class="flash-btn-wrap">
          <esp-web-install-button ref="espBtnRef" :manifest="manifestUrl">
            <button slot="activate" class="flash-button">⚡ Connect & Flash Sensor</button>
            <span slot="unsupported">Your browser does not support Web Serial. Please use Chrome or Edge.</span>
          </esp-web-install-button>
        </div>
        <details class="advanced">
          <summary>Advanced settings</summary>
          <div class="advanced-grid">
            <OnyxInput label="Factor" :model-value="factorStr" @update:model-value="onFactorChange" />
            <OnyxInput label="Sleep (sec)" :model-value="sleepStr" @update:model-value="onSleepChange" />
          </div>
        </details>
      </div>

      <!-- Flashing progress -->
      <div v-if="phase === 'flashing'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Flashing… {{ flashStateLabel }}</span>
      </div>

      <!-- Rebooting -->
      <div v-if="phase === 'rebooting'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Flash complete! Waiting for device to reboot…</span>
      </div>

      <!-- Provisioning -->
      <div v-if="phase === 'provisioning'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Provisioning device…</span>
        <pre class="log">{{ provLog }}</pre>
      </div>

      <!-- Done -->
      <OnyxInfoCard v-if="phase === 'done'" color="success">
        Device flashed and provisioned successfully!
      </OnyxInfoCard>

      <!-- Error -->
      <div v-if="phase === 'error'">
        <OnyxInfoCard color="danger">{{ errorMsg }}</OnyxInfoCard>
        <div class="error-actions">
          <OnyxButton label="Try Again" @click="reset" />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { OnyxButton, OnyxInfoCard, OnyxLoadingIndicator, OnyxInput } from 'sit-onyx'
import 'esp-web-tools'

const props = defineProps<{
  manifestUrl: string
  sensorUuid: string
  devEui: string
  appKey: string
  joinEui: string
}>()

type Phase = 'idle' | 'flashing' | 'rebooting' | 'provisioning' | 'done' | 'error'

const webSerialSupported = ref(false)
const phase = ref<Phase>('idle')
const flashState = ref('')
const provLog = ref('')
const errorMsg = ref('')

const factor = ref(0.7)
const sleepSec = ref(900)
const factorStr = computed(() => String(factor.value))
const sleepStr = computed(() => String(sleepSec.value))

function onFactorChange(v?: string) {
  const n = Number(v)
  if (!Number.isNaN(n)) factor.value = n
}

function onSleepChange(v?: string) {
  const n = Number(v)
  if (!Number.isNaN(n)) sleepSec.value = Math.max(1, Math.round(n))
}

const flashStateLabel = computed(() => {
  const labels: Record<string, string> = {
    INITIALIZING: 'Initializing…',
    MANIFEST: 'Loading manifest…',
    PREPARING: 'Preparing device…',
    ERASING: 'Erasing flash…',
    WRITING: 'Writing firmware…',
    FINISHED: 'Done!',
  }
  return labels[flashState.value] || flashState.value
})

const espBtnRef = ref<Element | null>(null)
let rebootTimer: ReturnType<typeof setTimeout> | null = null

function onFlashStateChanged(e: Event) {
  const state = (e as CustomEvent).detail?.state as string
  if (state === 'FINISHED') {
    phase.value = 'rebooting'
    rebootTimer = setTimeout(() => startProvisioning(), 4000)
  } else if (state === 'ERROR') {
    errorMsg.value = 'Flash failed. Check USB connection and try again.'
    phase.value = 'error'
  } else {
    phase.value = 'flashing'
    flashState.value = state
  }
}

onMounted(async () => {
  webSerialSupported.value = 'serial' in navigator
  // Wait for Vue to re-render the conditional template (webSerialSupported gate)
  // before attaching the listener — espBtnRef.value is null until the DOM updates.
  await nextTick()
  espBtnRef.value?.addEventListener('state-changed', onFlashStateChanged)
})

onUnmounted(() => {
  espBtnRef.value?.removeEventListener('state-changed', onFlashStateChanged)
  if (rebootTimer) clearTimeout(rebootTimer)
})

async function startProvisioning() {
  phase.value = 'provisioning'
  const nav = navigator as any
  if (!('serial' in nav)) {
    errorMsg.value = 'Web Serial not supported.'
    phase.value = 'error'
    return
  }
  let port: any
  try {
    const remembered = await nav.serial.getPorts()
    port = remembered.length > 0 ? remembered[0] : await nav.serial.requestPort()
    await doProvision(port)
  } catch (e: any) {
    errorMsg.value = e?.message || String(e)
    phase.value = 'error'
  }
}

async function doProvision(port: any) {
  try {
    await port.open({ baudRate: 115200 })
    const decoder = new TextDecoder()
    const encoder = new TextEncoder()
    const reader = port.readable.getReader()
    const writer = port.writable.getWriter()

    const readUntil = async (needle: string, timeoutMs = 8000) => {
      const deadline = Date.now() + timeoutMs
      let buf = ''
      while (Date.now() < deadline) {
        const { value, done } = await reader.read()
        if (done) break
        if (value) buf += decoder.decode(value)
        if (buf.includes(needle)) return buf
      }
      throw new Error(`Timed out waiting for: ${needle}`)
    }

    const writeLine = async (line: string) => {
      await writer.write(encoder.encode(line + '\n'))
    }

    provLog.value += 'Waiting for PROV_READY...\n'
    await readUntil('PROV_READY', 15000)

    const lines = [
      `sensor_id=${props.sensorUuid}`,
      `factor=${factor.value}`,
      `sleep_sec=${sleepSec.value}`,
      `joineui=${props.joinEui}`,
      `deveui=${props.devEui}`,
      `appkey=${props.appKey}`,
    ]

    for (const ln of lines) {
      provLog.value += `> ${ln}\n`
      await writeLine(ln)
      const out = await readUntil('OK')
      provLog.value += out
      if (out.includes('ERROR')) throw new Error(out.trim())
    }

    provLog.value += '> COMMIT\n'
    await writeLine('COMMIT')
    try {
      const out = await readUntil('RESTART', 5000)
      provLog.value += out
    } catch {
      // Device may restart immediately and close the port; treat as success
    }

    phase.value = 'done'
    try { reader.releaseLock() } catch {}
    try { writer.releaseLock() } catch {}
    try { await port.close() } catch {}
  } catch (e: any) {
    errorMsg.value = e?.message || String(e)
    phase.value = 'error'
  }
}

function reset() {
  phase.value = 'idle'
  flashState.value = ''
  provLog.value = ''
  errorMsg.value = ''
  if (rebootTimer) {
    clearTimeout(rebootTimer)
    rebootTimer = null
  }
}
</script>

<style scoped>
.flash-provision-wrap { margin-top: 1rem; }

.flash-btn-wrap {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin: 1rem 0;
}

.flash-button {
  font-family: var(--onyx-font-family-paragraph);
  font-size: var(--onyx-font-size-md);
  font-weight: 600;
  background: var(--onyx-color-base-primary-500);
  color: var(--onyx-color-text-icons-neutral-inverted);
  border: none;
  padding: var(--onyx-density-sm) var(--onyx-density-lg);
  border-radius: var(--onyx-radius-md);
  cursor: pointer;
}

.flash-button:hover { background: var(--onyx-color-base-primary-600); }

.advanced { margin-top: 1rem; }

.advanced-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  margin-top: 0.5rem;
}

.status-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem;
  gap: 0.5rem;
}

.log {
  margin-top: 0.5rem;
  max-height: 220px;
  overflow: auto;
  background: var(--onyx-color-base-neutral-100);
  border: 1px solid var(--onyx-color-base-neutral-300);
  padding: 0.5rem;
  border-radius: var(--onyx-radius-sm);
  width: 100%;
  box-sizing: border-box;
}

.error-actions { margin-top: 0.75rem; }

.hint {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-size: var(--onyx-font-size-sm);
  margin: 0;
}
</style>
