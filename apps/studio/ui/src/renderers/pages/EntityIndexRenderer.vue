<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUpRight } from '@lucide/vue'
import { RouterLink } from 'vue-router'

import { ErrorState, Spinner } from '@/design'
import { iconForEntity } from '@/features/metadata/entity-icons'
import type { MetadataEntity } from '@/features/metadata/metadata.api'
import { useMetadataEntitiesQuery } from '@/features/metadata/metadata.query'
import type { StudioPageDescriptor } from '@/features/pages/pages.api'
import PageHeader from '@/shell/PageHeader.vue'
import { storeError } from '@/stores/status'

defineProps<{
  page: StudioPageDescriptor
}>()

type EntityGroup = {
  app: MetadataEntity['app']
  entities: MetadataEntity[]
}

const entitiesQuery = useMetadataEntitiesQuery()
const loading = computed(() => entitiesQuery.isPending.value)
const error = computed(() => entitiesQuery.error.value
  ? storeError(entitiesQuery.error.value, 'Studio could not load the entity index.')
  : null)
const groups = computed<EntityGroup[]>(() => {
  const grouped = new Map<string, EntityGroup>()

  for (const entity of entitiesQuery.data.value ?? []) {
    if (entity['is-collection'] || !entity.slug) {
      continue
    }

    const current = grouped.get(entity.app.name)
    if (current) {
      current.entities.push(entity)
      continue
    }
    grouped.set(entity.app.name, { app: entity.app, entities: [entity] })
  }

  return [...grouped.values()]
    .map((group) => ({
      ...group,
      entities: [...group.entities].sort((left, right) => left.label.localeCompare(right.label)),
    }))
    .sort((left, right) => left.app.label.localeCompare(right.app.label))
})
</script>

<template>
  <section class="studio-page entity-index" aria-labelledby="studio-home-title">
    <PageHeader
      title-id="studio-home-title"
      :title="page.label"
    />

    <div v-if="loading" class="entity-index__state" aria-live="polite">
      <Spinner size="sm" label="Loading entities" />
      <p>Loading entities</p>
    </div>

    <div v-else-if="error" class="entity-index__state">
      <ErrorState title="Entity index unavailable" :message="error.message" />
    </div>

    <div v-else-if="groups.length === 0" class="entity-index__empty">
      <p class="entity-index__empty-title">No entities are available</p>
      <p>Your Studio access does not include an entity yet.</p>
    </div>

    <div v-else class="entity-index__groups">
      <section v-for="group in groups" :key="group.app.name" class="entity-index__group">
        <header class="entity-index__group-header">
          <div>
            <p class="entity-index__app-key">{{ group.app.name }}</p>
            <h2>{{ group.app.label }}</h2>
          </div>
          <span>{{ group.entities.length }} {{ group.entities.length === 1 ? 'entity' : 'entities' }}</span>
        </header>

        <div class="entity-index__links">
          <RouterLink
            v-for="entity in group.entities"
            :key="entity.name"
            class="entity-index__link"
            :to="`/${entity.slug}`"
          >
            <component
              :is="iconForEntity(entity.icon)"
              class="entity-index__icon"
              aria-hidden="true"
              :size="17"
              :stroke-width="1.7"
            />
            <span class="entity-index__copy">
              <strong>{{ entity.label }}</strong>
              <span v-if="entity.description">{{ entity.description }}</span>
              <span v-else>Open {{ entity.label }}</span>
            </span>
            <ArrowUpRight class="entity-index__arrow" aria-hidden="true" :size="15" :stroke-width="1.8" />
          </RouterLink>
        </div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.entity-index {
  gap: 0;
  min-height: 100%;
}

.entity-index__groups {
  display: grid;
  gap: 28px;
  padding: 22px 2px 40px;
}

.entity-index__group {
  display: grid;
  grid-template-columns: minmax(150px, 0.28fr) minmax(0, 1fr);
  gap: 24px;
}

.entity-index__group-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  border-top: 1px solid var(--studio-border-strong);
  padding-top: 12px;
}

.entity-index__app-key {
  margin: 0 0 4px;
  color: var(--studio-text-subtle);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
}

.entity-index__group h2 {
  margin: 0;
  color: var(--studio-text);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
}

.entity-index__group-header > span {
  color: var(--studio-text-subtle);
  font-size: 11px;
  line-height: 1.3;
  white-space: nowrap;
}

.entity-index__links {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-top: 1px solid var(--studio-border);
}

.entity-index__link {
  display: grid;
  min-height: 78px;
  align-items: center;
  grid-template-columns: 24px minmax(0, 1fr) 18px;
  gap: 10px;
  border-bottom: 1px solid var(--studio-border);
  color: var(--studio-text);
  padding: 12px 10px;
  text-decoration: none;
  transition: background-color 160ms ease;
}

.entity-index__link:nth-child(odd) {
  border-right: 1px solid var(--studio-border);
}

.entity-index__link:hover {
  background: var(--studio-surface-raised);
}

.entity-index__link:focus-visible {
  position: relative;
  z-index: 1;
  outline: 2px solid var(--studio-focus);
  outline-offset: -2px;
}

.entity-index__icon,
.entity-index__arrow {
  color: var(--studio-text-subtle);
}

.entity-index__arrow {
  opacity: 0;
  transition: opacity 160ms ease, transform 160ms ease;
}

.entity-index__link:hover .entity-index__arrow,
.entity-index__link:focus-visible .entity-index__arrow {
  opacity: 1;
  transform: translate(1px, -1px);
}

.entity-index__copy {
  display: grid;
  min-width: 0;
  gap: 4px;
}

.entity-index__copy strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 650;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entity-index__copy > span {
  display: -webkit-box;
  overflow: hidden;
  color: var(--studio-text-muted);
  font-size: 12px;
  line-height: 1.4;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.entity-index__state,
.entity-index__empty {
  display: grid;
  justify-items: start;
  gap: 8px;
  padding: 96px 2px 44px;
}

.entity-index__state p,
.entity-index__empty p {
  margin: 0;
  color: var(--studio-text-muted);
  font-size: 13px;
  line-height: 1.45;
}

.entity-index__empty .entity-index__empty-title {
  color: var(--studio-text);
  font-weight: 700;
}

@media (max-width: 900px) {
  .entity-index__group {
    grid-template-columns: 1fr;
    gap: 10px;
  }
}

@media (max-width: 620px) {
  .entity-index__links {
    grid-template-columns: 1fr;
  }

  .entity-index__link:nth-child(odd) {
    border-right: 0;
  }
}
</style>
