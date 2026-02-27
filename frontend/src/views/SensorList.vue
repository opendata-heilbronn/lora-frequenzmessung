<template>
  <div>
    <div v-if="loading" class="status">
      <OnyxLoadingIndicator type="circle" />
      <span>Loading sensors…</span>
    </div>
    <div v-else-if="error" class="status error">
      <OnyxInfoCard color="danger">{{ error }}</OnyxInfoCard>
    </div>

    <OnyxTable v-else-if="sensors.length">
      <template #headline>
        <OnyxHeadline is="h2">Registered Sensors</OnyxHeadline>
      </template>
      <template #actions>
        <OnyxButton label="↻ Refresh" color="primary" @click="refresh" />
      </template>
      <template #head>
        <tr>
          <th>Name</th>
          <th>UUID</th>
          <th>Coordinates</th>
          <th>Type</th>
          <th>TTN</th>
          <th>Last Data</th>
          <th>Battery</th>
          <th>Added</th>
          <th></th>
        </tr>
      </template>

      <template v-for="s in sensors" :key="s.uuid">
        <tr>
          <td class="bold">{{ s.name }}</td>
          <td class="mono">{{ s.uuid }}</td>
          <td>
            <span
              class="coord-btn"
              @click="toggleMap(s.uuid)"
              :title="mapUUID === s.uuid ? 'Hide map' : 'Show on map'"
            >{{ s.latitude.toFixed(6) }}, {{ s.longitude.toFixed(6) }}</span>
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
              <OnyxButton
                label="Delete"
                color="danger"
                density="compact"
                @click="deleteSensor(s)"
              />
            </div>
          </td>
        </tr>
        <tr v-if="mapUUID === s.uuid" class="map-row">
          <td colspan="9" class="map-cell">
            <SensorMap :lat="s.latitude" :lng="s.longitude" :name="s.name" />
          </td>
        </tr>
        <tr v-if="ttnInfoUUID === s.uuid" class="ttn-info-row">
          <td colspan="9">
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
          <td colspan="9">
            <div class="rebuild-panel">
              <!-- Building -->
              <div v-if="rebuildStatus === 'building'" class="rebuild-building">
                <OnyxLoadingIndicator type="circle" /> Compiling firmware with PlatformIO…
                <p class="hint">This typically takes 30–60 seconds.</p>
                <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
              </div>
              <!-- Done -->
              <div v-else-if="rebuildStatus === 'done' && !showFlash && !showProvision" class="rebuild-done">
                <OnyxInfoCard color="success">Firmware compiled successfully!</OnyxInfoCard>
                <div class="rebuild-actions">
                  <OnyxButton label="Flash Sensor" color="primary" @click="showFlash = true; showProvision = false" />
                  <OnyxButton label="Provision" mode="outline" color="neutral" @click="showProvision = true; showFlash = false" />
                  <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                </div>
              </div>
              <!-- Flash -->
              <div v-else-if="rebuildStatus === 'done' && showFlash" class="rebuild-flash">
                <p>Connect your ESP32 sensor via USB, then click <strong>Install</strong>.</p>
                <p class="hint">Requires Chrome or Edge browser.</p>
                <FlashStep :manifest-url="`/api/sensors/${s.uuid}/manifest.json`" />
                <div class="rebuild-actions">
                  <OnyxButton label="Close" color="neutral" @click="cancelRebuild" />
                </div>
              </div>
              <!-- Provision -->
              <div v-else-if="rebuildStatus === 'done' && showProvision" class="rebuild-provision">
                <ProvisionStep
                  :sensor-uuid="s.uuid"
                  :dev-eui="s.dev_eui || ''"
                  :app-key="s.app_key || ''"
                  :join-eui="JOIN_EUI"
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

      <template #empty>
        <OnyxEmpty label="No sensors registered yet." />
      </template>
    </OnyxTable>

    <div v-else class="empty">
      <p>No sensors registered yet.</p>
      <router-link to="/add">Add your first sensor →</router-link>
    </div>
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
import FlashStep from '../components/FlashStep.vue'
import ProvisionStep from '../components/ProvisionStep.vue'
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
}

const sensors = ref<Sensor[]>([])
const loading = ref(false)
const error = ref('')
const linkingUUID = ref('')

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
const showFlash = ref(false)
const showProvision = ref(false)
const JOIN_EUI = '0101010101010101'
let rebuildPollTimer: ReturnType<typeof setInterval> | null = null

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<Sensor[]>('/api/sensors')
    sensors.value = data
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load sensors'
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
    alert(msg)
  } finally {
    linkingUUID.value = ''
  }
}

async function deleteSensor(s: Sensor) {
  const message = s.ttn_device_id
    ? `Delete sensor "${s.name}"? This will also remove it from TTN.`
    : `Delete sensor "${s.name}"?`
  if (!confirm(message)) return
  try {
    await api.delete(`/api/sensors/${s.uuid}`)
    sensors.value = sensors.value.filter(x => x.uuid !== s.uuid)
  } catch (e: unknown) {
    alert(e instanceof Error ? e.message : 'Delete failed')
  }
}

async function startRebuild(sensor: Sensor) {
  cancelRebuild()
  rebuildUUID.value = sensor.uuid
  rebuildStatus.value = 'building'
  rebuildMessage.value = ''
  showFlash.value = false
  showProvision.value = false
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
  showFlash.value = false
  showProvision.value = false
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

.empty {
  padding: 3rem;
  text-align: center;
}

.bold { font-weight: 600; }
.mono { font-family: monospace; font-size: 0.875rem; }

.actions-cell {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.coord-btn {
  cursor: pointer;
  color: var(--onyx-color-base-primary-500);
  text-decoration: underline;
  text-decoration-style: dotted;
  font-size: 0.875rem;
  white-space: nowrap;
}
.coord-btn:hover { opacity: 0.8; }

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
.rebuild-flash { display: flex; flex-direction: column; align-items: center; gap: 0.5rem; }
.rebuild-flash p { margin: 0; }
.rebuild-error { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
.error-msg { width: 100%; }

/* Map cell */
.map-cell { padding: 0; }
</style>
