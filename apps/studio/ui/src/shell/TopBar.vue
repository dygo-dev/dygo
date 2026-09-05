<script setup lang="ts">
import LogoMark from '@/design/atoms/LogoMark.vue'
import NotificationMenu from '@/features/notifications/NotificationMenu.vue'
import CommandMenu from './CommandMenu.vue'
import UserMenu from './UserMenu.vue'

withDefaults(defineProps<{
  brandLabel?: string
  brandMark?: string
  userName?: string
  userAvatarUrl?: string
}>(), {
  brandLabel: 'dygo Studio',
  brandMark: 'd',
  userName: 'Studio user',
})
</script>

<template>
  <header class="studio-top-bar">
    <div class="studio-top-bar__brand">
      <LogoMark :label="brandLabel" :mark="brandMark" />
    </div>

    <div class="studio-top-bar__bar">
      <CommandMenu class="studio-top-bar__search" />

      <div class="studio-top-bar__right">
        <slot name="actions" />
        <NotificationMenu />
        <UserMenu :user-name="userName" :user-avatar-url="userAvatarUrl" />
      </div>
    </div>
  </header>
</template>

<style scoped>
.studio-top-bar {
  display: grid;
  min-height: var(--studio-shell-header-height);
  grid-template-columns: var(--studio-shell-sidebar-width) minmax(0, 1fr);
  align-items: center;
}

.studio-top-bar__brand {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  padding-left: var(--studio-shell-gutter);
  padding-right: 14px;
}

.studio-top-bar__bar {
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 312px) auto;
  align-items: center;
  gap: 14px;
  padding-right: var(--studio-shell-gutter);
}

.studio-top-bar__search {
  grid-column: 2;
}

.studio-top-bar__right {
  display: inline-flex;
  grid-column: 3;
  align-items: center;
  justify-self: end;
  gap: 10px;
}

@media (max-width: 720px) {
  .studio-top-bar {
    grid-template-columns: minmax(0, 1fr) auto;
    padding-inline: 14px;
  }

  .studio-top-bar__brand {
    padding: 0;
  }

  .studio-top-bar__bar {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 0;
  }

  .studio-top-bar__search {
    display: none;
  }
}
</style>
