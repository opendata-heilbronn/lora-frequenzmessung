<template>
  <div>
    <h2>Add New Sensor</h2>

    <!-- Step indicators -->
    <div class="steps">
      <div
        v-for="(label, i) in stepLabels"
        :key="i"
        class="step"
        :class="{ active: step === i + 1, done: step > i + 1 }"
      >
        <span class="step-num">{{ step > i + 1 ? '✓' : i + 1 }}</span>
        <span class="step-label">{{ label }}</span>
      </div>
    </div>

    <!-- Step 1: Details -->
    <div v-if="step === 1" class="card">
      <h3>Sensor Details</h3>
      <form @submit.prevent="createSensor">
        <label>
          Name
          <input v-model="form.name" type="text" required placeholder="e.g. density-02" />
        </label>
        <label>
          Latitude
          <input v-model.number="form.latitude" type="number" step="any" required placeholder="49.1438602" />
        </label>
        <label>
          Longitude
          <input v-model.number="form.longitude" type="number" step="any" required placeholder="9.2149624" />
        </label>
        <div class="actions">
          <button type="submit" class="btn-primary" :disabled="creating">
            {{ creating ? 'Creating…' : 'Create Sensor →' }}
          </button>
        </div>
        <div v-if="createError" class="error-msg">{{ createError }}</div>
      </form>
    </div>

    <!-- Step 2: Created -->
    <div v-if="step === 2" class="card">
      <h3>Sensor Created</h3>
      <div class="info-grid">
        <div class="info-row"><span>UUID</span><code>{{ sensor.uuid }}</code></div>
        <div class="info-row"><span>Name</span><code>{{ sensor.name }}</code></div>
        <div class="info-row"><span>Type</span><code>{{ sensor.type }}</code></div>
      </div>
      <div class="actions">
        <button class="btn-primary" @click="startTTN">Register with TTN →</button>
        <button class="btn-secondary" @click="skipTTN">Skip TTN, Build Firmware →</button>
      </div>
    </div>

    <!-- Step 3: TTN Registration -->
    <div v-if="step === 3" class="card">
      <h3>TTN Registration</h3>
      <div v-if="ttnRegistering" class="status-building">
        <span class="spinner">⏳</span> Registering device with The Things Network…
      </div>
      <div v-else-if="ttnData" class="status-done">
        Device registered with TTN!
        <div class="info-grid" style="margin-top: 1rem;">
          <div class="info-row"><span>DevEUI</span><code>{{ ttnData.dev_eui }}</code></div>
          <div class="info-row"><span>AppKey</span><code>{{ ttnData.app_key }}</code></div>
          <div class="info-row"><span>TTN Device</span><code>{{ ttnData.ttn_device_id }}</code></div>
        </div>
      </div>
      <div v-else-if="ttnError" class="error-msg">
        TTN registration failed: {{ ttnError }}
      </div>
      <div class="actions">
        <button v-if="ttnData || ttnError" class="btn-primary" @click="startBuild">Build Firmware →</button>
        <button v-if="ttnError" class="btn-secondary" @click="doRegisterTTN">Retry TTN</button>
      </div>
    </div>

    <!-- Step 4: Build firmware -->
    <div v-if="step === 4" class="card">
      <h3>Build Firmware</h3>
      <div v-if="!hasTTN" class="notice">
        Firmware will be built <strong>without LoRa</strong> — sensor will count PAX but not transmit.
      </div>
      <div v-if="buildStatus === 'building'" class="status-building">
        <span class="spinner">⏳</span> Compiling firmware with PlatformIO…
        <p class="hint">This typically takes 30–60 seconds.</p>
      </div>
      <div v-else-if="buildStatus === 'done'" class="status-done">
        Firmware compiled successfully!
      </div>
      <div v-else-if="buildStatus === 'error'" class="error-msg">
        Build failed: {{ buildMessage }}
      </div>
      <div v-else class="status-idle">
        <button class="btn-primary" @click="startBuild">Start Build</button>
      </div>
      <div v-if="buildStatus === 'done'" class="actions">
        <button class="btn-primary" @click="step = 5">Flash Sensor →</button>
      </div>
    </div>

    <!-- Step 5: Flash -->
    <div v-if="step === 5" class="card">
      <h3>Flash Sensor via USB</h3>
      <div class="flash-instructions">
        <p>Connect your ESP32 sensor via USB, then click <strong>Install</strong>.</p>
        <p class="hint">Requires Chrome or Edge browser. Web Serial API must be enabled.</p>
      </div>
      <FlashStep :manifest-url="`/api/sensors/${sensor.uuid}/manifest.json`" />
      <div class="actions">
        <router-link to="/" class="btn-secondary">← Back to sensor list</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import axios from 'axios'
import FlashStep from '../components/FlashStep.vue'

const stepLabels = ['Details', 'Created', 'TTN', 'Build', 'Flash']
const step = ref(1)

const form = ref({ name: '', latitude: 0, longitude: 0 })
const creating = ref(false)
const createError = ref('')

const sensor = ref({ uuid: '', name: '', type: '' })

const ttnRegistering = ref(false)
const ttnError = ref('')
const ttnData = ref<{ dev_eui: string; app_key: string; ttn_device_id: string } | null>(null)
const hasTTN = ref(false)

const buildStatus = ref<'idle' | 'building' | 'done' | 'error'>('idle')
const buildMessage = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

async function createSensor() {
  creating.value = true
  createError.value = ''
  try {
    const { data } = await axios.post('/api/sensors', {
      name: form.value.name,
      latitude: form.value.latitude,
      longitude: form.value.longitude,
    })
    sensor.value = data
    step.value = 2
  } catch (e: unknown) {
    createError.value = e instanceof Error ? e.message : 'Sensor creation failed'
  } finally {
    creating.value = false
  }
}

function startTTN() {
  step.value = 3
  doRegisterTTN()
}

async function doRegisterTTN() {
  ttnRegistering.value = true
  ttnError.value = ''
  ttnData.value = null
  try {
    const { data } = await axios.post(`/api/sensors/${sensor.value.uuid}/register-ttn`)
    ttnData.value = data
    hasTTN.value = true
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.data?.error) {
      ttnError.value = e.response.data.error
    } else {
      ttnError.value = e instanceof Error ? e.message : 'Registration failed'
    }
  } finally {
    ttnRegistering.value = false
  }
}

function skipTTN() {
  hasTTN.value = false
  startBuild()
}

async function startBuild() {
  step.value = 4
  buildStatus.value = 'building'
  try {
    await axios.post(`/api/sensors/${sensor.value.uuid}/build-firmware`)
  } catch {
    // If already building, continue polling
  }
  pollBuildStatus()
}

function pollBuildStatus() {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const { data } = await axios.get(`/api/sensors/${sensor.value.uuid}/build-status`)
      if (data.status === 'done') {
        buildStatus.value = 'done'
        clearInterval(pollTimer!)
      } else if (data.status === 'error') {
        buildStatus.value = 'error'
        buildMessage.value = data.message || 'Unknown error'
        clearInterval(pollTimer!)
      }
    } catch {
      // ignore transient errors
    }
  }, 3000)
}
</script>

<style scoped>
h2 { margin-top: 0; }
.steps { display: flex; gap: 0; margin-bottom: 2rem; }
.step { display: flex; align-items: center; gap: .5rem; padding: .75rem 1.25rem; background: #eee; flex: 1; position: relative; }
.step::after { content: ''; position: absolute; right: -14px; top: 0; bottom: 0; width: 0; border-top: 28px solid transparent; border-bottom: 28px solid transparent; border-left: 14px solid #eee; z-index: 1; }
.step:last-child::after { display: none; }
.step.active { background: #1a1a2e; color: white; }
.step.active::after { border-left-color: #1a1a2e; }
.step.done { background: #27ae60; color: white; }
.step.done::after { border-left-color: #27ae60; }
.step-num { width: 24px; height: 24px; border-radius: 50%; background: rgba(255,255,255,.25); display: flex; align-items: center; justify-content: center; font-weight: bold; font-size: .85rem; }
.step-label { font-size: .9rem; font-weight: 500; }
.card { background: white; border-radius: 8px; padding: 2rem; box-shadow: 0 1px 4px rgba(0,0,0,.1); }
.card h3 { margin-top: 0; }
label { display: flex; flex-direction: column; gap: .35rem; margin-bottom: 1rem; font-weight: 500; }
input { padding: .6rem .75rem; border: 1px solid #ccc; border-radius: 6px; font-size: 1rem; width: 100%; }
input:focus { outline: 2px solid #1a1a2e; border-color: transparent; }
.actions { margin-top: 1.5rem; display: flex; gap: 1rem; }
.btn-primary { background: #1a1a2e; color: white; border: none; padding: .65rem 1.25rem; border-radius: 6px; cursor: pointer; font-size: 1rem; }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-primary:not(:disabled):hover { background: #2d2d5a; }
.btn-secondary { background: #eee; color: #333; padding: .65rem 1.25rem; border-radius: 6px; text-decoration: none; font-size: 1rem; border: none; cursor: pointer; }
.btn-secondary:hover { background: #ddd; }
.info-grid { border: 1px solid #eee; border-radius: 6px; overflow: hidden; }
.info-row { display: flex; padding: .65rem 1rem; border-bottom: 1px solid #eee; }
.info-row:last-child { border-bottom: none; }
.info-row span:first-child { width: 120px; font-weight: 600; color: #666; flex-shrink: 0; }
.info-row code { font-family: monospace; font-size: .9rem; word-break: break-all; }
.badge-ok { background: #e8f8ef; color: #27ae60; padding: .2rem .6rem; border-radius: 4px; font-weight: 600; }
.badge-warn { background: #fdf0e8; color: #e67e22; padding: .2rem .6rem; border-radius: 4px; font-weight: 600; }
.error-msg { color: #c0392b; margin-top: .75rem; padding: .75rem; background: #fdecea; border-radius: 6px; }
.notice { background: #fef9e7; border: 1px solid #f9e79f; color: #7d6608; padding: .75rem 1rem; border-radius: 6px; margin-bottom: 1rem; }
.status-building { display: flex; flex-direction: column; align-items: center; padding: 2rem; gap: .5rem; }
.spinner { font-size: 2rem; animation: spin 2s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.hint { color: #999; font-size: .85rem; margin: 0; }
.status-done { color: #27ae60; font-size: 1.1rem; font-weight: 600; padding: 1rem 0; }
.flash-instructions p { margin: 0 0 .5rem; }
</style>
