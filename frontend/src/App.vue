<template>
  <OnyxAppLayout class="onyx-grid-max-md onyx-grid-center">
    <template v-if="!isLoginPage" #navBar>
      <OnyxNavBar app-name="LoRa Sensor Management">
        <OnyxNavItem label="Sensors" link="/" />
        <OnyxNavItem label="+ Add Sensor" link="/add" />
        <OnyxNavItem label="User" link="/users" />
        <OnyxNavItem label="Debug" link="/debug" />
        <template #mobileActivePage>
          <OnyxNavItem label="Sensors" link="/" />
        </template>
        <template #contextArea>
          <OnyxButton label="Logout" mode="plain" color="neutral" @click="logout" />
        </template>
      </OnyxNavBar>
    </template>
    <router-view />
  </OnyxAppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { OnyxAppLayout, OnyxNavBar, OnyxNavItem, OnyxButton } from 'sit-onyx'
import {loadAndRefreshToken} from "./auth/user.ts";

const router = useRouter()
const route = useRoute()

const isLoginPage = computed(() => route.path === '/login')

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}

loadAndRefreshToken();
</script>
