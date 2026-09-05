import { ref } from 'vue'

export type CapturedError = {
  id: number
  message: string
  stack: string | undefined
  timestamp: string
}

let nextErrorID = 0
const capturedErrors = ref<CapturedError[]>([])

export function useCapturedErrors() {
  function capture(err: unknown): void {
    const error = err instanceof Error ? err : new Error(String(err))
    capturedErrors.value = [
      {
        id: ++nextErrorID,
        message: error.message,
        stack: error.stack,
        timestamp: new Date().toISOString(),
      },
      ...capturedErrors.value,
    ].slice(0, 5)
  }

  function clear(): void {
    capturedErrors.value = []
  }

  return {
    errors: capturedErrors,
    capture,
    clear,
  }
}
