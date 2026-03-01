<template>
  <div class="map-wrapper">
    <div ref="mapEl" class="map-container"></div>
    <button
      class="locate-btn"
      type="button"
      :disabled="locating"
      :aria-label="locateError || 'Use my current location'"
      :title="locateError || 'Use my current location'"
      @click="locateMe"
    >
      <span v-if="locating" class="locate-spinner"></span>
      <span v-else>⊕</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

// Fix Leaflet's broken default icon paths under Vite
// eslint-disable-next-line @typescript-eslint/no-explicit-any
delete (L.Icon.Default.prototype as any)._getIconUrl
L.Icon.Default.mergeOptions({
  iconUrl: new URL('leaflet/dist/images/marker-icon.png', import.meta.url).href,
  iconRetinaUrl: new URL('leaflet/dist/images/marker-icon-2x.png', import.meta.url).href,
  shadowUrl: new URL('leaflet/dist/images/marker-shadow.png', import.meta.url).href,
})

const props = defineProps<{
  modelValue: { lat: number; lng: number }
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: { lat: number; lng: number }): void
}>()

const mapEl = ref<HTMLElement | null>(null)
const locating = ref(false)
const locateError = ref('')
let map: L.Map | null = null
let marker: L.Marker | null = null

function locateMe() {
  if (!navigator.geolocation) {
    locateError.value = 'Geolocation not supported by your browser'
    return
  }
  locating.value = true
  locateError.value = ''
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      locating.value = false
      const { latitude, longitude } = pos.coords
      marker!.setLatLng([latitude, longitude])
      map!.setView([latitude, longitude], 15)
      emit('update:modelValue', { lat: latitude, lng: longitude })
    },
    () => {
      locating.value = false
      locateError.value = 'Location access denied'
    },
  )
}

onMounted(() => {
  if (!mapEl.value) return

  map = L.map(mapEl.value).setView([props.modelValue.lat, props.modelValue.lng], 15)

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    maxZoom: 19,
  }).addTo(map)

  marker = L.marker([props.modelValue.lat, props.modelValue.lng], { draggable: true }).addTo(map)

  marker.on('dragend', () => {
    const pos = marker!.getLatLng()
    emit('update:modelValue', { lat: pos.lat, lng: pos.lng })
  })

  map.on('click', (e: L.LeafletMouseEvent) => {
    marker!.setLatLng(e.latlng)
    emit('update:modelValue', { lat: e.latlng.lat, lng: e.latlng.lng })
  })
})

// Sync map marker when parent inputs change
watch(
  () => props.modelValue,
  (val) => {
    if (!map || !marker) return
    const current = marker.getLatLng()
    if (Math.abs(current.lat - val.lat) > 0.000001 || Math.abs(current.lng - val.lng) > 0.000001) {
      marker.setLatLng([val.lat, val.lng])
      map.panTo([val.lat, val.lng])
    }
  },
  { deep: true },
)

onUnmounted(() => {
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<style scoped>
.map-wrapper {
  position: relative;
  margin-bottom: 1rem;
}

.map-container {
  width: 100%;
  height: 220px;
  border-radius: var(--onyx-radius-md);
  border: 1px solid var(--onyx-color-base-neutral-300);
}

.locate-btn {
  position: absolute;
  bottom: 0.5rem;
  left: 0.5rem;
  z-index: 1000;
  background: white;
  border: 2px solid rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 1.1rem;
  line-height: 1;
  padding: 0;
  color: #333;
}

.locate-btn:disabled {
  cursor: wait;
  opacity: 0.7;
}

.locate-btn:hover:not(:disabled) {
  background: #f4f4f4;
}

.locate-spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid #ccc;
  border-top-color: #555;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
