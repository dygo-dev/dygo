<script setup lang="ts">
import { computed, useSlots } from 'vue'

const props = withDefaults(defineProps<{
  ariaLabel?: string
}>(), {
  ariaLabel: 'Toolbar',
})

const slots = useSlots()

const hasLeft = computed(() => slotHasContent('left'))
const hasCenter = computed(() => slotHasContent('default'))
const hasRight = computed(() => slotHasContent('right'))

function slotHasContent(name: 'left' | 'default' | 'right'): boolean {
  return (slots[name]?.() ?? []).length > 0
}
</script>

<template>
  <section class="studio-toolbar" role="toolbar" :aria-label="props.ariaLabel">
    <div v-if="hasLeft" class="studio-toolbar__left">
      <slot name="left" />
    </div>

    <div v-if="hasCenter" class="studio-toolbar__center">
      <slot />
    </div>

    <div v-if="hasRight" class="studio-toolbar__right">
      <slot name="right" />
    </div>
  </section>
</template>

<style scoped>
.studio-toolbar {
  display: flex;
  flex-wrap: wrap;
  min-width: 0;
  min-height: 48px;
  align-items: center;
  gap: 8px;
  padding: 8px var(--studio-page-padding);
}

.studio-toolbar__left,
.studio-toolbar__center,
.studio-toolbar__right {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.studio-toolbar__left {
  flex: 1 1 240px;
}

.studio-toolbar__center {
  flex-basis: 100%;
  order: 1;
}

.studio-toolbar__right {
  margin-left: auto;
  justify-content: flex-end;
}

.studio-toolbar :deep(.d-button),
.studio-toolbar :deep(.d-icon-button),
.studio-toolbar :deep(select) {
  min-height: var(--studio-control-height-sm);
}

.studio-toolbar :deep(.d-icon-button) {
  width: var(--studio-control-height-sm);
  height: var(--studio-control-height-sm);
}

@media (max-width: 720px) {
  .studio-toolbar {
    align-items: stretch;
  }

  .studio-toolbar__left,
  .studio-toolbar__center,
  .studio-toolbar__right {
    width: 100%;
    flex-basis: 100%;
    justify-content: flex-start;
    margin-left: 0;
    order: 0;
  }
}
</style>
