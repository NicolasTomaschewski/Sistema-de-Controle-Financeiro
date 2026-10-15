// Cliente HTTP compartilhado por toda a aplicação. Centraliza:
// - envio de cookies de sessão (credentials: 'include');
// - leitura do token CSRF do cookie e envio no cabeçalho X-CSRF-Token em
//   métodos que alteram estado (exigido pelo backend);
// - decodificação do envelope padrão de resposta ({ data, meta } ou
//   { error }) e conversão de erros em exceções tipadas.

export class ApiError extends Error {
  status: number
  code: string
  fields?: Record<string, string>

  constructor(status: number, code: string, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.code = code
    this.fields = fields
  }
}

function readCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : null
}

type RequestOptions = {
  method?: string
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
}

function buildQuery(query?: RequestOptions['query']): string {
  if (!query) return ''
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = {}
  let body: string | undefined

  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.body)
  }

  if (method !== 'GET' && method !== 'HEAD') {
    const csrfToken = readCookie('csrf_token')
    if (csrfToken) headers['X-CSRF-Token'] = csrfToken
  }

  const response = await fetch(path + buildQuery(options.query), {
    method,
    headers,
    body,
    credentials: 'include',
  })

  if (response.status === 204) {
    return undefined as T
  }

  const contentType = response.headers.get('content-type') ?? ''
  const payload = contentType.includes('application/json') ? await response.json() : null

  if (!response.ok) {
    const errBody = payload?.error
    throw new ApiError(
      response.status,
      errBody?.code ?? 'unknown_error',
      errBody?.message ?? 'Ocorreu um erro inesperado.',
      errBody?.fields,
    )
  }

  return (payload?.data ?? payload) as T
}

export const api = {
  get: <T>(path: string, query?: RequestOptions['query']) => request<T>(path, { method: 'GET', query }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

export type Meta = { total?: number; limit?: number; offset?: number }

export async function getWithMeta<T>(path: string, query?: RequestOptions['query']): Promise<{ data: T; meta: Meta }> {
  const method = 'GET'
  const response = await fetch(path + buildQuery(query), { method, credentials: 'include' })
  const payload = await response.json()
  if (!response.ok) {
    throw new ApiError(response.status, payload?.error?.code ?? 'unknown_error', payload?.error?.message ?? 'Erro inesperado.', payload?.error?.fields)
  }
  return { data: payload.data as T, meta: (payload.meta ?? {}) as Meta }
}
