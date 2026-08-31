import { ApiClientError, apiRequest, type ApiErrorEnvelope, type DataEnvelope, type ListEnvelope } from '../api/client.ts'

export type StudioNotification = {
  id: number
  name: string
  title: string
  message: string
  'deep-link': string
  'created-at': string
  'read-at': string | null
}

export type NotificationListMeta = {
  limit: number
  count: number
}

export class NotificationApiError extends ApiClientError {
  constructor(code: string, message: string, details?: Record<string, unknown>) {
    super('NotificationApiError', code, message, details)
  }
}

export async function getUnreadNotifications(limit = 20, signal?: AbortSignal): Promise<StudioNotification[]> {
  const payload = await notificationRequest<ListEnvelope<StudioNotification[], NotificationListMeta>>(
    `/api/v1/notifications?limit=${limit}`,
    { method: 'GET', signal },
  )
  return payload.data
}

export async function getUnreadNotificationCount(signal?: AbortSignal): Promise<number> {
  const payload = await notificationRequest<DataEnvelope<{ count: number }>>(
    '/api/v1/notifications/unread-count',
    { method: 'GET', signal },
  )
  return payload.data.count
}

export async function markNotificationRead(id: number): Promise<StudioNotification> {
  const payload = await notificationRequest<DataEnvelope<StudioNotification>>(
    `/api/v1/notifications/${id}/read`,
    { method: 'POST' },
  )
  return payload.data
}

export async function getNotificationDeepLink(id: number): Promise<string> {
  const payload = await notificationRequest<DataEnvelope<{ 'deep-link': string }>>(
    `/api/v1/notifications/${id}/deep-link`,
    { method: 'GET' },
  )
  return payload.data['deep-link']
}

function notificationRequest<T>(input: string, init: RequestInit): Promise<T> {
  return apiRequest<T, NotificationApiError>(input, init, {
    error: NotificationApiError,
    fallbackCode: 'notification_failed',
    invalidResponseMessage: 'Studio could not read the notification response.',
    message: notificationErrorMessage,
  })
}

function notificationErrorMessage(payload: ApiErrorEnvelope): string {
  switch (payload.error?.code) {
    case 'unauthenticated':
      return 'Sign in to load notifications.'
    case 'not_found':
      return 'This notification is no longer available.'
    default:
      return payload.error?.message ?? 'Studio could not load notifications.'
  }
}
