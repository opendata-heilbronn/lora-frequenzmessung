<template>
  <div class="login-container">
    <div class="login-card" data-onyx-theme="light">
      <h1>LoRa Sensor Management</h1>
      <form @submit.prevent="handleLogin">
        <OnyxInput
          v-model="username"
          label="Username"
          type="text"
          autocomplete="username"
          required
        />
        <OnyxInput
          v-model="password"
          label="Password"
          type="password"
          autocomplete="current-password"
          required
        />
        <p v-if="error" class="error">{{ error }}</p>
        <OnyxButton type="submit" label="Sign in" :loading="loading" />
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { OnyxInput, OnyxButton } from 'sit-onyx'
import axios from 'axios'
import { loadAndRefreshToken } from "../auth/user.ts";

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const resp = await axios.post('/auth/login', {
      username: username.value,
      password: password.value,
    })
    localStorage.setItem('token', resp.data.token)
    loadAndRefreshToken()
    router.push('/')
  } catch (e: any) {
    error.value = e.response?.data?.error ?? 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}

.login-card {
  background: white;
  border-radius: 8px;
  padding: 2rem;
  width: 100%;
  max-width: 360px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  color-scheme: light;
}

h1 {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0;
  text-align: center;
  color: var(--onyx-color-text-icons-neutral-intense, #0d1117);
}

:deep(.onyx-button) {
  width: 100%;
}

form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.error {
  color: var(--onyx-color-base-danger-500, #d32f2f);
  font-size: 0.875rem;
  margin: 0;
}
</style>
