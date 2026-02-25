<template>
  <div>
    <OnyxHeadline is="h2">Add New Sensor</OnyxHeadline>

    <!-- Step indicators -->
    <OnyxProgressSteps :steps="progressSteps" :model-value="step" style="margin-bottom: 2rem;" />

    <!-- Step 1: Details -->
    <OnyxCard v-if="step === 1">
      <OnyxHeadline is="h3">Sensor Details</OnyxHeadline>
      <form class="form-fields" @submit.prevent="createSensor">
        <OnyxInput
          label="Name"
          :model-value="form.name"
          @update:model-value="form.name = $event ?? ''"
          type="text"
          required
          :max-length="64"
          placeholder="e.g. density-02"
        />
        <p class="hint">Location — click on the map or drag the marker, or enter coordinates manually.</p>
        <MapPicker v-model="mapCoords" />
        <div class="coord-row">
          <label class="coord-label">
            Latitude
            <input v-model.number="form.latitude" type="number" step="any" required min="-90" max="90" placeholder="49.1438602" class="coord-input" />
          </label>
          <label class="coord-label">
            Longitude
            <input v-model.number="form.longitude" type="number" step="any" required min="-180" max="180" placeholder="9.2149624" class="coord-input" />
          </label>
        </div>
        <div class="actions">
          <OnyxButton type="submit" label="Create Sensor →" color="primary" :loading="creating" :disabled="creating" />
        </div>
        <div v-if="createError" class="error-msg">
          <OnyxInfoCard color="danger">{{ createError }}</OnyxInfoCard>
        </div>
      </form>
    </OnyxCard>

    <!-- Step 2: Created -->
    <OnyxCard v-if="step === 2">
      <OnyxHeadline is="h3">Sensor Created</OnyxHeadline>
      <dl class="info-grid">
        <div class="info-row"><dt>UUID</dt><dd><code>{{ sensor.uuid }}</code></dd></div>
        <div class="info-row"><dt>Name</dt><dd><code>{{ sensor.name }}</code></dd></div>
        <div class="info-row"><dt>Type</dt><dd><code>{{ sensor.type }}</code></dd></div>
      </dl>
      <div class="actions">
        <OnyxButton label="Register with TTN →" color="primary" @click="startTTN" />
        <OnyxButton label="Skip TTN, Build Firmware →" color="neutral" @click="skipTTN" />
      </div>
    </OnyxCard>

    <!-- Step 3: TTN Registration -->
    <OnyxCard v-if="step === 3">
      <OnyxHeadline is="h3">TTN Registration</OnyxHeadline>
      <div v-if="ttnRegistering" class="status-building">
        <OnyxLoadingIndicator type="circle" />
        <span>Registering device with The Things Network…</span>
      </div>
      <div v-else-if="ttnData">
        <OnyxInfoCard color="success">Device registered with TTN!</OnyxInfoCard>
        <dl class="info-grid" style="margin-top: 1rem;">
          <div class="info-row">
            <dt>DevEUI</dt>
            <dd>
              <code>{{ maskKey(ttnData.dev_eui) }}</code>
              <OnyxButton label="Copy" mode="outline" color="neutral" density="compact" @click="copyToClipboard(ttnData!.dev_eui)" />
            </dd>
          </div>
          <div class="info-row">
            <dt>AppKey</dt>
            <dd>
              <code>{{ maskKey(ttnData.app_key) }}</code>
              <OnyxButton label="Copy" mode="outline" color="neutral" density="compact" @click="copyToClipboard(ttnData!.app_key)" />
            </dd>
          </div>
          <div class="info-row"><dt>TTN Device</dt><dd><code>{{ ttnData.ttn_device_id }}</code></dd></div>
        </dl>
      </div>
      <div v-else-if="ttnError" class="error-msg">
        <OnyxInfoCard color="danger">TTN registration failed: {{ ttnError }}</OnyxInfoCard>
      </div>
      <div class="actions">
        <OnyxButton v-if="ttnData || ttnError" label="Build Firmware →" color="primary" @click="startBuild" />
        <OnyxButton v-if="ttnError" label="Retry TTN" color="neutral" @click="doRegisterTTN" />
      </div>
    </OnyxCard>

    <!-- Step 4: Build firmware -->
    <OnyxCard v-if="step === 4">
      <OnyxHeadline is="h3">Build Firmware</OnyxHeadline>
      <OnyxInfoCard v-if="!hasTTN" color="warning" style="margin-bottom: 1rem;">
        Firmware will be built <strong>without LoRa</strong> — sensor will count PAX but not transmit.
      </OnyxInfoCard>
      <div v-if="buildStatus === 'building'" class="status-building">
        <OnyxLoadingIndicator type="circle" />
        <span>Compiling firmware with PlatformIO…</span>
        <p class="hint">This typically takes 30–60 seconds.</p>
      </div>
      <div v-else-if="buildStatus === 'done'">
        <OnyxInfoCard color="success">Firmware compiled successfully!</OnyxInfoCard>
      </div>
      <div v-else-if="buildStatus === 'error'" class="error-msg">
        <OnyxInfoCard color="danger">Build failed: {{ buildMessage }}</OnyxInfoCard>
      </div>
      <div v-else class="status-idle">
        <OnyxButton label="Start Build" color="primary" @click="startBuild" />
      </div>
      <div v-if="buildStatus === 'done'" class="actions">
        <OnyxButton label="Flash Sensor →" color="primary" @click="step = 5" />
      </div>
    </OnyxCard>

    <!-- Step 5: Flash -->
    <OnyxCard v-if="step === 5">
      <OnyxHeadline is="h3">Flash Sensor via USB</OnyxHeadline>
      <div class="flash-instructions">
        <p>Connect your ESP32 sensor via USB, then click <strong>Install</strong>.</p>
        <p class="hint">Requires Chrome or Edge browser. Web Serial API must be enabled.</p>
      </div>
      <FlashStep :manifest-url="`/api/sensors/${sensor.uuid}/manifest.json`" />
      <div class="actions">
        <OnyxButton label="← Back to sensor list" color="neutral" link="/" />
      </div>
    </OnyxCard>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import {
  OnyxHeadline,
  OnyxCard,
  OnyxInput,
  OnyxButton,
  OnyxLoadingIndicator,
  OnyxInfoCard,
  OnyxProgressSteps,
} from 'sit-onyx'
import api from '../api'
import axios from 'axios'
import FlashStep from '../components/FlashStep.vue'
import MapPicker from '../components/MapPicker.vue'

const progressSteps = [
  { label: 'Details' },
  { label: 'Created' },
  { label: 'TTN' },
  { label: 'Build' },
  { label: 'Flash' },
]

const step = ref(1)

const DEFAULT_LAT = 49.143845257365456
const DEFAULT_LNG = 9.214797255696741

const form = ref({ name: '', latitude: DEFAULT_LAT, longitude: DEFAULT_LNG })

const mapCoords = computed({
  get: () => ({ lat: form.value.latitude, lng: form.value.longitude }),
  set: (val: { lat: number; lng: number }) => {
    form.value.latitude = Math.round(val.lat * 1e7) / 1e7
    form.value.longitude = Math.round(val.lng * 1e7) / 1e7
  },
})
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

// Mask a hex key, showing first 4 and last 4 chars
function maskKey(key: string): string {
  if (key.length <= 8) return key
  return key.slice(0, 4) + '****' + key.slice(-4)
}

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    // Fallback: do nothing if clipboard API unavailable
  }
}

async function createSensor() {
  creating.value = true
  createError.value = ''

  // Client-side validation
  const name = form.value.name.trim()
  if (!name || name.length > 64) {
    createError.value = 'Name is required and must be 1-64 characters'
    creating.value = false
    return
  }
  if (form.value.latitude < -90 || form.value.latitude > 90) {
    createError.value = 'Latitude must be between -90 and 90'
    creating.value = false
    return
  }
  if (form.value.longitude < -180 || form.value.longitude > 180) {
    createError.value = 'Longitude must be between -180 and 180'
    creating.value = false
    return
  }

  try {
    const { data } = await api.post('/api/sensors', {
      name: name,
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
    const { data } = await api.post(`/api/sensors/${sensor.value.uuid}/register-ttn`)
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
    await api.post(`/api/sensors/${sensor.value.uuid}/build-firmware`)
  } catch {
    // If already building, continue polling
  }
  pollBuildStatus()
}

function pollBuildStatus() {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const { data } = await api.get(`/api/sensors/${sensor.value.uuid}/build-status`)
      if (data.status === 'done') {
        buildStatus.value = 'done'
        clearInterval(pollTimer!)
        pollTimer = null
      } else if (data.status === 'error') {
        buildStatus.value = 'error'
        buildMessage.value = data.message || 'Unknown error'
        clearInterval(pollTimer!)
        pollTimer = null
      }
    } catch {
      // ignore transient errors
    }
  }, 3000)
}

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>

<style scoped>
.form-fields {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.coord-row {
  display: flex;
  gap: 1rem;
}
.coord-label {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-weight: 500;
  font-size: var(--onyx-font-size-sm);
  color: var(--onyx-color-text-icons-neutral-medium);
}
.coord-input {
  padding: 0.6rem 0.75rem;
  border: 1px solid var(--onyx-color-base-neutral-300);
  border-radius: var(--onyx-radius-md);
  font-family: var(--onyx-font-family-paragraph), sans-serif;
  font-size: 1rem;
  width: 100%;
  box-sizing: border-box;
  background: var(--onyx-color-base-background-blank);
  color: var(--onyx-color-text-icons-neutral-intense);
}
.coord-input:focus {
  outline: 2px solid var(--onyx-color-base-primary-500);
  border-color: transparent;
}

.actions {
  margin-top: 1.5rem;
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.info-grid {
  border: 1px solid var(--onyx-color-base-neutral-200);
  border-radius: var(--onyx-radius-md);
  overflow: hidden;
  margin: 0;
  padding: 0;
}
.info-row {
  display: flex;
  padding: 0.65rem 1rem;
  border-bottom: 1px solid var(--onyx-color-base-neutral-200);
  align-items: center;
  gap: 0.5rem;
}
.info-row:nth-child(even) {
  background: var(--onyx-color-base-neutral-100);
}
.info-row:last-child { border-bottom: none; }
.info-row dt {
  width: 120px;
  font-weight: 600;
  color: var(--onyx-color-text-icons-neutral-medium);
  flex-shrink: 0;
}
.info-row dd {
  font-family: monospace;
  font-size: 0.9rem;
  word-break: break-all;
  flex: 1;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.error-msg { margin-top: 0.75rem; }

.status-building {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem;
  gap: 0.5rem;
}

.hint {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-size: var(--onyx-font-size-sm);
  margin: 0;
}

.flash-instructions p { margin: 0 0 0.5rem; }
</style>
