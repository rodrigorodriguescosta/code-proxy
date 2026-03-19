<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })

const status = ref(null)
const tokenInfo = ref(null)
const loading = ref(true)
const toggling = ref(false)
const savingToken = ref(false)
const tokenInput = ref('')
const showTokenInput = ref(false)
let pollInterval = null

async function loadStatus() {
  try {
    status.value = await api.getTunnelStatus()
  } catch (e) {
    console.error(e)
  }
  loading.value = false
}

async function loadTokenInfo() {
  try {
    tokenInfo.value = await api.getTunnelToken()
  } catch (e) {
    console.error(e)
  }
}

async function enableTunnel() {
  toggling.value = true
  try {
    await api.enableTunnel()
    for (let i = 0; i < 30; i++) {
      await new Promise(r => setTimeout(r, 3000))
      await loadStatus()
      if (status.value?.url) break
    }
  } catch (e) {
    console.error(e)
  }
  toggling.value = false
}

async function disableTunnel() {
  toggling.value = true
  try {
    await api.disableTunnel()
    await loadStatus()
  } catch (e) {
    console.error(e)
  }
  toggling.value = false
}

async function saveToken() {
  savingToken.value = true
  try {
    await api.setTunnelToken(tokenInput.value.trim())
    tokenInput.value = ''
    showTokenInput.value = false
    await loadTokenInfo()
    await loadStatus()
  } catch (e) {
    console.error(e)
  }
  savingToken.value = false
}

async function removeToken() {
  savingToken.value = true
  try {
    await api.deleteTunnelToken()
    await loadTokenInfo()
    await loadStatus()
  } catch (e) {
    console.error(e)
  }
  savingToken.value = false
}

function copyURL() {
  if (status.value?.url) {
    navigator.clipboard.writeText(status.value.url)
  }
}

onMounted(() => {
  loadStatus()
  loadTokenInfo()
  pollInterval = setInterval(loadStatus, 10000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold mb-6" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Cloudflare Tunnel</h2>

    <div v-if="loading" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Loading...</div>

    <div v-else class="space-y-6">
      <!-- Status Card -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Status</h3>
          <span :class="status?.running
                  ? (props.theme === 'light' ? 'text-green-700 bg-green-100' : 'text-green-400 bg-green-400/10')
                  : (props.theme === 'light' ? 'text-gray-500 bg-gray-100' : 'text-gray-500 bg-gray-500/10')"
                class="text-xs px-3 py-1 rounded-full font-medium">
            {{ status?.running ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <div v-if="status?.url" class="mb-4">
          <label class="block text-sm mb-2" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Public URL</label>
          <div class="flex items-center gap-2">
            <code class="flex-1 px-4 py-2 rounded-lg text-blue-400 text-sm font-mono truncate"
                  :class="props.theme === 'light' ? 'bg-gray-50' : 'bg-zinc-950'">
              {{ status.url }}
            </code>
            <button @click="copyURL"
                    class="px-3 py-2 rounded-lg text-sm transition-colors"
                    :class="props.theme === 'light'
                      ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                      : 'bg-zinc-800 hover:bg-zinc-700 text-white'">
              Copy
            </button>
          </div>
        </div>

        <div class="flex gap-3">
          <button v-if="!status?.running"
                  @click="enableTunnel"
                  :disabled="toggling"
                  class="bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
            {{ toggling ? 'Starting...' : 'Enable Tunnel' }}
          </button>
          <button v-else
                  @click="disableTunnel"
                  :disabled="toggling"
                  class="bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
            {{ toggling ? 'Stopping...' : 'Disable Tunnel' }}
          </button>
        </div>
      </div>

      <!-- Token Configuration -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-lg font-semibold mb-3" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Tunnel Token (Fixed URL)</h3>
        <p class="text-sm mb-4" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
          Configure a Cloudflare Zero Trust token for a fixed URL that persists across restarts.
        </p>

        <div v-if="tokenInfo?.has_token" class="mb-4">
          <div class="flex items-center gap-3 rounded-lg p-3"
               :class="props.theme === 'light' ? 'bg-green-50 border border-green-200' : 'bg-green-900/20 border border-green-700/30'">
            <span class="text-sm font-medium" :class="props.theme === 'light' ? 'text-green-700' : 'text-green-400'">Token configured</span>
            <code class="text-green-300/70 text-xs font-mono">{{ tokenInfo.masked }}</code>
            <button @click="removeToken"
                    :disabled="savingToken"
                    class="ml-auto bg-red-600/20 hover:bg-red-600/40 text-red-400 px-3 py-1 rounded text-xs transition-colors">
              Remove
            </button>
          </div>
        </div>

        <div v-else class="mb-4">
          <div class="rounded-lg p-3 text-xs border"
               :class="props.theme === 'light'
                 ? 'bg-yellow-50 border-yellow-200 text-yellow-700'
                 : 'bg-yellow-900/20 border-yellow-700/30 text-yellow-300/80'">
            No token configured. The tunnel uses Quick Tunnel with a random URL that changes on each restart.
          </div>
        </div>

        <div v-if="showTokenInput" class="space-y-3">
          <input v-model="tokenInput"
                 type="password"
                 placeholder="Paste your Cloudflare token here (eyJ...)"
                 class="w-full border rounded-lg px-4 py-2 text-sm focus:outline-none focus:border-blue-500"
                 :class="props.theme === 'light'
                   ? 'bg-gray-50 border-gray-200 text-gray-900 placeholder-gray-400'
                   : 'bg-zinc-950 border-zinc-800 text-white placeholder-gray-600'" />
          <div class="flex gap-2">
            <button @click="saveToken"
                    :disabled="savingToken || !tokenInput.trim()"
                    class="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
              {{ savingToken ? 'Saving...' : 'Save Token' }}
            </button>
            <button @click="showTokenInput = false; tokenInput = ''"
                    class="px-4 py-2 rounded-lg text-sm transition-colors"
                    :class="props.theme === 'light'
                      ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                      : 'bg-zinc-800 hover:bg-zinc-700 text-white'">
              Cancel
            </button>
          </div>
        </div>
        <button v-else
                @click="showTokenInput = true"
                class="px-4 py-2 rounded-lg text-sm transition-colors"
                :class="props.theme === 'light'
                  ? 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                  : 'bg-zinc-800 hover:bg-zinc-700 text-white'">
          {{ tokenInfo?.has_token ? 'Change Token' : 'Configure Token' }}
        </button>
      </div>

      <!-- How to use -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-lg font-semibold mb-3" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">How to Use</h3>
        <div class="space-y-3 text-sm" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
          <p>The Cloudflare tunnel creates a public URL for your local proxy, allowing use of Cursor/IDEs from anywhere.</p>

          <div class="rounded-lg p-4 font-mono text-sm"
               :class="props.theme === 'light' ? 'bg-gray-50 text-gray-600' : 'bg-zinc-950 text-gray-300'">
            <p :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'" class="mb-2"># In remote Cursor, configure:</p>
            <p>OpenAI Base URL: <span class="text-blue-400">{{ status?.url || 'https://xxx.trycloudflare.com' }}/v1</span></p>
            <p>API Key: <span class="text-blue-400">(your API key)</span></p>
          </div>

          <div class="rounded-lg p-4 text-sm"
               :class="props.theme === 'light' ? 'bg-gray-50' : 'bg-zinc-950'">
            <p class="font-semibold mb-2" :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">How to get a token for a fixed URL:</p>
            <ol class="list-decimal list-inside space-y-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              <li>Go to <a href="https://one.dash.cloudflare.com" target="_blank" class="text-blue-400 hover:underline">Cloudflare Zero Trust</a></li>
              <li>Navigate to <strong :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">Networks &gt; Tunnels</strong></li>
              <li>Create a new tunnel and copy the token</li>
              <li>Configure the tunnel to point to <code class="text-blue-400">http://localhost:3456</code></li>
              <li>Paste the token above</li>
            </ol>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
