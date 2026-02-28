<template>
  <div class="prov-wrap">
    <OnyxInfoCard v-if="!webSerialSupported" color="warning" headline="Web Serial not supported.">
      Use Chrome or Edge (desktop) to provision the sensor via USB.
    </OnyxInfoCard>

    <div v-else>
      <div class="grid">
        <OnyxInput label="Factor" type="number" :step="0.01" :model-value="factorStr"
                    @update:model-value="onFactorChange" />
        <OnyxInput label="Sleep (sec)" type="number" :model-value="sleepStr"
                    @update:model-value="onSleepChange" />
      </div>

      <details class="details">
        <summary>Advanced (read-only)</summary>
        <dl class="kv">
          <div><dt>Sensor ID</dt><dd><code>{{ sensorUuid }}</code></dd></div>
          <div><dt>JoinEUI</dt><dd><code>{{ joinEui }}</code></dd></div>
          <div><dt>DevEUI</dt><dd><code>{{ devEui }}</code></dd></div>
          <div><dt>AppKey</dt><dd><code>{{ maskedAppKey }}</code></dd></div>
        </dl>
      </details>

      <div class="actions">
        <OnyxButton :disabled="busy" :loading="busy" color="primary" label="Connect & Provision"
                    @click="provision" />
      </div>

      <div v-if="message" class="log">
        <pre><code>{{ message }}</code></pre>
      </div>

      <OnyxInfoCard v-if="success" color="success" style="margin-top: 1rem;">
        Provisioning completed and device is rebooting.
      </OnyxInfoCard>
      <OnyxInfoCard v-if="error" color="danger" style="margin-top: 1rem;">
        {{ error }}
      </OnyxInfoCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { OnyxButton, OnyxInfoCard, OnyxInput } from 'sit-onyx'

const props = defineProps<{
  sensorUuid: string
  devEui: string
  appKey: string
  joinEui: string
}>()

const webSerialSupported = ref(false)
const busy = ref(false)
const success = ref(false)
const error = ref('')
const message = ref('')

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

const maskedAppKey = computed(() => props.appKey?.length > 8
  ? props.appKey.slice(0, 4) + '****' + props.appKey.slice(-4)
  : props.appKey)

onMounted(() => {
  webSerialSupported.value = 'serial' in navigator
})

async function provision() {
  error.value = ''
  success.value = false
  message.value = ''

  if (!('serial' in navigator)) {
    error.value = 'Web Serial API not supported in this browser.'
    return
  }

  busy.value = true
  try {
    const port = await (navigator as any).serial.requestPort()
    await port.open({ baudRate: 115200 })

    const decoder = new TextDecoder()
    const encoder = new TextEncoder()

    // Helpers
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

    // 1) Wait for device to announce provisioning readiness
    message.value += 'Waiting for PROV_READY...\n'
    await readUntil('PROV_READY', 15000)

    // 2) Send staged key=value lines and wait for OKs (device prints OK per valid line)
    const lines = [
      `sensor_id=${props.sensorUuid}`,
      `factor=${factor.value}`,
      `sleep_sec=${sleepSec.value}`,
      `joineui=${props.joinEui}`,
      `deveui=${props.devEui}`,
      `appkey=${props.appKey}`,
    ]

    for (const ln of lines) {
      message.value += `> ${ln}\n`
      await writeLine(ln)
      const out = await readUntil('OK')
      message.value += out
      if (out.includes('ERROR')) throw new Error(out.trim())
    }

    // 3) Commit and expect reboot
    message.value += '> COMMIT\n'
    await writeLine('COMMIT')
    try {
      const out = await readUntil('RESTART', 5000)
      message.value += out
    } catch {
      // Some firmwares restart immediately and close the port; treat as success
    }

    success.value = true

    try { reader.releaseLock(); } catch {}
    try { writer.releaseLock(); } catch {}
    try { await port.close(); } catch {}
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.prov-wrap { margin-top: 1rem; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; margin-bottom: 0.5rem; }
.kv { display: grid; grid-template-columns: max-content 1fr; gap: 0.25rem 1rem; margin-top: 0.5rem; }
.kv dt { font-weight: 600; }
.details { margin: 0.5rem 0 1rem; }
.actions { margin-top: 0.5rem; display: flex; gap: 0.75rem; }
.log { margin-top: 0.5rem; max-height: 220px; overflow: auto; background: var(--onyx-color-base-neutral-100); border: 1px solid var(--onyx-color-base-neutral-300); padding: 0.5rem; border-radius: var(--onyx-radius-sm); }
</style>
