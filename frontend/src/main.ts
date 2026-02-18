import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import SensorList from './views/SensorList.vue'
import AddSensor from './views/AddSensor.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: SensorList },
    { path: '/add', component: AddSensor },
  ],
})

createApp(App).use(router).mount('#app')
