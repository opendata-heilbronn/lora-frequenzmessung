<template>
  <div class="log-pane" ref="paneRef" @scroll="handleScroll">
    <div v-if="lines.length === 0" class="log-pane__empty">
      Waiting for serial data…
    </div>
    <div
      v-for="line in lines"
      :key="line.id"
      class="log-line"
      :class="{
        'log-line--error':    line.isError,
        'log-line--state':    line.state !== null && !line.isError,
      }"
    >
      <span class="log-line__ts">{{ formatTs(line.timestamp) }}</span>
      <span class="log-line__text">{{ line.text }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { LogLine } from '../types/serial-debugger'

const props = withDefaults(defineProps<{
  lines: LogLine[]
  autoScroll?: boolean
}>(), { autoScroll: true })

const paneRef = ref<HTMLElement | null>(null)
let userScrolled = false

function formatTs(d: Date): string {
  const h  = String(d.getHours()).padStart(2, '0')
  const m  = String(d.getMinutes()).padStart(2, '0')
  const s  = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${h}:${m}:${s}.${ms}`
}

watch(
  () => props.lines.length,
  async () => {
    if (!props.autoScroll || userScrolled) return
    await nextTick()
    if (paneRef.value) paneRef.value.scrollTop = paneRef.value.scrollHeight
  },
)

function handleScroll() {
  if (!paneRef.value) return
  const { scrollTop, scrollHeight, clientHeight } = paneRef.value
  userScrolled = scrollTop + clientHeight < scrollHeight - 16
}
</script>

<style scoped>
.log-pane {
  font-family: 'Source Code Pro', 'Fira Mono', ui-monospace, monospace;
  font-size: 0.78rem;
  line-height: 1.65;
  overflow-y: auto;
  background: #1e1e2e;
  color: #cdd6f4;
  border-radius: 6px;
  padding: 0.5rem 0;
  height: 100%;
  box-sizing: border-box;
}

.log-pane__empty {
  padding: 1rem 0.875rem;
  color: #6c7086;
  font-style: italic;
}

.log-line {
  display: flex;
  gap: 0;
  padding: 0.05rem 0.875rem;
  border-left: 3px solid transparent;
}

.log-line:hover {
  background: rgba(205, 214, 244, 0.05);
}

.log-line--state {
  border-left-color: #89b4fa;
  background: rgba(137, 180, 250, 0.07);
}

.log-line--error {
  border-left-color: #f38ba8;
  background: rgba(243, 139, 168, 0.1);
  color: #f38ba8;
  font-weight: 600;
}

.log-line__ts {
  color: #6c7086;
  min-width: 7.5rem;
  flex-shrink: 0;
  user-select: none;
  font-size: 0.72rem;
  padding-top: 1px;
}

.log-line--state .log-line__ts { color: #89b4fa; }
.log-line--error .log-line__ts { color: #f38ba8; }

.log-line__text {
  word-break: break-all;
  flex: 1;
}
</style>
