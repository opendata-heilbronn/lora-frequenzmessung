import { describe, it, expect, beforeEach, vi } from 'vitest'
import api from '../api'

// Extract the interceptor handlers registered on the axios instance.
// axios 1.x stores them in interceptors.request.handlers / interceptors.response.handlers
// as arrays of { fulfilled, rejected } objects (null entries = ejected interceptors).
type ReqConfig = { headers: Record<string, string> }
type ResHandler = { fulfilled: (r: unknown) => unknown; rejected: (e: unknown) => Promise<never> }

const reqFulfilled = ((api.interceptors.request as any).handlers as Array<{ fulfilled: (c: ReqConfig) => ReqConfig } | null>)[0]!.fulfilled
const resHandlers = ((api.interceptors.response as any).handlers as Array<ResHandler | null>)[0]!

describe('api — request interceptor', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('attaches Bearer token from localStorage when present', () => {
    localStorage.setItem('token', 'test-jwt-token')
    const config: ReqConfig = { headers: {} }
    const result = reqFulfilled(config)
    expect(result.headers['Authorization']).toBe('Bearer test-jwt-token')
  })

  it('does not set Authorization header when no token is stored', () => {
    const config: ReqConfig = { headers: {} }
    const result = reqFulfilled(config)
    expect(result.headers['Authorization']).toBeUndefined()
  })
})

describe('api — response interceptor', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.stubGlobal('location', { href: '' })
  })

  it('passes successful responses through unchanged', () => {
    const response = { status: 200, data: { ok: true } }
    expect(resHandlers.fulfilled(response)).toBe(response)
  })

  it('clears the stored token and redirects to /login on 401', async () => {
    localStorage.setItem('token', 'old-token')
    const error = { response: { status: 401 } }

    await expect(resHandlers.rejected(error)).rejects.toEqual(error)

    expect(localStorage.getItem('token')).toBeNull()
    expect(window.location.href).toBe('/login')
  })

  it('re-throws non-401 errors without clearing the token', async () => {
    localStorage.setItem('token', 'valid-token')
    const error = { response: { status: 500 } }

    await expect(resHandlers.rejected(error)).rejects.toEqual(error)

    expect(localStorage.getItem('token')).toBe('valid-token')
    expect(window.location.href).not.toBe('/login')
  })

  it('re-throws network errors (no response) without clearing the token', async () => {
    localStorage.setItem('token', 'valid-token')
    const error = { message: 'Network Error' } // no .response

    await expect(resHandlers.rejected(error)).rejects.toEqual(error)

    expect(localStorage.getItem('token')).toBe('valid-token')
  })
})
