export interface ApiErrorBody {
  code: string
  message: string
  field_errors?: Array<{ field: string; message: string }>
  request_id: string
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: ApiErrorBody,
  ) {
    super(body.message)
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body) headers.set('Content-Type', 'application/json')
  headers.set('X-Actor-ID', 'admin-demo')
  const response = await fetch(`/api/v1${path}`, { ...init, headers })
  const body = await response.json()
  if (!response.ok) throw new ApiError(response.status, body as ApiErrorBody)
  return body as T
}

