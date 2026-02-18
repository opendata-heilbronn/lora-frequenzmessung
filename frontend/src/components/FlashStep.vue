<template>
  <div class="flash-wrap">
    <!-- Import esp-web-tools as a side-effect to register the custom element -->
    <div v-if="!webSerialSupported" class="no-serial">
      <strong>Web Serial not supported.</strong>
      Use Chrome or Edge (desktop) to flash the sensor via USB.
    </div>
    <div v-else class="flash-btn-wrap">
      <esp-web-install-button :manifest="manifestUrl">
        <button slot="activate" class="flash-button">⚡ Connect & Flash Sensor</button>
        <span slot="unsupported">Your browser does not support Web Serial. Please use Chrome or Edge.</span>
      </esp-web-install-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

// Register the web component
import 'esp-web-tools'

defineProps<{
  manifestUrl: string
}>()

const webSerialSupported = ref(false)

onMounted(() => {
  webSerialSupported.value = 'serial' in navigator
})
</script>

<style scoped>
.flash-wrap { margin-top: 1.5rem; }
.no-serial { background: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; padding: 1rem; color: #856404; }
.flash-btn-wrap { display: flex; align-items: center; gap: 1rem; }
.flash-button {
  background: #e67e22;
  color: white;
  border: none;
  padding: .75rem 1.5rem;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1.05rem;
  font-weight: 600;
}
.flash-button:hover { background: #d35400; }
</style>
