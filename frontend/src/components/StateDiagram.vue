<template>
  <div class="diagram">
    <div class="diagram__label">State Machine</div>
    <div class="diagram__nodes">
      <template v-for="(node, i) in nodes" :key="node.state">
        <div
          class="snode"
          :class="{
            'snode--active':  isNext(node.state),
            'snode--done':    isDone(node.state),
            'snode--error':   node.state === 'ERROR' && currentState === 'ERROR',
            'snode--idle':    node.state === 'IDLE' && currentState === 'IDLE',
          }"
          :data-state="node.state"
        >
          <div class="snode__icon">{{ node.icon }}</div>
          <div class="snode__info">
            <div class="snode__name">{{ node.state }}</div>
            <div class="snode__hint">{{ node.hint }}</div>
          </div>
          <div v-if="node.state === 'ERROR' && errorCount > 0" class="snode__badge">
            {{ errorCount }}
          </div>
          <div v-else-if="isDone(node.state)" class="snode__check">✓</div>
        </div>
        <!-- connector between nodes (not after last) -->
        <div v-if="i < nodes.length - 1" class="diagram__connector">
          <div class="diagram__connector-line" :class="{ 'diagram__connector-line--active': isConnectorActive(i) }" />
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SensorState, LogLine } from '../types/serial-debugger'

const props = defineProps<{
  currentState: SensorState
  errorLines?: LogLine[]
}>()

const ORDER: SensorState[] = ['IDLE', 'INIT', 'SCAN', 'SEND', 'RECEIVE', 'SLEEP', 'ERROR']

const nodes = [
  { state: 'IDLE'    as SensorState, icon: '○',  hint: 'Awaiting connection' },
  { state: 'INIT'    as SensorState, icon: '⚡',  hint: 'Battery:' },
  { state: 'SCAN'    as SensorState, icon: '📡', hint: 'PAX count:' },
  { state: 'SEND'    as SensorState, icon: '↑',  hint: 'Payload:' },
  { state: 'RECEIVE' as SensorState, icon: '↓',  hint: 'TX ok' },
  { state: 'SLEEP'   as SensorState, icon: '💤', hint: 'Sleeping for…' },
  { state: 'ERROR'   as SensorState, icon: '✕',  hint: 'TX error / Join failed' },
]

const errorCount = computed(() => props.errorLines?.length ?? 0)

const currentIdx = computed(() => ORDER.indexOf(props.currentState))

// Each log line = that state is done; so current state counts as done
function isDone(state: SensorState): boolean {
  if (props.currentState === 'IDLE' || props.currentState === 'ERROR') return false
  const nodeIdx = ORDER.indexOf(state)
  return nodeIdx <= currentIdx.value && state !== 'ERROR'
}

// Pulse the NEXT upcoming state, not the one that just logged
function isNext(state: SensorState): boolean {
  if (props.currentState === 'IDLE' || props.currentState === 'ERROR' || props.currentState === 'SLEEP') return false
  return ORDER[currentIdx.value + 1] === state && state !== 'ERROR'
}

function isConnectorActive(nodeIndex: number): boolean {
  return nodeIndex <= currentIdx.value && props.currentState !== 'IDLE' && props.currentState !== 'ERROR'
}
</script>

<style scoped>
.diagram {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  overflow-y: auto;
  height: 100%;
}

.diagram__label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--onyx-color-base-neutral-500, #737373);
  padding: 0 0.25rem;
}

.diagram__nodes {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.diagram__connector {
  display: flex;
  justify-content: center;
  padding: 0 1.125rem;
  height: 16px;
  align-items: center;
}

.diagram__connector-line {
  width: 2px;
  height: 100%;
  background: var(--onyx-color-base-neutral-200, #e5e5e5);
  border-radius: 1px;
  transition: background 0.3s;
}

.diagram__connector-line--active {
  background: var(--onyx-color-base-primary-400, #60a5fa);
}

.snode {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.5rem 0.625rem;
  border-radius: 8px;
  border: 1.5px solid var(--onyx-color-base-neutral-200, #e5e5e5);
  background: white;
  transition: all 0.2s ease;
  position: relative;
  cursor: default;
}

.snode--idle {
  background: #f9fafb;
  border-color: #e5e7eb;
}

.snode--done {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.snode--active {
  border-color: var(--onyx-color-base-primary-500, #3b82f6);
  background: #eff6ff;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
  opacity: 1;
  animation: pulse-border 2s ease-in-out infinite;
}

.snode--error {
  border-color: #ef4444;
  background: #fef2f2;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.15);
  animation: shake 0.4s ease-in-out;
}

.snode__icon {
  font-size: 1rem;
  width: 1.75rem;
  height: 1.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--onyx-color-base-neutral-100, #f5f5f5);
  flex-shrink: 0;
  font-style: normal;
}

.snode--active .snode__icon {
  background: #dbeafe;
}

.snode--error .snode__icon {
  background: #fee2e2;
  color: #ef4444;
}

.snode--done .snode__icon {
  background: #f0fdf4;
}

.snode__info {
  flex: 1;
  min-width: 0;
}

.snode__name {
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0.03em;
  line-height: 1.2;
  color: #111827;
}

.snode--active .snode__name {
  color: #1d4ed8;
}

.snode--error .snode__name {
  color: #b91c1c;
}

.snode__hint {
  font-size: 0.65rem;
  font-family: monospace;
  color: #6b7280;
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.snode__check {
  font-size: 0.75rem;
  color: #22c55e;
  font-weight: 700;
  flex-shrink: 0;
}

.snode__badge {
  background: #ef4444;
  color: white;
  border-radius: 999px;
  font-size: 0.65rem;
  font-weight: 700;
  padding: 0.1rem 0.35rem;
  min-width: 1.1rem;
  text-align: center;
  flex-shrink: 0;
}

@keyframes pulse-border {
  0%, 100% { box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15); }
  50%       { box-shadow: 0 0 0 5px rgba(59, 130, 246, 0.08); }
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20%       { transform: translateX(-3px); }
  40%       { transform: translateX(3px); }
  60%       { transform: translateX(-2px); }
  80%       { transform: translateX(1px); }
}
</style>
