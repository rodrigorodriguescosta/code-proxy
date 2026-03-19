<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api, setDashboardToken } from './api.js'

const route = useRoute()

// Theme
const theme = ref(localStorage.getItem('theme') || 'dark')
function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem('theme', theme.value)
  document.documentElement.classList.toggle('light', theme.value === 'light')
}
onMounted(() => {
  document.documentElement.classList.toggle('light', theme.value === 'light')
})

// Auth gate
const authChecked = ref(false)
const needsLogin = ref(false)
const loginPassword = ref('')
const loginError = ref('')

onMounted(async () => {
  try {
    const status = await api.authStatus()
    if (status.has_password && !status.authenticated) {
      needsLogin.value = true
    }
  } catch {
    // No auth configured — allow through
  }
  authChecked.value = true
})

async function doLogin() {
  loginError.value = ''
  try {
    const res = await api.authLogin(loginPassword.value)
    setDashboardToken(res.token)
    needsLogin.value = false
  } catch (e) {
    loginError.value = e.message || 'Invalid password'
  }
}

const nav = [
  { path: '/', label: 'Usage' },
  { path: '/providers', label: 'Providers' },
  { path: '/keys', label: 'API Keys' },
  { path: '/accounts', label: 'Account Usage' },
  { path: '/tunnel', label: 'Tunnel' },
  { path: '/logs', label: 'Logs' },
  { path: '/settings', label: 'Settings' },
]
</script>

<template>
  <!-- Login gate -->
  <div v-if="needsLogin" class="flex items-center justify-center min-h-screen"
       :class="theme === 'light' ? 'bg-gray-100' : 'bg-gray-950'">
    <div class="w-80 p-6 rounded-xl border"
         :class="theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
      <div class="flex items-center gap-2 mb-6">
        <div class="w-8 h-8 rounded-lg bg-orange-500/20 flex items-center justify-center">
          <span class="text-orange-400 text-sm font-bold">CP</span>
        </div>
        <h1 class="text-lg font-bold" :class="theme === 'light' ? 'text-gray-900' : 'text-white'">Code Proxy</h1>
      </div>
      <form @submit.prevent="doLogin" class="space-y-4">
        <input v-model="loginPassword" type="password" placeholder="Dashboard password"
               class="w-full px-4 py-2.5 rounded-lg border text-sm focus:outline-none focus:border-blue-500"
               :class="theme === 'light'
                 ? 'bg-gray-50 border-gray-200 text-gray-900 placeholder-gray-400'
                 : 'bg-gray-950 border-gray-700 text-white placeholder-gray-600'" />
        <p v-if="loginError" class="text-red-400 text-xs">{{ loginError }}</p>
        <button type="submit"
                class="w-full bg-blue-600 hover:bg-blue-700 text-white py-2.5 rounded-lg text-sm font-medium transition-colors">
          Login
        </button>
      </form>
    </div>
  </div>

  <!-- Main app -->
  <div v-else-if="authChecked" class="flex min-h-screen"
       :class="theme === 'light' ? 'bg-gray-50' : 'bg-gray-950'">
    <!-- Sidebar -->
    <aside class="w-52 border-r flex flex-col shrink-0"
           :class="theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
      <div class="p-4 border-b" :class="theme === 'light' ? 'border-gray-200' : 'border-gray-800'">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-lg bg-orange-500/20 flex items-center justify-center">
            <span class="text-orange-400 text-xs font-bold">CP</span>
          </div>
          <div>
            <h1 class="text-sm font-bold leading-tight" :class="theme === 'light' ? 'text-gray-900' : 'text-white'">Code Proxy</h1>
            <p class="text-[10px]" :class="theme === 'light' ? 'text-gray-400' : 'text-gray-500'">v1.0.0</p>
          </div>
        </div>
      </div>
      <nav class="flex-1 p-2 space-y-0.5">
        <router-link
          v-for="item in nav" :key="item.path"
          :to="item.path"
          class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors"
          :class="route.path === item.path
            ? 'bg-orange-500/10 text-orange-400 font-medium'
            : theme === 'light'
              ? 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'
              : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'"
        >
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
      <!-- Theme toggle at bottom of sidebar -->
      <div class="p-3 border-t" :class="theme === 'light' ? 'border-gray-200' : 'border-gray-800'">
        <button @click="toggleTheme"
                class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs transition-colors"
                :class="theme === 'light'
                  ? 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'">
          {{ theme === 'light' ? 'Dark Mode' : 'Light Mode' }}
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 p-6 overflow-auto">
      <router-view :theme="theme" />
    </main>
  </div>
</template>
