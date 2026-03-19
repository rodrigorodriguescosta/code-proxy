<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })

const keys = ref([])
const newKeyName = ref('')
const loading = ref(true)
const copiedId = ref(null)
const requireApiKey = ref(true)
const loadingSettings = ref(true)

async function load() {
  keys.value = await api.listKeys()
  loading.value = false
}

async function loadSettings() {
  try {
    const s = await api.getSettings()
    requireApiKey.value = s.require_api_key !== false
  } catch {}
  loadingSettings.value = false
}

async function toggleRequireKey() {
  requireApiKey.value = !requireApiKey.value
  try {
    await api.updateSettings({ require_api_key: requireApiKey.value ? 'true' : 'false' })
  } catch {
    requireApiKey.value = !requireApiKey.value
  }
}

async function createKey() {
  if (!newKeyName.value.trim()) return
  await api.createKey(newKeyName.value.trim())
  newKeyName.value = ''
  await load()
}

async function toggleKey(key) {
  await api.toggleKey(key.id, !key.is_active)
  await load()
}

async function deleteKey(key) {
  if (!confirm(`Delete key "${key.name}"?`)) return
  await api.deleteKey(key.id)
  await load()
}

function copyKey(key) {
  navigator.clipboard.writeText(key.key)
  copiedId.value = key.id
  setTimeout(() => { copiedId.value = null }, 2000)
}

onMounted(() => {
  load()
  loadSettings()
})
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">API Keys</h2>
      <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
        Manage access keys for the /v1/ API
      </p>
    </div>

    <!-- Require API Key toggle -->
    <div v-if="!loadingSettings" class="border rounded-xl p-4 mb-6 flex items-center justify-between"
         :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
      <div>
        <p class="text-sm font-medium" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Require API Key</p>
        <p class="text-xs mt-0.5" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
          When disabled, /v1/* endpoints accept requests without authentication
        </p>
      </div>
      <button @click="toggleRequireKey"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="requireApiKey ? 'bg-blue-600' : props.theme === 'light' ? 'bg-gray-300' : 'bg-gray-600'">
        <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
              :class="requireApiKey ? 'translate-x-6' : 'translate-x-1'" />
      </button>
    </div>

    <!-- Create Key -->
    <div class="border rounded-xl p-5 mb-6"
         :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
      <h3 class="text-sm font-semibold mb-3 uppercase tracking-wider"
          :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Create New Key</h3>
      <div class="flex gap-3">
        <input v-model="newKeyName" placeholder="Key name (e.g. Cursor, Production)"
               class="flex-1 border rounded-lg px-4 py-2 text-sm focus:outline-none focus:border-blue-500"
               :class="props.theme === 'light'
                 ? 'bg-gray-50 border-gray-200 text-gray-900'
                 : 'bg-zinc-950 border-zinc-800 text-white'"
               @keyup.enter="createKey" />
        <button @click="createKey"
                class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
          Create
        </button>
      </div>
    </div>

    <!-- Keys List -->
    <div class="space-y-3">
      <div v-if="loading" class="text-gray-500 text-center py-8">Loading...</div>
      <div v-else-if="keys.length === 0"
           class="text-center py-8 border rounded-xl"
           :class="props.theme === 'light' ? 'text-gray-400 bg-white border-gray-200' : 'text-gray-500 bg-gray-900 border-zinc-800/50'">
        No keys created
      </div>

      <div v-for="key in keys" :key="key.id"
           class="border rounded-xl p-4"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ key.name }}</span>
            <span :class="key.is_active
                    ? (props.theme === 'light' ? 'bg-green-100 text-green-700' : 'bg-green-400/10 text-green-400')
                    : (props.theme === 'light' ? 'bg-gray-100 text-gray-500' : 'bg-gray-500/10 text-gray-500')"
                  class="text-xs px-2 py-0.5 rounded-full font-medium">
              {{ key.is_active ? 'Active' : 'Paused' }}
            </span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
              {{ new Date(key.created_at).toLocaleDateString('en-US') }}
            </span>
            <button @click="toggleKey(key)"
                    class="text-xs px-2 py-1 rounded transition-colors"
                    :class="props.theme === 'light'
                      ? 'text-gray-400 hover:text-gray-700 hover:bg-gray-100'
                      : 'text-gray-400 hover:text-white hover:bg-zinc-900'">
              {{ key.is_active ? 'Pause' : 'Activate' }}
            </button>
            <button @click="deleteKey(key)"
                    class="text-xs px-2 py-1 rounded transition-colors"
                    :class="props.theme === 'light'
                      ? 'text-gray-400 hover:text-red-500 hover:bg-gray-100'
                      : 'text-gray-400 hover:text-red-400 hover:bg-zinc-900'">
              Delete
            </button>
          </div>
        </div>
        <div v-if="key.key" class="flex items-center gap-2">
          <code class="flex-1 px-3 py-1.5 rounded text-xs font-mono truncate select-all"
                :class="props.theme === 'light' ? 'bg-gray-50 text-gray-600' : 'bg-zinc-950 text-gray-400'">{{ key.key }}</code>
          <button @click="copyKey(key)"
                  class="px-3 py-1.5 rounded text-xs transition-colors shrink-0"
                  :class="props.theme === 'light'
                    ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                    : 'bg-zinc-900 hover:bg-zinc-800 text-white'">
            {{ copiedId === key.id ? 'Copied!' : 'Copy' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
