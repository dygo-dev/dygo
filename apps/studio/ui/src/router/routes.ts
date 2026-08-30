export const RouteName = {
  Login: 'login',
  Home: 'home',
  EntityRecords: 'entity-records',
  RecordNew: 'record-new',
  RecordDetail: 'record-detail',
  NotFound: 'not-found',
} as const

export type RouteNameValue = (typeof RouteName)[keyof typeof RouteName]

export const entityChildReservedSlugs = new Set(['new'])

export function isEntityChildReservedSlug(value: string): boolean {
  return entityChildReservedSlugs.has(normalizeSlug(value))
}

export function normalizeSlug(value: string): string {
  return value.trim().toLowerCase()
}

export function pageRouteName(app: string, key: string): string {
  return `page:${app.trim()}:${key.trim()}`
}

export function normalizePageClaimPath(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const path = value.trim()
  if (
    path === ''
    || !path.startsWith('/')
    || path.startsWith('//')
    || path.includes('?')
    || path.includes('#')
    || path.includes(':')
    || path.includes('*')
  ) {
    return null
  }

  return path.length > 1 ? path.replace(/\/+$/, '') : path
}

export function routeParam(value: string | string[]): string {
  return Array.isArray(value) ? (value[0] ?? '') : value
}
