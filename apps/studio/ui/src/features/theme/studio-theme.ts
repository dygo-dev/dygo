export const studioThemeStorageKey = 'studio:theme'

export const studioThemePreferences = ['light', 'dark', 'system'] as const

export type StudioThemePreference = (typeof studioThemePreferences)[number]

export type StudioResolvedTheme = 'light' | 'dark'

export const studioThemeOptions: ReadonlyArray<{ value: StudioThemePreference; label: string }> = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
]

export function isStudioThemePreference(value: unknown): value is StudioThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

export function readStudioThemePreference(storedValue: string | null): StudioThemePreference {
  if (isStudioThemePreference(storedValue)) {
    return storedValue
  }

  return 'system'
}

export function resolveStudioTheme(preference: StudioThemePreference, prefersDark: boolean): StudioResolvedTheme {
  if (preference === 'light' || preference === 'dark') {
    return preference
  }

  return prefersDark ? 'dark' : 'light'
}

export function prefersDarkColorScheme(): boolean {
  if (typeof window === 'undefined') {
    return false
  }

  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}

export function getStudioThemePreference(): StudioThemePreference {
  if (typeof window === 'undefined') {
    return 'system'
  }

  try {
    return readStudioThemePreference(window.localStorage.getItem(studioThemeStorageKey))
  } catch {
    return 'system'
  }
}

export function applyStudioTheme(theme: StudioResolvedTheme, root?: HTMLElement | null): void {
  const target = root ?? (typeof document === 'undefined' ? null : document.documentElement)
  if (!target) {
    return
  }

  target.dataset.theme = theme
}

export function syncStudioTheme(root?: HTMLElement | null): StudioResolvedTheme {
  const theme = resolveStudioTheme(getStudioThemePreference(), prefersDarkColorScheme())
  applyStudioTheme(theme, root)
  return theme
}

let preferenceChanged: ((value: StudioThemePreference) => void) | null = null

export function bindStudioThemePreference(listener: typeof preferenceChanged) {
  preferenceChanged = listener
}

export function setStudioThemePreference(preference: StudioThemePreference, root?: HTMLElement | null, notify = true): StudioResolvedTheme {
  if (notify) preferenceChanged?.(preference)
  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(studioThemeStorageKey, preference)
    } catch {
      // Theme preferences are best-effort when browser storage is unavailable.
    }
  }

  const theme = resolveStudioTheme(preference, prefersDarkColorScheme())
  applyStudioTheme(theme, root)
  return theme
}

let systemThemeBound = false

export function installStudioTheme(): StudioResolvedTheme {
  const theme = syncStudioTheme()
  if (typeof window === 'undefined' || systemThemeBound) {
    return theme
  }

  try {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (getStudioThemePreference() === 'system') {
        syncStudioTheme()
      }
    })
    systemThemeBound = true
  } catch {
    // Color-scheme listeners are best-effort when matchMedia is unavailable.
  }

  return theme
}
