import { getToken } from './auth'
import type { ApiError } from '../types'

export async function api<T = any>(path: string, opts: RequestInit = {}): Promise<T> {
  const token = getToken()
  const authHeader = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...authHeader, ...(opts.headers || {}) },
    ...opts,
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const data = (await res.json()) as Partial<ApiError>
      msg = (data.message || (data as any).error || msg) as string
      if (data.traceId) msg += ` (traceId: ${data.traceId})`
    } catch {}
    const err = new Error(msg) as Error & { status?: number }
    err.status = res.status
    throw err
  }
  const ct = res.headers.get('Content-Type') || ''
  if (ct.includes('application/json')) return res.json() as Promise<T>
  return (res.text() as unknown) as T
}
