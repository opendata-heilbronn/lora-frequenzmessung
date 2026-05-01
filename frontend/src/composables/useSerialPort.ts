import { ref } from 'vue'
import type { ConnectionState } from '../types/serial-debugger'

export function useSerialPort() {
  const connectionState = ref<ConnectionState>('disconnected')
  const errorMessage = ref('')
  const webSerialSupported = !!(navigator as any).serial

  let port: any = null
  let reader: ReadableStreamDefaultReader<Uint8Array> | null = null
  let lineCallback: ((line: string) => void) | null = null

  function onLine(cb: (line: string) => void) {
    lineCallback = cb
  }

  async function connect() {
    if (!webSerialSupported) return
    errorMessage.value = ''
    connectionState.value = 'connecting'
    try {
      port = await (navigator as any).serial.requestPort()
      await port!.open({ baudRate: 115200 })
      connectionState.value = 'connected'
      readLoop()
    } catch (e: any) {
      connectionState.value = 'error'
      errorMessage.value = e?.message ?? String(e)
    }
  }

  async function readLoop() {
    if (!port?.readable) return
    reader = port.readable.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    try {
      while (true) {
        const { value, done } = await reader!.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop() ?? ''
        for (const line of lines) {
          const trimmed = line.trimEnd()
          if (trimmed && lineCallback) lineCallback(trimmed)
        }
      }
    } catch {
      // port closed or disconnected
    } finally {
      try { reader?.releaseLock() } catch {}
      reader = null
      if (connectionState.value === 'connected') {
        connectionState.value = 'disconnected'
      }
    }
  }

  async function disconnect() {
    connectionState.value = 'disconnected'
    try { reader?.cancel() } catch {}
    try { await port?.close() } catch {}
    port = null
  }

  return { webSerialSupported, connectionState, errorMessage, connect, disconnect, onLine }
}
