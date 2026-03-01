<template>
  <div>
    <div v-if="loading" class="status">
      <OnyxLoadingIndicator type="circle" />
      <span>Loading sensors…</span>
    </div>
    <div v-else-if="loadError" class="status error">
      <OnyxInfoCard color="danger">Failed to load sensors. Please try again.</OnyxInfoCard>
      <OnyxButton label="Retry" color="neutral" mode="outline" @click="refresh" />
    </div>

    <template v-else>
      <div v-if="confirmAllVersions" class="confirm-banner">
        <OnyxInfoCard color="warning">
          Send a version-request downlink to ALL linked sensors? This uses LoRa duty cycle on every device.
        </OnyxInfoCard>
        <div class="confirm-banner-actions">
          <OnyxButton label="Yes, send to all" color="danger" @click="confirmAndRequestAllVersions" />
          <OnyxButton label="Cancel" color="neutral" mode="outline" @click="confirmAllVersions = false" />
        </div>
      </div>
      <div v-if="pageError" class="page-error">
        <OnyxInfoCard color="danger">{{ pageError }}</OnyxInfoCard>
      </div>
      <div v-if="pageSuccess" class="page-success">
        <OnyxInfoCard color="success">{{ pageSuccess }}</OnyxInfoCard>
      </div>

      <div v-if="sensors.length" class="table-wrapper">
        <OnyxTable>
          <template #headline>
            <OnyxHeadline is="h2">Registered Sensors</OnyxHeadline>
          </template>
          <template #actions>
            <OnyxButton label="Request All Versions" color="neutral" mode="outline" @click="confirmAllVersions = true" />
            <OnyxButton label="Refresh" color="primary" @click="refresh" />
          </template>
          <template #head>
            <tr>
              <th>Name</th>
              <th>UUID</th>
              <th>Map</th>
              <th>Type</th>
              <th>TTN</th>
              <th>Last Data</th>
              <th>Battery</th>
              <th>Firmware</th>
              <th>Added</th>
              <th></th>
            </tr>
          </template>

          <template v-for="s in sensors" :key="s.uuid">
            <tr>
              <td class="bold">{{ s.name }}</td>
              <td class="mono">{{ s.uuid }}</td>
              <td>
                <button
                  class="map-pin-btn"
                  @click="toggleMap(s.uuid)"
                  :title="`${s.latitude.toFixed(6)}, ${s.longitude.toFixed(6)}`"
                  :aria-label="mapUUID === s.uuid ? 'Hide map' : 'Show on map'"
                >📍</button>
              </td>
              <td>{{ s.type }}</td>
              <td>
                <div class="actions-cell" v-if="s.ttn_device_id">
                  <OnyxBadge
                    :class="['badge-linked']"
                    color="success"
                    :clickable="ttnInfoUUID === s.uuid ? 'Hide TTN info' : 'Show TTN info'"
                    @click="toggleTTNInfo(s.uuid)"
                  >Linked {{ ttnInfoUUID === s.uuid ? '▲' : '▼' }}</OnyxBadge>
                </div>
                <div class="actions-cell" v-else>
                  <OnyxBadge :class="['badge-unlinked']" color="neutral">Not linked</OnyxBadge>
                  <OnyxButton
                    label="Link TTN"
                    mode="outline"
                    color="neutral"
                    density="compact"
                    :disabled="linkingUUID === s.uuid"
                    :loading="linkingUUID === s.uuid"
                    @click="linkTTN(s.uuid)"
                  />
                </div>
              </td>
              <td>
                <OnyxBadge
                  :class="['data-badge', lastDataClass(s.last_data_time)]"
                  :color="lastDataColor(s.last_data_time)"
                >{{ formatAge(s.last_data_time) }}</OnyxBadge>
              </td>
              <td>
                <OnyxBadge
                  v-if="s.last_battery_value !== null"
                  :class="['battery-badge', batteryClass(s.last_battery_value!)]"
                  :color="batteryColor(s.last_battery_value!)"
                >{{ Math.round(s.last_battery_value!) }}%</OnyxBadge>
                <span v-else class="battery-unknown">—</span>
              </td>
              <td>
                <div class="actions-cell">
                  <OnyxBadge
                    :class="['firmware-badge']"
                    color="neutral"
                    :title="s.firmware_version_time ? `Reported: ${formatDate(s.firmware_version_time)}` : undefined"
                  >{{ s.firmware_version ?? 'Unknown' }}</OnyxBadge>
                  <button
                    class="version-refresh-btn"
                    :title="'Request version update'"
                    :disabled="versionRequestUUID === s.uuid"
                    @click="requestVersion(s.uuid)"
                  >↻</button>
                </div>
              </td>
              <td>{{ formatDate(s.created_at) }}</td>
              <td>
                <div class="actions-cell">
                  <OnyxButton
                    :label="rebuildUUID === s.uuid ? 'Building…' : 'Rebuild'"
                    mode="outline"
                    color="neutral"
                    density="compact"
                    :disabled="rebuildUUID === s.uuid"
                    :loading="rebuildUUID === s.uuid"
                    @click="startRebuild(s)"
                  />
                  <span class="action-separator"></span>
                  <template v-if="deleteConfirmUUID === s.uuid">
                    <span class="delete-confirm-text">Delete?</span>
                    <OnyxButton label="Yes" color="danger" density="compact" @click="confirmDelete(s)" />
                    <OnyxButton label="Cancel" color="neutral" density="compact" @click="deleteConfirmUUID = ''" />
                  </template>
                  <OnyxButton
                    v-else
                    label="Delete"
                    color="danger"
                    density="compact"
                    @click="deleteConfirmUUID = s.uuid"
                  />
                </div>
              </td>
            </tr>
            <tr v-if="mapUUID === s.uuid" class="map-row">
              <td colspan="10" class="map-cell">
                <SensorMap :lat="s.latitude" :lng="s.longitude" :name="s.name" />
              </td>
            </tr>
            <tr v-if="ttnInfoUUID === s.uuid" class="ttn-info-row">
              <td colspan="10">
                <div class="ttn-info-panel">
                  <div class="ttn-info-grid">
                    <div class="ttn-info-item">
                      <span class="ttn-label">Device ID</span>
                      <span class="ttn-value mono">{{ s.ttn_device_id }}</span>
                    </div>
                    <div class="ttn-info-item">
                      <span class="ttn-label">DevEUI</span>
                      <span class="ttn-value mono">{{ s.dev_eui }}</span>
                    </div>
                    <div class="ttn-info-item">
                      <span class="ttn-label">AppKey</span>
                      <span class="ttn-value mono">{{ s.app_key }}</span>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-if="rebuildUUID === s.uuid" class="rebuild-row">
              <td colspan="10">
                <div class="rebuild-panel">
                  <!-- Building -->
                  <div v-if="rebuildStatus === 'building'" class="rebuild-building">
                    <OnyxLoadingIndicator type="circle" /> Downloading firmware…
                    <p class="hint">This should only take a moment.</p>
                    <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                  </div>
                  <!-- Done -->
                  <div v-else-if="rebuildStatus === 'done' && !showFlashAndProvision" class="rebuild-done">
                    <OnyxInfoCard color="success">Firmware compiled successfully!</OnyxInfoCard>
                    <div class="rebuild-actions">
                      <OnyxButton label="Flash & Configure" color="primary" @click="showFlashAndProvision = true" />
                      <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                    </div>
                  </div>
                  <!-- Flash & Provision -->
                  <div v-else-if="rebuildStatus === 'done' && showFlashAndProvision" class="rebuild-flash-provision">
                    <FlashAndProvisionStep
                      :manifest-url="`/api/sensors/${s.uuid}/manifest.json`"
                      :sensor-uuid="s.uuid"
                      :dev-eui="s.dev_eui || ''"
                      :app-key="s.app_key || ''"
                      join-eui="0101010101010101"
                    />
                    <div class="rebuild-actions">
                      <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                    </div>
                  </div>
                  <!-- Error -->
                  <div v-else-if="rebuildStatus === 'error'" class="rebuild-error">
                    <div class="error-msg">
                      <OnyxInfoCard color="danger">Build failed: {{ rebuildMessage }}</OnyxInfoCard>
                    </div>
                    <div class="rebuild-actions">
                      <OnyxButton label="Retry" mode="outline" color="neutral" @click="startRebuild(s)" />
                      <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </OnyxTable>
      </div>

      <div v-else class="empty">
        <p>No sensors registered yet.</p>
        <OnyxButton label="Add your first sensor" color="primary" link="/add" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  OnyxTable,
  OnyxButton,
  OnyxBadge,
  OnyxHeadline,
  OnyxLoadingIndicator,
  OnyxInfoCard,
} from 'sit-onyx'
import api from '../api'
import axios from 'axios'
import FlashAndProvisionStep from '../components/FlashAndProvisionStep.vue'
import SensorMap from '../components/SensorMap.vue'

interface Sensor {
  id: number
  uuid: string
  name: string
  longitude: number
  latitude: number
  type: string
  dev_eui: string
  app_key: string
  ttn_device_id: string
  created_at: string
  last_battery_value: number | null
  last_battery_time: string | null
  last_data_time: string | null
  firmware_version: string | null
  firmware_version_time: string | null
}

const sensors = ref<Sensor[]>([])
const loading = ref(false)
const loadError = ref(false)
const linkingUUID = ref('')
const versionRequestUUID = ref('')

// Inline delete confirmation
const deleteConfirmUUID = ref('')

// Page-level action error (TTN, delete) — auto-clears after 6s
const pageError = ref('')
let errorTimer: ReturnType<typeof setTimeout> | null = null

function showError(msg: string) {
  pageError.value = msg
  if (errorTimer) clearTimeout(errorTimer)
  errorTimer = setTimeout(() => { pageError.value = '' }, 6000)
}

// Page-level success toast — auto-clears after 4s
const pageSuccess = ref('')
let successTimer: ReturnType<typeof setTimeout> | null = null

function showSuccess(msg: string) {
  pageSuccess.value = msg
  if (successTimer) clearTimeout(successTimer)
  successTimer = setTimeout(() => { pageSuccess.value = '' }, 4000)
}

const confirmAllVersions = ref(false)

const ttnInfoUUID = ref('')
const mapUUID = ref('')

function toggleTTNInfo(uuid: string) {
  ttnInfoUUID.value = ttnInfoUUID.value === uuid ? '' : uuid
}

function toggleMap(uuid: string) {
  mapUUID.value = mapUUID.value === uuid ? '' : uuid
}

const rebuildUUID = ref('')
const rebuildStatus = ref<'building' | 'done' | 'error'>('building')
const rebuildMessage = ref('')
const showFlashAndProvision = ref(false)
let rebuildPollTimer: ReturnType<typeof setInterval> | null = null

async function refresh() {
  loading.value = true
  loadError.value = false
  try {
    const { data } = await api.get<Sensor[]>('/api/sensors')
    sensors.value = data
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

async function linkTTN(uuid: string) {
  linkingUUID.value = uuid
  try {
    const { data } = await api.post(`/api/sensors/${uuid}/register-ttn`)
    const sensor = sensors.value.find(s => s.uuid === uuid)
    if (sensor) {
      sensor.dev_eui = data.dev_eui
      sensor.app_key = data.app_key
      sensor.ttn_device_id = data.ttn_device_id
    }
  } catch (e: unknown) {
    const msg = axios.isAxiosError(e) && e.response?.data?.error
      ? e.response.data.error
      : 'TTN registration failed'
    showError(msg)
  } finally {
    linkingUUID.value = ''
  }
}

async function confirmDelete(s: Sensor) {
  deleteConfirmUUID.value = ''
  try {
    await api.delete(`/api/sensors/${s.uuid}`)
    sensors.value = sensors.value.filter(x => x.uuid !== s.uuid)
  } catch (e: unknown) {
    showError(e instanceof Error ? e.message : 'Delete failed')
  }
}

async function requestVersion(uuid: string) {
  versionRequestUUID.value = uuid
  try {
    await api.post(`/api/sensors/${uuid}/request-version`)
    showSuccess('Version request queued — sensor will report on next wake')
  } catch (e: unknown) {
    const msg = axios.isAxiosError(e) && e.response?.data?.error
      ? e.response.data.error
      : 'Version request failed'
    showError(msg)
  } finally {
    versionRequestUUID.value = ''
  }
}

async function confirmAndRequestAllVersions() {
  confirmAllVersions.value = false
  try {
    await api.post('/api/sensors/request-all-versions')
    showSuccess('Version request queued for all linked sensors')
  } catch (e: unknown) {
    const msg = axios.isAxiosError(e) && e.response?.data?.error
      ? e.response.data.error
      : 'Bulk version request failed'
    showError(msg)
  }
}

async function startRebuild(sensor: Sensor) {
  cancelRebuild()
  rebuildUUID.value = sensor.uuid
  rebuildStatus.value = 'building'
  rebuildMessage.value = ''
  showFlashAndProvision.value = false
  try {
    await api.post(`/api/sensors/${sensor.uuid}/build-firmware`)
  } catch {
    // If already building, continue polling
  }
  pollRebuildStatus(sensor.uuid)
}

function pollRebuildStatus(uuid: string) {
  if (rebuildPollTimer) clearInterval(rebuildPollTimer)
  rebuildPollTimer = setInterval(async () => {
    try {
      const { data } = await api.get(`/api/sensors/${uuid}/build-status`)
      if (data.status === 'done') {
        rebuildStatus.value = 'done'
        clearInterval(rebuildPollTimer!)
        rebuildPollTimer = null
      } else if (data.status === 'error') {
        rebuildStatus.value = 'error'
        rebuildMessage.value = data.message || 'Unknown error'
        clearInterval(rebuildPollTimer!)
        rebuildPollTimer = null
      }
    } catch {
      // ignore transient errors
    }
  }, 3000)
}

function cancelRebuild() {
  rebuildUUID.value = ''
  rebuildStatus.value = 'building'
  rebuildMessage.value = ''
  showFlashAndProvision.value = false
  if (rebuildPollTimer) {
    clearInterval(rebuildPollTimer)
    rebuildPollTimer = null
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString()
}

function batteryClass(pct: number): string {
  if (pct >= 60) return 'battery-high'
  if (pct >= 25) return 'battery-mid'
  return 'battery-low'
}

function batteryColor(pct: number): 'success' | 'warning' | 'danger' {
  if (pct >= 60) return 'success'
  if (pct >= 25) return 'warning'
  return 'danger'
}

function lastDataClass(iso: string | null): string {
  if (!iso) return 'data-never'
  const ageMs = Date.now() - new Date(iso).getTime()
  if (ageMs < 24 * 3600 * 1000) return 'data-fresh'
  if (ageMs < 7 * 24 * 3600 * 1000) return 'data-stale'
  return 'data-old'
}

function lastDataColor(iso: string | null): 'success' | 'warning' | 'danger' | 'neutral' {
  if (!iso) return 'neutral'
  const ageMs = Date.now() - new Date(iso).getTime()
  if (ageMs < 24 * 3600 * 1000) return 'success'
  if (ageMs < 7 * 24 * 3600 * 1000) return 'warning'
  return 'danger'
}

function formatAge(iso: string | null): string {
  if (!iso) return 'Never'
  const ageMs = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(ageMs / 60000)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(ageMs / 3600000)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(ageMs / 86400000)
  return `${days}d ago`
}

onMounted(refresh)
onUnmounted(() => {
  if (rebuildPollTimer) {
    clearInterval(rebuildPollTimer)
    rebuildPollTimer = null
  }
  if (errorTimer) {
    clearTimeout(errorTimer)
    errorTimer = null
  }
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
})
</script>

<style scoped>
.status {
  padding: 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  color: var(--onyx-color-text-icons-neutral-medium);
}

.confirm-banner {
  margin-bottom: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.confirm-banner-actions {
  display: flex;
  gap: 0.75rem;
}

.page-error {
  margin-bottom: 1rem;
}

.page-success {
  margin-bottom: 1rem;
}

.version-refresh-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  color: var(--onyx-color-base-primary-500);
  padding: 0 0.25rem;
  line-height: 1;
}
.version-refresh-btn:hover { opacity: 0.7; }
.version-refresh-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.table-wrapper {
  overflow-x: auto;
}

.empty {
  padding: 3rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.bold { font-weight: 600; }
.mono { font-family: monospace; font-size: 0.875rem; }

.actions-cell {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.action-separator {
  width: 1px;
  height: 1.25rem;
  background: var(--onyx-color-base-neutral-300);
  flex-shrink: 0;
}

.delete-confirm-text {
  font-size: 0.8rem;
  color: var(--onyx-color-text-icons-neutral-medium);
  white-space: nowrap;
}

.map-pin-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.1rem;
  padding: 0.1rem 0.2rem;
  line-height: 1;
  border-radius: 4px;
}
.map-pin-btn:hover { background: var(--onyx-color-base-neutral-200); }

.battery-unknown {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-size: 0.9rem;
}

/* Detail row resets */
.ttn-info-row td,
.rebuild-row td,
.map-row td {
  padding: 0 !important;
}
.ttn-info-row:hover td,
.rebuild-row:hover td,
.map-row:hover td {
  background: transparent !important;
}

/* TTN info panel */
.ttn-info-panel {
  padding: 1.25rem 1.5rem;
  background: var(--onyx-color-base-neutral-100);
}
.ttn-info-grid { display: flex; flex-direction: column; gap: 0.75rem; }
.ttn-info-item { display: flex; align-items: baseline; gap: 1rem; }
.ttn-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--onyx-color-text-icons-neutral-medium);
  min-width: 7rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.ttn-value { font-size: 0.875rem; word-break: break-all; }

/* Rebuild panel */
.rebuild-panel {
  padding: 1.5rem;
  background: var(--onyx-color-base-neutral-100);
}
.rebuild-building {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 0;
}
.hint {
  color: var(--onyx-color-text-icons-neutral-soft);
  font-size: var(--onyx-font-size-sm);
  margin: 0;
}
.rebuild-done { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
.rebuild-actions { display: flex; gap: 0.75rem; margin-top: 0.5rem; }
.rebuild-flash-provision { display: flex; flex-direction: column; gap: 0.5rem; }
.rebuild-error { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
.error-msg { width: 100%; }

/* Map cell */
.map-cell { padding: 0; }
</style>
