import { createApp, computed } from 'vue'
import { createOnyx } from 'sit-onyx'
import '@fontsource-variable/source-sans-3'
import '@fontsource-variable/source-code-pro'
import 'sit-onyx/style.css'
import 'sit-onyx/global.css'
import App from './App.vue'
import router from './router'

createApp(App)
  .use(createOnyx({
    router: {
      push: (to: string) => router.push(to),
      currentRoute: computed(() => ({ path: router.currentRoute.value.path })),
    },
  }))
  .use(router)
  .mount('#app')
