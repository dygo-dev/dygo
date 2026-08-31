import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'

import {
  getNotificationDeepLink,
  getUnreadNotificationCount,
  getUnreadNotifications,
  markNotificationRead,
} from './notifications.api'

export const notificationListKey = ['notifications', 'unread'] as const
export const notificationCountKey = ['notifications', 'unread-count'] as const

export function useNotificationsQuery(enabled: MaybeRefOrGetter<boolean> = true) {
  return useQuery({
    queryKey: notificationListKey,
    queryFn: ({ signal }) => getUnreadNotifications(20, signal),
    enabled: computed(() => toValue(enabled)),
    refetchInterval: 30_000,
  })
}

export function useNotificationCountQuery(enabled: MaybeRefOrGetter<boolean> = true) {
  return useQuery({
    queryKey: notificationCountKey,
    queryFn: ({ signal }) => getUnreadNotificationCount(signal),
    enabled: computed(() => toValue(enabled)),
    refetchInterval: 30_000,
  })
}

export function useOpenNotification() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: async (id: number) => {
      const deepLink = await getNotificationDeepLink(id)
      await markNotificationRead(id)
      return deepLink
    },
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: notificationListKey }),
        client.invalidateQueries({ queryKey: notificationCountKey }),
      ])
    },
  })
}
