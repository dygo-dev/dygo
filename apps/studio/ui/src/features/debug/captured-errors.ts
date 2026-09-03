import { ref } from 'vue'

export type CapturedError = {
  message: string
  stack: string | undefined
  timestamp: string
}

const _capturedErrors = ref<CapturedError[]>([])

export function useCapturedErrors() {
  function capture(err: unknown): void {
    const error = err instanceof Error ? err : new Error(String(err))
    _capturedErrors.value = [
      {
        message: error.message,
        stack: error.stack,
        timestamp: new Date().toISOString(),
      },
      ..._capturedErrors.value,
    ].slice(0, 5)
  }

  function clear(): void {
    _capturedErrors.value = []
  }

  return {
    errors: _capturedErrors,
    capture,
    clear,
  }
}
