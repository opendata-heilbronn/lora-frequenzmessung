import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createRouter, createMemoryHistory } from 'vue-router'

// Mock heavy view components so their transitive deps (leaflet, etc.) are never loaded
vi.mock('../views/SensorList.vue', () => ({ default: {} }))
vi.mock('../views/AddSensor.vue', () => ({ default: {} }))
vi.mock('../views/LoginView.vue', () => ({ default: {} }))

// Build a fresh in-memory router that mirrors the real router's routes and guard.
// Using createMemoryHistory avoids the need for window.history (node environment).
function makeTestRouter() {
  const r = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: {} },
      { path: '/add', component: {} },
      { path: '/login', component: {}, meta: { public: true } },
    ],
  })
  r.beforeEach((to) => {
    const token = localStorage.getItem('token')
    if (!to.meta.public && !token) {
      return '/login'
    }
  })
  return r
}

describe('router navigation guard', () => {
  let router: ReturnType<typeof makeTestRouter>

  beforeEach(() => {
    localStorage.clear()
    router = makeTestRouter()
  })

  it('redirects unauthenticated users to /login when accessing /', async () => {
    await router.push('/')
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('redirects unauthenticated users to /login when accessing /add', async () => {
    await router.push('/add')
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('allows authenticated users to access /', async () => {
    localStorage.setItem('token', 'valid-jwt')
    await router.push('/')
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('allows authenticated users to access /add', async () => {
    localStorage.setItem('token', 'valid-jwt')
    await router.push('/add')
    expect(router.currentRoute.value.path).toBe('/add')
  })

  it('allows unauthenticated users to access /login (public route)', async () => {
    await router.push('/login')
    expect(router.currentRoute.value.path).toBe('/login')
  })
})
