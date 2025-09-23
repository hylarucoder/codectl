const KEY = 'rb_token'

export function getToken(): string {
  return localStorage.getItem(KEY) || ''
}

export function setToken(t: string): void {
  localStorage.setItem(KEY, t)
}

export function clearToken(): void {
  localStorage.removeItem(KEY)
}
