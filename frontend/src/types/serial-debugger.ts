export type SensorState = 'IDLE' | 'INIT' | 'SCAN' | 'SEND' | 'RECEIVE' | 'SLEEP' | 'ERROR'

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

export interface LogLine {
  id: number
  text: string
  state: SensorState | null
  isError: boolean
  timestamp: Date
}
