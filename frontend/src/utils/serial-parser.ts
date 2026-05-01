import type { SensorState } from '../types/serial-debugger'

interface ParseResult {
  state: SensorState | null
  isError: boolean
}

const STATE_PATTERNS: Array<{ re: RegExp; state: SensorState; isError?: boolean; label: string }> = [
  { re: /^Battery:/,          state: 'INIT',    label: 'Battery:' },
  { re: /^PAX count:/,        state: 'SCAN',    label: 'PAX count:' },
  { re: /^Payload:/,          state: 'SEND',    label: 'Payload:' },
  { re: /^sendReceive/,       state: 'SEND',    label: 'sendReceive' },
  { re: /^TX ok/,             state: 'RECEIVE', label: 'TX ok' },
  { re: /^Downlink:/,         state: 'RECEIVE', label: 'Downlink:' },
  { re: /^TX error/,          state: 'ERROR',   label: 'TX error', isError: true },
  { re: /^Radio init failed/, state: 'ERROR',   label: 'Radio init failed', isError: true },
  { re: /^Join failed/,       state: 'ERROR',   label: 'Join failed', isError: true },
  { re: /^Sleeping for/,      state: 'SLEEP',   label: 'Sleeping for' },
]

export { STATE_PATTERNS }

export function parseLine(text: string): ParseResult {
  for (const pattern of STATE_PATTERNS) {
    if (pattern.re.test(text)) {
      return { state: pattern.state, isError: pattern.isError ?? false }
    }
  }
  return { state: null, isError: false }
}
