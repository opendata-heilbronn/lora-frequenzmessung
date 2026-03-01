<template>
  <div class="flash-provision-wrap">
    <OnyxInfoCard v-if="!webSerialSupported" color="warning" headline="Web Serial not supported.">
      Use Chrome or Edge (desktop) to flash and provision the sensor via USB.
    </OnyxInfoCard>

    <template v-else>
      <!-- Idle -->
      <div v-if="phase === 'idle'">
        <p>Connect your ESP32 sensor via USB and hold the BOOT button, then click below.</p>
        <p class="hint">The device will be erased, flashed and provisioned in one step. Requires Chrome or Edge.</p>
        <details class="advanced">
          <summary>Advanced settings</summary>
          <div class="advanced-grid">
            <OnyxInput label="Factor" :model-value="factorStr" @update:model-value="onFactorChange" />
            <OnyxInput label="Sleep (sec)" :model-value="sleepStr" @update:model-value="onSleepChange" />
          </div>
        </details>
        <div class="action-row">
          <OnyxButton label="⚡ Connect & Flash Sensor" color="primary" @click="startFlash" />
        </div>
      </div>

      <!-- Erasing -->
      <div v-else-if="phase === 'erasing'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Erasing flash…</span>
      </div>

      <!-- Flashing -->
      <div v-else-if="phase === 'flashing'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Writing firmware… {{ flashPercent }}%</span>
        <div class="progress-bar-wrap">
          <div class="progress-bar" :style="{ width: flashPercent + '%' }"></div>
        </div>
      </div>

      <!-- Rebooting -->
      <div v-else-if="phase === 'rebooting'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Flash complete! Waiting for device to reboot…</span>
      </div>

      <!-- Provisioning -->
      <div v-else-if="phase === 'provisioning'" class="status-block">
        <OnyxLoadingIndicator type="circle" />
        <span>Provisioning device…</span>
        <pre class="log">{{ provLog }}</pre>
      </div>

      <!-- Done -->
      <OnyxInfoCard v-else-if="phase === 'done'" color="success">
        Device flashed and provisioned successfully!
      </OnyxInfoCard>

      <!-- Error -->
      <div v-else-if="phase === 'error'">
        <OnyxInfoCard color="danger">{{ errorMsg }}</OnyxInfoCard>
        <div class="error-actions">
          <OnyxButton label="Try Again" @click="reset" />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { OnyxButton, OnyxInfoCard, OnyxLoadingIndicator, OnyxInput } from 'sit-onyx'
import { flash as espFlashImpl } from 'esp-web-tools/dist/flash.js'
import api from '../api'

const props = defineProps<{
  manifestUrl: string
  sensorUuid: string
  devEui: string
  appKey: string
  joinEui: string
}>()

type Phase = 'idle' | 'erasing' | 'flashing' | 'rebooting' | 'provisioning' | 'done' | 'error'

const webSerialSupported = ref(false)
const phase = ref<Phase>('idle')
const flashPercent = ref(0)
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

onMounted(() => {
  webSerialSupported.value = 'serial' in navigator
})

async function startFlash() {
  // Allow tests to inject a mock flash function via window.__espFlash
  const espFlash: typeof espFlashImpl = (window as any).__espFlash ?? espFlashImpl
  // Allow tests to shorten the reboot wait via window.__rebootWaitMs
  const rebootWaitMs: number = (window as any).__rebootWaitMs ?? 4000

  // 1. Request serial port — if user cancels, stay on idle
  let port: any
  try {
    port = await (navigator as any).serial.requestPort()
  } catch {
    return
  }

  // 2. Fetch manifest JSON
  let manifest: any
  try {
    const res = await fetch(props.manifestUrl)
    if (!res.ok) throw new Error(`Manifest fetch failed: ${res.status}`)
    manifest = await res.json()
  } catch (e: any) {
    errorMsg.value = e?.message || 'Failed to fetch firmware manifest.'
    phase.value = 'error'
    return
  }

  // 3. Erase + flash (flash() disconnects port when done)
  phase.value = 'erasing'
  flashPercent.value = 0
  let flashFailed = false

  try {
    await espFlash(
      (s: any) => {
        if (s.state === 'erasing') {
          phase.value = 'erasing'
        } else if (s.state === 'writing') {
          phase.value = 'flashing'
          flashPercent.value = s.details?.percentage ?? 0
        } else if (s.state === 'error') {
          flashFailed = true
          errorMsg.value = s.message || 'Flash failed. Check USB connection and try again.'
          phase.value = 'error'
        }
      },
      port,
      props.manifestUrl,
      manifest,
      true, // eraseFirst
    )
  } catch (e: any) {
    errorMsg.value = e?.message || 'Flash failed unexpectedly.'
    phase.value = 'error'
    return
  }

  if (flashFailed) return

  // 4. Wait for device to reboot (port was closed by flash())
  phase.value = 'rebooting'
  await new Promise(resolve => setTimeout(resolve, rebootWaitMs))

  // 5. Reopen port and provision
  await doProvision(port)
}

async function doProvision(port: any) {
  phase.value = 'provisioning'

  try {
    await port.open({ baudRate: 115200 })
  } catch (e: any) {
    errorMsg.value = e?.message || 'Failed to reopen serial port for provisioning.'
    phase.value = 'error'
    return
  }

  try {
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

    // Fetch WiFi credentials from server (optional — skip if not configured)
    let wifiSsid = ''
    let wifiPassword = ''
    try {
      const cfg = await api.get<{ wifi_ssid: string; wifi_password: string }>('/api/provision-config')
      wifiSsid = cfg.data.wifi_ssid ?? ''
      wifiPassword = cfg.data.wifi_password ?? ''
    } catch {
      // Non-fatal — proceed without WiFi credentials
    }

    const lines = [
      `sensor_id=${props.sensorUuid}`,
      `factor=${factor.value}`,
      `sleep_sec=${sleepSec.value}`,
      `joineui=${props.joinEui}`,
      `deveui=${props.devEui}`,
      `appkey=${props.appKey}`,
      ...(wifiSsid     ? [`wifi_ssid=${wifiSsid}`]         : []),
      ...(wifiPassword ? [`wifi_password=${wifiPassword}`] : []),
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
      // Device may restart immediately — treat as success
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
  flashPercent.value = 0
  provLog.value = ''
  errorMsg.value = ''
}
</script>

<style scoped>
.flash-provision-wrap { margin-top: 1rem; }

.action-row {
  margin-top: 1.5rem;
}

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

.progress-bar-wrap {
  width: 100%;
  max-width: 320px;
  height: 6px;
  background: var(--onyx-color-base-neutral-200);
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: var(--onyx-color-base-primary-500);
  border-radius: 3px;
  transition: width 0.3s ease;
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
  margin: 0.25rem 0 0;
}
</style>
