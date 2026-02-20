<template>
  <div class="flash-wrap">
    <!-- Import esp-web-tools as a side-effect to register the custom element -->
    <OnyxInfoCard v-if="!webSerialSupported" color="warning" headline="Web Serial not supported.">
      Use Chrome or Edge (desktop) to flash the sensor via USB.
    </OnyxInfoCard>
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
import { OnyxInfoCard } from 'sit-onyx'

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
.flash-btn-wrap { display: flex; align-items: center; gap: 1rem; }
.flash-button {
  font-family: var(--onyx-font-family-paragraph);
  font-size: var(--onyx-font-size-md);
  font-weight: 600;
  background: var(--onyx-color-base-primary-500);
  color: var(--onyx-color-text-icons-neutral-inverted);
  border: none;
  padding: var(--onyx-density-sm) var(--onyx-density-lg);
  border-radius: var(--onyx-radius-md);
  cursor: pointer;
}
.flash-button:hover {
  background: var(--onyx-color-base-primary-600);
}
</style>
