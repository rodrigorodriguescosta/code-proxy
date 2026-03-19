<script setup>
import { ref, onMounted } from 'vue'
import { api, setDashboardToken } from '../api.js'

const props = defineProps({ theme: String })

const hasPassword = ref(false)
const loading = ref(true)

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref('')
const success = ref('')

async function loadStatus() {
  try {
    const status = await api.authStatus()
    hasPassword.value = status.has_password
  } catch {}
  loading.value = false
}

async function setPassword() {
  error.value = ''
  success.value = ''
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  if (newPassword.value.length < 4) {
    error.value = 'Password must be at least 4 characters'
    return
  }
  try {
    await api.setPassword(currentPassword.value, newPassword.value)
    success.value = hasPassword.value ? 'Password updated' : 'Password set successfully'
    hasPassword.value = true
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e) {
    error.value = e.message || 'Failed to set password'
  }
}

async function removePassword() {
  error.value = ''
  success.value = ''
  if (!currentPassword.value) {
    error.value = 'Enter your current password to remove it'
    return
  }
  try {
    await api.removePassword(currentPassword.value)
    success.value = 'Password removed. Dashboard is now open.'
    hasPassword.value = false
    setDashboardToken('')
    currentPassword.value = ''
  } catch (e) {
    error.value = e.message || 'Failed to remove password'
  }
}

onMounted(loadStatus)
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Settings</h2>
      <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
        Dashboard security and preferences
      </p>
    </div>

    <div v-if="loading" class="text-gray-500">Loading...</div>

    <template v-else>
      <!-- Dashboard Password -->
      <div class="border rounded-xl p-6 max-w-lg"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-1"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Dashboard Password</h3>
        <p class="text-xs mb-4" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
          {{ hasPassword ? 'Password protection is enabled. You can change or remove it.' : 'No password set. The dashboard is accessible without authentication.' }}
        </p>

        <div class="space-y-3">
          <div v-if="hasPassword">
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Current Password</label>
            <input v-model="currentPassword" type="password" placeholder="Enter current password"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>

          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              {{ hasPassword ? 'New Password' : 'Password' }}
            </label>
            <input v-model="newPassword" type="password" placeholder="Enter password"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>

          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Confirm Password</label>
            <input v-model="confirmPassword" type="password" placeholder="Confirm password"
                   @keyup.enter="setPassword"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>

          <p v-if="error" class="text-red-400 text-xs">{{ error }}</p>
          <p v-if="success" class="text-green-400 text-xs">{{ success }}</p>

          <div class="flex gap-2 pt-1">
            <button @click="setPassword"
                    class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
              {{ hasPassword ? 'Update Password' : 'Set Password' }}
            </button>
            <button v-if="hasPassword" @click="removePassword"
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
                    :class="props.theme === 'light'
                      ? 'bg-gray-100 hover:bg-gray-200 text-red-500'
                      : 'bg-gray-800 hover:bg-gray-700 text-red-400'">
              Remove Password
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
