import { describe, it, expect, beforeEach } from 'vitest'
import router from '../router'

describe('router navigation guard', () => {
  beforeEach(async () => {
    localStorage.clear()
    // Reset to a known public route so each test starts from a clean slate
    await router.push('/login')
  })

  it('redirects unauthenticated users to /login when accessing a protected route', async () => {
    await router.push('/')
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('redirects unauthenticated users to /login when accessing /add', async () => {
    await router.push('/add')
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('allows authenticated users to access protected routes', async () => {
    localStorage.setItem('token', 'valid-jwt')
    await router.push('/')
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('allows authenticated users to access /add', async () => {
    localStorage.setItem('token', 'valid-jwt')
    await router.push('/add')
    expect(router.currentRoute.value.path).toBe('/add')
  })

  it('allows unauthenticated users to reach /login (public route)', async () => {
    await router.push('/login')
    expect(router.currentRoute.value.path).toBe('/login')
  })
})
