import type { MetadataEntity } from '../features/metadata/metadata.api'

export function humanizeEntity(value: string): string {
  return value
    .replace(/[-_]+/g, ' ')
    .replace(/\b\w/g, (letter) => letter.toUpperCase())
}

export function findEntityByRouteSlug(entities: MetadataEntity[], slug: string): MetadataEntity | undefined {
  return entities.find((entity) => entity.slug !== null && entity.slug === slug)
}
