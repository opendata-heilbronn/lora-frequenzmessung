import { createRouter, createWebHistory } from 'vue-router'
import SensorList from './views/SensorList.vue'
import AddSensor from './views/AddSensor.vue'
import LoginView from './views/LoginView.vue'
import UserList from "./views/UserList.vue";
import SerialDebugger from './views/SerialDebugger.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: SensorList },
    { path: '/add', component: AddSensor },
    { path: '/users', component: UserList },
    { path: '/debug', component: SerialDebugger },
    { path: '/login', component: LoginView, meta: { public: true } },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    return '/login'
  }
})

export default router
