<template>
  <div class="debugger">
    <div class="debugger__header">
      <h1 class="debugger__title">Serial Debugger</h1>
      <div class="debugger__controls">
        <span class="debugger__baud">115200 baud</span>
        <span class="status-badge" :class="`status-badge--${connectionState}`">
          <span class="status-badge__dot" />
          {{ statusLabel }}
        </span>
        <OnyxButton
          v-if="connectionState !== 'connected'"
          label="Connect"
          color="primary"
          size="sm"
          :loading="connectionState === 'connecting'"
          :disabled="connectionState === 'connecting'"
          @click="connect"
        />
        <OnyxButton
          v-else
          label="Disconnect"
          color="danger"
          size="sm"
          @click="disconnect"
        />
        <OnyxButton
          v-if="lines.length"
          label="Clear"
          color="neutral"
          mode="plain"
          size="sm"
          @click="lines = []"
        />
      </div>
    </div>

    <OnyxInfoCard v-if="!webSerialSupported" color="warning" headline="Web Serial not supported.">
      Use Chrome or Edge (desktop) to debug the sensor via USB.
    </OnyxInfoCard>

    <OnyxInfoCard
      v-else-if="errorMessage"
      color="danger"
      :headline="errorMessage"
    />

    <div v-else class="debugger__body">
      <SerialLogPane class="debugger__log" :lines="lines" />
      <StateDiagram
        class="debugger__diagram"
        :current-state="currentState"
        :error-lines="errorLines"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { OnyxButton, OnyxInfoCard } from 'sit-onyx'
import SerialLogPane from '../components/SerialLogPane.vue'
import StateDiagram from '../components/StateDiagram.vue'
import { useSerialPort } from '../composables/useSerialPort'
import { parseLine } from '../utils/serial-parser'
import type { LogLine, SensorState } from '../types/serial-debugger'

const { webSerialSupported, connectionState, errorMessage, connect, disconnect, onLine } = useSerialPort()

const lines = ref<LogLine[]>([])
const currentState = ref<SensorState>('IDLE')
let lineCounter = 0

const errorLines = computed(() => lines.value.filter(l => l.isError))

const statusLabel = computed(() => {
  switch (connectionState.value) {
    case 'connected':    return 'Connected'
    case 'connecting':   return 'Connecting…'
    case 'disconnected': return 'Disconnected'
    case 'error':        return 'Error'
  }
})

onLine((text) => {
  const { state, isError } = parseLine(text)
  lines.value.push({ id: lineCounter++, text, state, isError, timestamp: new Date() })
  if (state !== null) currentState.value = state
})
</script>

<style scoped>
.debugger {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  height: 100%;
  padding: 1.25rem 1.5rem;
  box-sizing: border-box;
}

.debugger__header {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  border-bottom: 1px solid var(--onyx-color-base-neutral-200, #e5e5e5);
  padding-bottom: 0.875rem;
}

.debugger__title {
  font-size: 1.125rem;
  font-weight: 700;
  margin: 0;
  flex: 1;
  white-space: nowrap;
}

.debugger__controls {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  flex-wrap: wrap;
}

.debugger__baud {
  font-size: 0.72rem;
  font-family: monospace;
  color: var(--onyx-color-base-neutral-500, #737373);
  background: var(--onyx-color-base-neutral-100, #f5f5f5);
  border: 1px solid var(--onyx-color-base-neutral-200, #e5e5e5);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.6rem;
  border-radius: 999px;
  background: var(--onyx-color-base-neutral-100, #f5f5f5);
  border: 1px solid var(--onyx-color-base-neutral-200, #e5e5e5);
  white-space: nowrap;
}

.status-badge__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--onyx-color-base-neutral-400, #a3a3a3);
  flex-shrink: 0;
}

.status-badge--connected    { background: #f0fdf4; border-color: #86efac; color: #15803d; }
.status-badge--connected .status-badge__dot { background: #22c55e; }
.status-badge--connecting   { background: #fefce8; border-color: #fde047; color: #a16207; }
.status-badge--connecting .status-badge__dot { background: #eab308; animation: blink 1s ease-in-out infinite; }
.status-badge--error        { background: #fef2f2; border-color: #fca5a5; color: #b91c1c; }
.status-badge--error .status-badge__dot { background: #ef4444; }

.debugger__body {
  display: grid;
  grid-template-columns: 1fr 240px;
  gap: 0.875rem;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

@media (max-width: 768px) {
  .debugger__body { grid-template-columns: 1fr; }
  .debugger__diagram { order: -1; max-height: 220px; overflow-y: auto; }
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.3; }
}
</style>
