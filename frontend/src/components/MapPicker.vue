<template>
  <div ref="mapEl" class="map-container"></div>
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
let map: L.Map | null = null
let marker: L.Marker | null = null

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
.map-container {
  width: 100%;
  height: 300px;
  border-radius: 6px;
  border: 1px solid #ccc;
  margin-bottom: 1rem;
}
</style>
