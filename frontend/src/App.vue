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
  { path: '/', label: 'Usage', icon: 'M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z' },
  { path: '/providers', label: 'Providers', icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z' },
  { path: '/combos', label: 'Combos', icon: 'M11.99 18.54l-7.37-5.73L3 14.07l9 7 9-7-1.63-1.27-7.38 5.74zM12 16l7.36-5.73L21 9l-9-7-9 7 1.63 1.27L12 16z' },
  { path: '/keys', label: 'API Keys', icon: 'M12.65 10C11.83 7.67 9.61 6 7 6c-3.31 0-6 2.69-6 6s2.69 6 6 6c2.61 0 4.83-1.67 5.65-4H17v4h4v-4h2v-4H12.65zM7 14c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2z' },
  { path: '/accounts', label: 'Account Usage', icon: 'M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zM9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4z' },
  { path: '/tunnel', label: 'Tunnel', icon: 'M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z' },
  { path: '/logs', label: 'Logs', icon: 'M4 6h18V4H4c-1.1 0-2 .9-2 2v11H0v3h14v-3H4V6zm19 2h-6c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h6c.55 0 1-.45 1-1V9c0-.55-.45-1-1-1zm-1 9h-4v-7h4v7z' },
  { path: '/settings', label: 'Settings', icon: 'M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z' },
  { path: '/about', label: 'About', icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z' },
]
</script>

<template>
  <!-- Login gate -->
  <div v-if="needsLogin" class="flex items-center justify-center min-h-screen"
       :class="theme === 'light' ? 'bg-gray-100' : 'bg-zinc-950'">
    <div class="w-80 p-6 rounded-xl border"
         :class="theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
      <div class="flex items-center gap-2 mb-6">
        <img src="/favicon.svg" alt="CP" class="w-8 h-8 rounded-lg" />
        <h1 class="text-lg font-bold" :class="theme === 'light' ? 'text-gray-900' : 'text-white'">Code Proxy</h1>
      </div>
      <form @submit.prevent="doLogin" class="space-y-4">
        <input v-model="loginPassword" type="password" placeholder="Dashboard password"
               class="w-full px-4 py-2.5 rounded-lg border text-sm focus:outline-none focus:border-blue-500"
               :class="theme === 'light'
                 ? 'bg-gray-50 border-gray-200 text-gray-900 placeholder-gray-400'
                 : 'bg-zinc-950 border-zinc-800 text-white placeholder-zinc-600'" />
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
       :class="theme === 'light' ? 'bg-gray-50' : 'bg-zinc-950'">
    <!-- Sidebar -->
    <aside class="w-52 border-r flex flex-col shrink-0"
           :class="theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
      <div class="p-4 border-b" :class="theme === 'light' ? 'border-gray-200' : 'border-zinc-800/50'">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <img src="/favicon.svg" alt="CP" class="w-7 h-7 rounded-lg" />
            <div>
              <h1 class="text-sm font-bold leading-tight" :class="theme === 'light' ? 'text-gray-900' : 'text-white'">Code Proxy</h1>
              <p class="text-[10px]" :class="theme === 'light' ? 'text-gray-400' : 'text-gray-500'">v1.0.0</p>
            </div>
          </div>
          <button @click="toggleTheme"
                  class="p-1.5 rounded-lg transition-colors"
                  :class="theme === 'light'
                    ? 'text-gray-400 hover:bg-gray-100 hover:text-gray-700'
                    : 'text-zinc-500 hover:bg-zinc-800 hover:text-zinc-200'"
                  :title="theme === 'light' ? 'Dark Mode' : 'Light Mode'">
            <svg v-if="theme === 'light'" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
            <svg v-else class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
          </button>
        </div>
      </div>
      <nav class="flex-1 p-2 space-y-0.5">
        <router-link
          v-for="item in nav" :key="item.path"
          :to="item.path"
          class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors"
          :class="route.path === item.path
            ? 'bg-blue-500/10 text-blue-400 font-medium'
            : theme === 'light'
              ? 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'
              : 'text-zinc-500 hover:bg-zinc-900 hover:text-zinc-200'"
        >
          <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="currentColor"><path :d="item.icon" /></svg>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
    </aside>

    <!-- Main content -->
    <main class="flex-1 p-6 overflow-auto">
      <router-view :theme="theme" />
    </main>
  </div>
</template>
