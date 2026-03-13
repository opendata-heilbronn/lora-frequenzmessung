import axios from 'axios'

const api = axios.create({ timeout: 30_000 })

// Inject JWT token on every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Redirect to /login on 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      globalThis.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export default api
