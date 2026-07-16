import { request, setAdminCSRFToken } from '../src/api/http'

const requests: RequestInit[] = []
globalThis.fetch = (async (_input: RequestInfo | URL, init?: RequestInit) => {
  requests.push(init || {})
  return new Response(JSON.stringify({ code: 0, message: 'ok', data: { ok: true } }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  })
}) as typeof fetch

setAdminCSRFToken('session-csrf-token')
await request('/admin/runtime-settings/groups/test', { method: 'PUT', body: '{}' })
await request('/admin/auth/me')
setAdminCSRFToken('')
await request('/admin/auth/logout', { method: 'POST' })

const writeHeaders = new Headers(requests[0]?.headers)
if (writeHeaders.get('X-CSRF-Token') !== 'session-csrf-token') {
  throw new Error('admin write request did not include the session CSRF token')
}
const readHeaders = new Headers(requests[1]?.headers)
if (readHeaders.has('X-CSRF-Token')) {
  throw new Error('admin read request must not include the CSRF token')
}
const clearedHeaders = new Headers(requests[2]?.headers)
if (clearedHeaders.has('X-CSRF-Token')) {
  throw new Error('admin request retained the CSRF token after session clear')
}

console.log('Admin CSRF request checks passed')
