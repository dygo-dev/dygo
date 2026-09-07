<script setup lang="ts">
import { ref } from 'vue'
import { Check, ChevronRight, LogOut, Palette, RefreshCw } from '@lucide/vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuItemIndicator,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from 'reka-ui'
import { useRouter } from 'vue-router'

import { queryClient } from '@/app/query'
import { reloadStudioApp } from '@/app/reload'
import Avatar from '@/design/atoms/Avatar.vue'
import {
  getStudioThemePreference,
  isStudioThemePreference,
  setStudioThemePreference,
  studioThemeOptions,
  type StudioThemePreference,
} from '@/features/theme'
import { RouteName } from '@/router/routes'
import { useAuthStore } from '@/stores/auth.store'

withDefaults(defineProps<{
  userName?: string
  userAvatarUrl?: string
}>(), {
  userName: 'Studio user',
})

const router = useRouter()
const authStore = useAuthStore()
const reloading = ref(false)
const themePreference = ref<StudioThemePreference>(getStudioThemePreference())

async function reloadApp() {
  if (reloading.value) {
    return
  }

  reloading.value = true
  try {
    await reloadStudioApp(router)
  } finally {
    reloading.value = false
  }
}

function onThemePreference(value: unknown) {
  if (!isStudioThemePreference(value)) {
    return
  }

  themePreference.value = value
  setStudioThemePreference(value)
}

async function logout() {
  await authStore.logout()
  queryClient.clear()
  await router.replace({ name: RouteName.Login })
}
</script>

<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger as-child>
      <button class="studio-user-menu__trigger" type="button" :aria-label="`${userName} menu`">
        <Avatar :name="userName" :image-url="userAvatarUrl" />
      </button>
    </DropdownMenuTrigger>

    <DropdownMenuPortal>
      <DropdownMenuContent
        class="studio-user-menu__content"
        align="end"
        :side-offset="8"
      >
        <DropdownMenuItem class="studio-user-menu__item" :disabled="reloading" @select="reloadApp">
          <RefreshCw :size="14" :stroke-width="1.8" aria-hidden="true" />
          <span>Reload</span>
        </DropdownMenuItem>
        <DropdownMenuSub>
          <DropdownMenuSubTrigger class="studio-user-menu__item">
            <Palette :size="14" :stroke-width="1.8" aria-hidden="true" />
            <span>Theme</span>
            <ChevronRight class="studio-user-menu__chevron" :size="14" :stroke-width="1.8" aria-hidden="true" />
          </DropdownMenuSubTrigger>
          <DropdownMenuPortal>
            <DropdownMenuSubContent class="studio-user-menu__content" :side-offset="6">
              <DropdownMenuRadioGroup :model-value="themePreference" @update:model-value="onThemePreference">
                <DropdownMenuRadioItem
                  v-for="option in studioThemeOptions"
                  :key="option.value"
                  class="studio-user-menu__item studio-user-menu__item--radio"
                  :value="option.value"
                >
                  <DropdownMenuItemIndicator class="studio-user-menu__indicator">
                    <Check :size="13" :stroke-width="2.2" aria-hidden="true" />
                  </DropdownMenuItemIndicator>
                  <span>{{ option.label }}</span>
                </DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuPortal>
        </DropdownMenuSub>
        <DropdownMenuSeparator class="studio-user-menu__separator" />
        <DropdownMenuItem class="studio-user-menu__item" @select="logout">
          <LogOut :size="14" :stroke-width="1.8" aria-hidden="true" />
          <span>Logout</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>

<style scoped>
.studio-user-menu__trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: inherit;
  padding: 0;
}

.studio-user-menu__trigger:focus-visible {
  outline: 2px solid var(--studio-focus);
  outline-offset: 2px;
}

.studio-user-menu__content {
  z-index: 50;
  min-width: 160px;
  overflow: hidden;
  border: 1px solid var(--studio-border);
  border-radius: var(--studio-radius-control);
  background: var(--studio-surface);
  box-shadow: var(--studio-shadow-sheet);
  padding: 5px;
}

.studio-user-menu__item {
  display: flex;
  min-height: 30px;
  align-items: center;
  gap: 8px;
  border-radius: 5px;
  color: var(--studio-text-muted);
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  outline: none;
  padding: 0 8px;
  user-select: none;
}

.studio-user-menu__item--radio {
  position: relative;
  padding-left: 28px;
}

.studio-user-menu__item[data-highlighted],
.studio-user-menu__item[data-state='open'] {
  background: var(--studio-surface-raised);
  color: var(--studio-text);
}

.studio-user-menu__item[data-disabled] {
  color: var(--studio-text-subtle);
  pointer-events: none;
}

.studio-user-menu__chevron {
  margin-left: auto;
}

.studio-user-menu__indicator {
  position: absolute;
  left: 8px;
  display: inline-flex;
  width: 14px;
  align-items: center;
  justify-content: center;
}

.studio-user-menu__separator {
  height: 1px;
  background: var(--studio-border);
  margin: 5px -5px;
}
</style>
