import { ApiClientError, apiRequest, type ApiErrorEnvelope, type DataEnvelope } from '../api/client.ts'

export type StudioPageApp = {
  name: string
  label: string
}

export type StudioPageDescriptor = {
  name: string
  key: string
  label: string
  description: string
  icon: string
  path: string
  renderer: string
  options: Record<string, unknown>
  app: StudioPageApp
}

type PageRequestOptions = {
  signal?: AbortSignal
}

export class PageApiError extends ApiClientError {
  constructor(code: string, message: string, details?: Record<string, unknown>) {
    super('PageApiError', code, message, details)
  }
}

export async function getPage(app: string, key: string, options: PageRequestOptions = {}): Promise<StudioPageDescriptor> {
  const payload = await apiRequest<DataEnvelope<StudioPageDescriptor>, PageApiError>(
    `/api/v1/pages/${encodeURIComponent(app)}/${encodeURIComponent(key)}`,
    { method: 'GET', signal: options.signal },
    {
      error: PageApiError,
      fallbackCode: 'page_failed',
      invalidResponseMessage: 'Studio could not read the Page response.',
      message: pageErrorMessage,
    },
  )

  return payload.data
}

function pageErrorMessage(payload: ApiErrorEnvelope): string {
  switch (payload.error?.code) {
    case 'unauthenticated':
      return 'Sign in to open this Page.'
    case 'forbidden':
      return 'You do not have access to this Page.'
    case 'not_found':
      return payload.error.message ?? 'Studio could not find this Page.'
    default:
      return payload.error?.message ?? 'Studio could not load this Page.'
  }
}
