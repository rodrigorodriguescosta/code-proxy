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

// Export / Import
const exportLoading = ref(false)
const importLoading = ref(false)
const importMode = ref('merge')
const importResult = ref(null)
const importError = ref('')

async function exportData(includeLogs = false) {
  exportLoading.value = true
  try {
    const data = await api.exportData(includeLogs)
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `code-proxy-backup-${new Date().toISOString().slice(0,10)}.json`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    importError.value = e.message || 'Export failed'
  }
  exportLoading.value = false
}

function triggerImport() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json'
  input.onchange = async (e) => {
    const file = e.target.files[0]
    if (!file) return
    importLoading.value = true
    importError.value = ''
    importResult.value = null
    try {
      const text = await file.text()
      const data = JSON.parse(text)
      const result = await api.importData(data, importMode.value)
      importResult.value = result
    } catch (e) {
      importError.value = e.message || 'Import failed'
    }
    importLoading.value = false
  }
  input.click()
}

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
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
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
                     : 'bg-zinc-950 border-zinc-800 text-white'" />
          </div>

          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              {{ hasPassword ? 'New Password' : 'Password' }}
            </label>
            <input v-model="newPassword" type="password" placeholder="Enter password"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-zinc-950 border-zinc-800 text-white'" />
          </div>

          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Confirm Password</label>
            <input v-model="confirmPassword" type="password" placeholder="Confirm password"
                   @keyup.enter="setPassword"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-zinc-950 border-zinc-800 text-white'" />
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
                      : 'bg-zinc-900 hover:bg-zinc-800 text-red-400'">
              Remove Password
            </button>
          </div>
        </div>
      </div>
      <!-- Export / Import -->
      <div class="border rounded-xl p-6 max-w-lg mt-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-1"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Backup & Restore</h3>
        <p class="text-xs mb-4" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
          Export all data (accounts, API keys, settings) to a JSON file, or import from a backup.
        </p>

        <!-- Export -->
        <div class="mb-4">
          <p class="text-xs font-medium mb-2" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-300'">Export</p>
          <div class="flex gap-2">
            <button @click="exportData(false)" :disabled="exportLoading"
                    class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50">
              {{ exportLoading ? 'Exporting...' : 'Export Data' }}
            </button>
            <button @click="exportData(true)" :disabled="exportLoading"
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
                    :class="props.theme === 'light'
                      ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                      : 'bg-zinc-900 hover:bg-zinc-800 text-gray-300'">
              Export with Logs
            </button>
          </div>
        </div>

        <!-- Import -->
        <div>
          <p class="text-xs font-medium mb-2" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-300'">Import</p>
          <div class="flex items-center gap-3 mb-2">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" v-model="importMode" value="merge" class="accent-blue-500" />
              <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">Merge (skip existing)</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" v-model="importMode" value="replace" class="accent-blue-500" />
              <span class="text-xs text-red-400">Replace (wipe + import)</span>
            </label>
          </div>
          <button @click="triggerImport" :disabled="importLoading"
                  class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
                  :class="props.theme === 'light'
                    ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                    : 'bg-zinc-900 hover:bg-zinc-800 text-gray-300'">
            {{ importLoading ? 'Importing...' : 'Import from File' }}
          </button>
        </div>

        <!-- Import result -->
        <div v-if="importResult" class="mt-3 border rounded-lg p-3 text-xs"
             :class="props.theme === 'light' ? 'bg-green-50 border-green-200 text-green-700' : 'bg-green-500/10 border-green-500/20 text-green-400'">
          <p class="font-medium mb-1">Import complete</p>
          <p v-if="importResult.accounts_imported">Accounts: {{ importResult.accounts_imported }} imported, {{ importResult.accounts_skipped }} skipped</p>
          <p v-if="importResult.api_keys_imported">API Keys: {{ importResult.api_keys_imported }} imported, {{ importResult.api_keys_skipped }} skipped</p>
          <p v-if="importResult.settings_imported">Settings: {{ importResult.settings_imported }} imported, {{ importResult.settings_skipped }} skipped</p>
          <p v-if="importResult.logs_imported">Logs: {{ importResult.logs_imported }} imported, {{ importResult.logs_skipped }} skipped</p>
        </div>
        <p v-if="importError" class="mt-2 text-red-400 text-xs">{{ importError }}</p>
      </div>
    </template>
  </div>
</template>
