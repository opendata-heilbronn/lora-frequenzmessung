<template>
  <OnyxAppLayout class="onyx-grid-max-md onyx-grid-center">
    <template v-if="!isLoginPage" #navBar>
      <OnyxNavBar app-name="LoRa Sensor Management">
        <OnyxNavItem label="Sensors" link="/" />
        <OnyxNavItem label="+ Add Sensor" link="/add" />
        <template #mobileActivePage>
          <OnyxNavItem label="Sensors" link="/" />
        </template>
        <template #contextArea>
          <OnyxButton label="Logout" variant="plain" @click="logout" />
        </template>
      </OnyxNavBar>
    </template>
    <OnyxPageLayout>
      <router-view />
    </OnyxPageLayout>
  </OnyxAppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { OnyxAppLayout, OnyxNavBar, OnyxPageLayout, OnyxNavItem, OnyxButton } from 'sit-onyx'

const router = useRouter()
const route = useRoute()

const isLoginPage = computed(() => route.path === '/login')

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}
</script>
