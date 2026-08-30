import type { Component } from 'vue'

import EntityIndexRenderer from '@/renderers/pages/EntityIndexRenderer.vue'

const renderers = new Map<string, Component>([
  ['entity-index', EntityIndexRenderer],
])

// TODO: Define a sandboxed, versioned module contract before loading Page
// renderers from App-provided JavaScript at runtime.

export function pageRenderer(name: string): Component | null {
  return renderers.get(name.trim()) ?? null
}

export function registerPageRenderer(name: string, renderer: Component): () => void {
  const key = name.trim()
  if (key === '') {
    throw new Error('Page renderer name is required')
  }
  if (renderers.has(key)) {
    throw new Error(`Page renderer ${key} is already registered`)
  }

  renderers.set(key, renderer)
  return () => {
    if (renderers.get(key) === renderer) {
      renderers.delete(key)
    }
  }
}
