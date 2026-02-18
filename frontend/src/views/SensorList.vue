<template>
  <div>
    <div class="page-header">
      <h2>Registered Sensors</h2>
      <button class="btn-primary" @click="refresh">↻ Refresh</button>
    </div>

    <div v-if="loading" class="status">Loading sensors…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>

    <table v-else-if="sensors.length" class="sensor-table">
      <thead>
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
      </thead>
      <tbody>
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
              <span
                v-if="s.ttn_device_id"
                class="badge-linked badge-linked-btn"
                @click="toggleTTNInfo(s.uuid)"
                :title="ttnInfoUUID === s.uuid ? 'Hide TTN info' : 'Show TTN info'"
              >Linked {{ ttnInfoUUID === s.uuid ? '▲' : '▼' }}</span>
              <template v-else>
                <span class="badge-unlinked">Not linked</span>
                <button
                  class="btn-link-ttn"
                  :disabled="linkingUUID === s.uuid"
                  @click="linkTTN(s.uuid)"
                >
                  {{ linkingUUID === s.uuid ? 'Linking…' : 'Link TTN' }}
                </button>
              </template>
            </td>
            <td>
              <span :class="['data-badge', lastDataClass(s.last_data_time)]">
                {{ formatAge(s.last_data_time) }}
              </span>
            </td>
            <td>
              <span v-if="s.last_battery_value !== null" :class="['battery-badge', batteryClass(s.last_battery_value!)]">
                {{ Math.round(s.last_battery_value!) }}%
              </span>
              <span v-else class="battery-unknown">—</span>
            </td>
            <td>{{ formatDate(s.created_at) }}</td>
            <td>
              <button
                class="btn-rebuild"
                :disabled="rebuildUUID === s.uuid"
                @click="startRebuild(s)"
              >
                {{ rebuildUUID === s.uuid ? 'Building…' : 'Rebuild' }}
              </button>
              <button class="btn-danger" @click="deleteSensor(s)">Delete</button>
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
                  <span class="spinner">⏳</span> Compiling firmware with PlatformIO…
                  <p class="hint">This typically takes 30–60 seconds.</p>
                  <button class="btn-secondary" @click="cancelRebuild">Close</button>
                </div>
                <!-- Done -->
                <div v-else-if="rebuildStatus === 'done' && !showFlash" class="rebuild-done">
                  <div class="status-done">Firmware compiled successfully!</div>
                  <div class="rebuild-actions">
                    <button class="btn-flash" @click="showFlash = true">Flash Sensor</button>
                    <button class="btn-secondary" @click="cancelRebuild">Close</button>
                  </div>
                </div>
                <!-- Flash -->
                <div v-else-if="rebuildStatus === 'done' && showFlash" class="rebuild-flash">
                  <p>Connect your ESP32 sensor via USB, then click <strong>Install</strong>.</p>
                  <p class="hint">Requires Chrome or Edge browser.</p>
                  <FlashStep :manifest-url="`/api/sensors/${s.uuid}/manifest.json`" />
                  <div class="rebuild-actions">
                    <button class="btn-secondary" @click="cancelRebuild">Close</button>
                  </div>
                </div>
                <!-- Error -->
                <div v-else-if="rebuildStatus === 'error'" class="rebuild-error">
                  <div class="error-msg">Build failed: {{ rebuildMessage }}</div>
                  <div class="rebuild-actions">
                    <button class="btn-rebuild" @click="startRebuild(s)">Retry</button>
                    <button class="btn-secondary" @click="cancelRebuild">Close</button>
                  </div>
                </div>
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>

    <div v-else class="empty">
      No sensors registered yet. <router-link to="/add">Add your first sensor →</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import api from '../api'
import axios from 'axios'
import FlashStep from '../components/FlashStep.vue'
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

function lastDataClass(iso: string | null): string {
  if (!iso) return 'data-never'
  const ageMs = Date.now() - new Date(iso).getTime()
  if (ageMs < 24 * 3600 * 1000) return 'data-fresh'
  if (ageMs < 7 * 24 * 3600 * 1000) return 'data-stale'
  return 'data-old'
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
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
.page-header h2 { margin: 0; }
.sensor-table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 4px rgba(0,0,0,.1); }
.sensor-table th { background: #1a1a2e; color: white; padding: .75rem 1rem; text-align: left; font-weight: 600; }
.sensor-table td { padding: .75rem 1rem; border-bottom: 1px solid #eee; }
.sensor-table tr:last-child td { border-bottom: none; }
.sensor-table tr:hover td { background: #f0f8ff; }
.bold { font-weight: 600; }
.mono { font-family: monospace; font-size: .875rem; }
.status { padding: 2rem; text-align: center; color: #666; }
.status.error { color: #c0392b; }
.empty { padding: 3rem; text-align: center; color: #666; background: white; border-radius: 8px; }
.btn-primary { background: #1a1a2e; color: white; border: none; padding: .5rem 1rem; border-radius: 6px; cursor: pointer; }
.btn-primary:hover { background: #2d2d5a; }
.btn-danger { background: #e74c3c; color: white; border: none; padding: .4rem .75rem; border-radius: 5px; cursor: pointer; font-size: .85rem; }
.btn-danger:hover { background: #c0392b; }
.badge-linked { background: #e8f8ef; color: #27ae60; padding: .2rem .6rem; border-radius: 4px; font-weight: 600; font-size: .85rem; white-space: nowrap; }
.badge-unlinked { background: #f0f0f0; color: #999; padding: .2rem .6rem; border-radius: 4px; font-size: .85rem; white-space: nowrap; }
.btn-link-ttn { background: none; border: 1px solid #1a1a2e; color: #1a1a2e; padding: .2rem .5rem; border-radius: 4px; cursor: pointer; font-size: .8rem; margin-left: .5rem; }
.btn-link-ttn:hover { background: #1a1a2e; color: white; }
.btn-link-ttn:disabled { opacity: .5; cursor: not-allowed; }

.btn-rebuild { background: none; border: 1px solid #e67e22; color: #e67e22; padding: .4rem .75rem; border-radius: 5px; cursor: pointer; font-size: .85rem; margin-right: .5rem; }
.btn-rebuild:hover { background: #e67e22; color: white; }
.btn-rebuild:disabled { opacity: .5; cursor: not-allowed; }

.rebuild-row td { padding: 0 !important; border-bottom: 1px solid #eee; }
.rebuild-row:hover td { background: transparent !important; }
.rebuild-panel { background: #f8f9fa; padding: 1.5rem; border-top: 2px solid #e67e22; }
.rebuild-building { display: flex; flex-direction: column; align-items: center; gap: .5rem; padding: 1rem 0; }
.spinner { font-size: 2rem; animation: spin 2s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.hint { color: #999; font-size: .85rem; margin: 0; }
.rebuild-done { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
.status-done { color: #27ae60; font-size: 1.1rem; font-weight: 600; }
.rebuild-actions { display: flex; gap: .75rem; margin-top: .5rem; }
.rebuild-flash { display: flex; flex-direction: column; align-items: center; gap: .5rem; }
.rebuild-flash p { margin: 0; }
.rebuild-error { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
.error-msg { color: #c0392b; padding: .75rem; background: #fdecea; border-radius: 6px; }
.btn-flash { background: #e67e22; color: white; border: none; padding: .5rem 1rem; border-radius: 6px; cursor: pointer; font-weight: 600; }
.btn-flash:hover { background: #d35400; }
.btn-secondary { background: #eee; color: #333; padding: .5rem 1rem; border-radius: 6px; border: none; cursor: pointer; }
.btn-secondary:hover { background: #ddd; }

.badge-linked-btn { cursor: pointer; user-select: none; }
.badge-linked-btn:hover { background: #c8f0d8; }

.ttn-info-row td { padding: 0 !important; border-bottom: 1px solid #eee; }
.ttn-info-row:hover td { background: transparent !important; }
.ttn-info-panel { background: #f0f8ff; padding: 1.25rem 1.5rem; border-top: 2px solid #27ae60; }
.ttn-info-grid { display: flex; flex-direction: column; gap: .75rem; }
.ttn-info-item { display: flex; align-items: baseline; gap: 1rem; }
.ttn-label { font-size: .8rem; font-weight: 600; color: #555; min-width: 7rem; text-transform: uppercase; letter-spacing: .04em; }
.ttn-value { font-size: .875rem; color: #1a1a2e; word-break: break-all; }

.battery-badge { padding: .2rem .5rem; border-radius: 4px; font-size: .85rem; font-weight: 600; white-space: nowrap; }
.battery-high { background: #e8f8ef; color: #27ae60; }
.battery-mid  { background: #fef9e7; color: #d4ac0d; }
.battery-low  { background: #fdecea; color: #c0392b; }
.battery-unknown { color: #bbb; font-size: .9rem; }

.data-badge { padding: .2rem .5rem; border-radius: 4px; font-size: .85rem; font-weight: 600; white-space: nowrap; }
.data-fresh  { background: #e8f8ef; color: #1e8449; }
.data-stale  { background: #fef9e7; color: #b7770d; }
.data-old    { background: #fdecea; color: #c0392b; }
.data-never  { background: #f0f0f0; color: #333; }

.coord-btn { cursor: pointer; color: #1a6eb5; text-decoration: underline; text-decoration-style: dotted; font-size: .875rem; white-space: nowrap; }
.coord-btn:hover { color: #0d4d8a; }

.map-row td { padding: 0 !important; border-bottom: 1px solid #eee; }
.map-row:hover td { background: transparent !important; }
.map-cell { border-top: 2px solid #1a6eb5; }
</style>
