<template>
  <OnyxPageLayout>
    <OnyxHeadline is="h2">Add New Sensor</OnyxHeadline>

    <!-- Stage 1: Form -->
    <OnyxCard v-if="setupPhase === 'idle'">
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
            <input v-model.number="form.latitude" type="number" step="any" required min="-90" max="90" placeholder="49.143845" class="coord-input" />
          </label>
          <label class="coord-label">
            Longitude
            <input v-model.number="form.longitude" type="number" step="any" required min="-180" max="180" placeholder="9.214797" class="coord-input" />
          </label>
        </div>
        <div class="actions">
          <OnyxButton type="submit" label="Create Sensor" color="primary" :loading="creating" :disabled="creating" />
        </div>
        <div v-if="createError" class="error-msg">
          <OnyxInfoCard color="danger">{{ createError }}</OnyxInfoCard>
        </div>
      </form>
    </OnyxCard>

    <!-- Stage 2: Auto-chained setup -->
    <OnyxCard v-else-if="setupPhase !== 'flash'">
      <OnyxHeadline is="h3">Setting Up Sensor</OnyxHeadline>
      <div class="setup-steps">

        <!-- Sensor created -->
        <div class="setup-row">
          <span class="step-icon step-done">✓</span>
          <div class="step-content">
            <span class="step-label">Sensor created — <code>{{ sensor.uuid }}</code></span>
          </div>
        </div>

        <!-- TTN registration -->
        <div class="setup-row">
          <span class="step-icon-wrap">
            <OnyxLoadingIndicator v-if="ttnStatus === 'running'" type="circle" />
            <span v-else-if="ttnStatus === 'done'" class="step-icon step-done">✓</span>
            <span v-else-if="ttnStatus === 'error'" class="step-icon step-error">✗</span>
            <span v-else class="step-icon step-pending">○</span>
          </span>
          <div class="step-content">
            <span class="step-label" :class="{ 'step-label-muted': ttnStatus === 'skipped' }">
              {{ ttnStatus === 'running' ? 'Registering with TTN…' : ttnStatus === 'skipped' ? 'TTN registration skipped' : 'TTN registration' }}
            </span>
            <div v-if="ttnStatus === 'done' && ttnData" class="ttn-keys">
              <div class="key-row">
                <span class="key-name">DevEUI</span>
                <code>{{ maskKey(ttnData.dev_eui) }}</code>
                <OnyxButton label="Copy" mode="outline" color="neutral" density="compact" @click="copyToClipboard(ttnData!.dev_eui)" />
              </div>
              <div class="key-row">
                <span class="key-name">AppKey</span>
                <code>{{ maskKey(ttnData.app_key) }}</code>
                <OnyxButton label="Copy" mode="outline" color="neutral" density="compact" @click="copyToClipboard(ttnData!.app_key)" />
              </div>
            </div>
            <div v-if="ttnStatus === 'error'" class="step-error-block">
              <span class="step-error-msg">{{ ttnError }}</span>
              <div class="step-retry-actions">
                <OnyxButton label="Retry TTN" color="primary" density="compact" @click="retryTTN" />
                <OnyxButton label="Skip TTN & build without LoRa" color="neutral" density="compact" @click="skipTTN" />
              </div>
            </div>
          </div>
        </div>

        <!-- Firmware build -->
        <div class="setup-row">
          <span class="step-icon-wrap">
            <OnyxLoadingIndicator v-if="buildStepStatus === 'running'" type="circle" />
            <span v-else-if="buildStepStatus === 'done'" class="step-icon step-done">✓</span>
            <span v-else-if="buildStepStatus === 'error'" class="step-icon step-error">✗</span>
            <span v-else class="step-icon step-pending">○</span>
          </span>
          <div class="step-content">
            <span class="step-label" :class="{ 'step-label-muted': buildStepStatus === 'pending' }">
              {{ buildStepStatus === 'running' ? 'Building firmware…' : 'Firmware build' }}
            </span>
            <div v-if="buildStepStatus === 'error'" class="step-error-block">
              <span class="step-error-msg">{{ buildMessage }}</span>
              <div class="step-retry-actions">
                <OnyxButton label="Retry Build" color="primary" density="compact" @click="retryBuild" />
              </div>
            </div>
          </div>
        </div>

      </div>
      <div class="actions">
        <OnyxButton v-if="setupPhase === 'done'" label="Flash Sensor" color="primary" @click="setupPhase = 'flash'" />
        <OnyxButton label="Back to sensor list" color="neutral" link="/" />
      </div>
    </OnyxCard>

    <!-- Stage 3: Flash & Provision -->
    <OnyxCard v-else>
      <OnyxHeadline is="h3">Flash & Provision Sensor</OnyxHeadline>
      <FlashAndProvisionStep
        :manifest-url="`/api/sensors/${sensor.uuid}/manifest.json`"
        :sensor-uuid="sensor.uuid"
        :dev-eui="ttnData?.dev_eui || ''"
        :app-key="ttnData?.app_key || ''"
        join-eui="0101010101010101"
      />
      <div class="actions">
        <OnyxButton label="Back to sensor list" color="neutral" link="/" />
      </div>
    </OnyxCard>
  </OnyxPageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import {
  OnyxHeadline,
  OnyxCard,
  OnyxInput,
  OnyxButton,
  OnyxLoadingIndicator,
  OnyxInfoCard, OnyxPageLayout,
} from 'sit-onyx'
import api from '../api'
import axios from 'axios'
import FlashAndProvisionStep from '../components/FlashAndProvisionStep.vue'
import MapPicker from '../components/MapPicker.vue'

const setupPhase = ref<'idle' | 'ttn' | 'building' | 'done' | 'flash'>('idle')
const ttnStatus = ref<'running' | 'done' | 'error' | 'skipped'>('running')
const ttnError = ref('')
const ttnData = ref<{ dev_eui: string; app_key: string; ttn_device_id: string } | null>(null)
const buildStepStatus = ref<'pending' | 'running' | 'done' | 'error'>('pending')
const buildMessage = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const DEFAULT_LAT = 49.1438453
const DEFAULT_LNG = 9.2147973

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
      name,
      latitude: form.value.latitude,
      longitude: form.value.longitude,
    })
    sensor.value = data
    setupPhase.value = 'ttn'
    creating.value = false
    await runTTN()
  } catch (e: unknown) {
    createError.value = e instanceof Error ? e.message : 'Sensor creation failed'
    creating.value = false
  }
}

async function runTTN() {
  ttnStatus.value = 'running'
  ttnError.value = ''
  ttnData.value = null
  try {
    const { data } = await api.post(`/api/sensors/${sensor.value.uuid}/register-ttn`)
    ttnData.value = data
    ttnStatus.value = 'done'
    await runBuild()
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.data?.error) {
      ttnError.value = e.response.data.error
    } else {
      ttnError.value = e instanceof Error ? e.message : 'Registration failed'
    }
    ttnStatus.value = 'error'
  }
}

async function retryTTN() {
  buildStepStatus.value = 'pending'
  await runTTN()
}

async function skipTTN() {
  ttnStatus.value = 'skipped'
  await runBuild()
}

async function runBuild(): Promise<void> {
  setupPhase.value = 'building'
  buildStepStatus.value = 'running'
  try {
    await api.post(`/api/sensors/${sensor.value.uuid}/build-firmware`)
  } catch {
    // ignore if already building
  }
  return new Promise<void>((resolve) => {
    if (pollTimer) clearInterval(pollTimer)
    pollTimer = setInterval(async () => {
      try {
        const { data } = await api.get(`/api/sensors/${sensor.value.uuid}/build-status`)
        if (data.status === 'done') {
          buildStepStatus.value = 'done'
          setupPhase.value = 'done'
          clearInterval(pollTimer!)
          pollTimer = null
          resolve()
        } else if (data.status === 'error') {
          buildStepStatus.value = 'error'
          buildMessage.value = data.message || 'Unknown error'
          clearInterval(pollTimer!)
          pollTimer = null
          resolve()
        }
      } catch {
        // ignore transient errors
      }
    }, 3000)
  })
}

async function retryBuild() {
  buildMessage.value = ''
  await runBuild()
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

.error-msg { margin-top: 0.75rem; }

.hint {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-size: var(--onyx-font-size-sm);
  margin: 0;
}

/* Setup stage */
.setup-steps {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 0.25rem 0;
}

.setup-row {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.step-icon-wrap {
  width: 1.5rem;
  height: 1.5rem;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.step-icon {
  font-size: 1rem;
  line-height: 1;
}

.step-done { color: var(--onyx-color-base-success-500, #16a34a); }
.step-error { color: var(--onyx-color-base-danger-500, #dc2626); }
.step-pending { color: var(--onyx-color-text-icons-neutral-soft, #9ca3af); }

.step-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.1rem;
}

.step-label {
  font-weight: 500;
  line-height: 1.5rem;
}

.step-label-muted {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-weight: 400;
}

.ttn-keys {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.875rem;
}

.key-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.key-name {
  width: 60px;
  font-weight: 600;
  color: var(--onyx-color-text-icons-neutral-medium);
  flex-shrink: 0;
}

.step-error-block {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.step-error-msg {
  color: var(--onyx-color-base-danger-600, #b91c1c);
  font-size: 0.875rem;
}

.step-retry-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
</style>
