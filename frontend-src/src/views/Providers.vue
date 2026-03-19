<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })

const accounts = ref([])
const loading = ref(true)
const showAddForm = ref(false)
const addProviderType = ref('')
const addAuthMode = ref('apikey')
const addForm = ref({ label: '', api_key: '', base_url: '', metadata: {} })
const oauthState = ref(null)
const oauthCallbackUrl = ref('')
const oauthProviders = ref([])
const expandedModels = ref('')
const providerModels = ref({})
const copiedModelId = ref('')
const editingAccountId = ref('')
const editLabel = ref('')
const providerStatuses = ref([])

// Provider catalog
const catalog = [
  // CLI Tools
  { type: 'claude-cli', label: 'Claude Code', icon: 'C', color: 'orange', category: 'cli', desc: 'Claude via CLI', auth: ['oauth'], oauthName: 'claude', prefix: 'cc' },
  { type: 'codex-cli', label: 'OpenAI Codex', icon: 'X', color: 'emerald', category: 'cli', desc: 'Codex CLI', auth: ['oauth'], oauthName: 'codex', prefix: 'codex' },
  { type: 'gemini-cli', label: 'Gemini CLI', icon: 'G', color: 'blue', category: 'cli', desc: 'Gemini via CLI', auth: ['oauth'], oauthName: 'gemini', prefix: 'gc' },
  // OAuth Providers
  { type: 'anthropic-api', label: 'Anthropic API', icon: 'A', color: 'orange', category: 'oauth', desc: 'Claude via API', auth: ['apikey', 'oauth'], oauthName: 'claude', prefix: 'anthropic' },
  { type: 'openai-api', label: 'OpenAI API', icon: 'O', color: 'emerald', category: 'oauth', desc: 'GPT 5.x, Codex, o3, o4', auth: ['apikey', 'oauth'], oauthName: 'codex', prefix: 'openai' },
  { type: 'gemini-api', label: 'Gemini API', icon: 'G', color: 'blue', category: 'oauth', desc: 'Gemini 2.5 Pro/Flash', auth: ['apikey', 'oauth'], oauthName: 'gemini', prefix: 'gemini' },
  // API key providers (or other OpenAI-compatible endpoints)
  { type: 'generic-openai', label: 'DeepSeek', icon: 'D', color: 'cyan', category: 'other', desc: 'DeepSeek Chat/Coder/Reasoner', auth: ['apikey'], defaultBaseUrl: 'https://api.deepseek.com', prefix: 'deepseek' },
  { type: 'generic-openai', label: 'Groq', icon: 'Q', color: 'purple', category: 'other', desc: 'Llama, Mixtral via Groq', auth: ['apikey'], defaultBaseUrl: 'https://api.groq.com/openai', providerSubtype: 'groq', prefix: 'groq' },
  { type: 'generic-openai', label: 'Together', icon: 'T', color: 'pink', category: 'other', desc: 'Llama, Mistral via Together', auth: ['apikey'], defaultBaseUrl: 'https://api.together.xyz', providerSubtype: 'together', prefix: 'together' },
  { type: 'generic-openai', label: 'Ollama (Local)', icon: 'L', color: 'gray', category: 'other', desc: 'Local models via Ollama', auth: ['apikey'], defaultBaseUrl: 'http://localhost:11434', providerSubtype: 'ollama', prefix: 'ollama' },
  { type: 'generic-openai', label: 'Custom OpenAI', icon: '?', color: 'gray', category: 'other', desc: 'Any OpenAI-compatible API', auth: ['apikey'], prefix: 'generic' },
]

const categories = [
  { key: 'cli', label: 'CLI Tools', desc: 'Detected from local machine' },
  { key: 'oauth', label: 'OAuth Providers', desc: 'Connect via OAuth' },
  { key: 'other', label: 'API Key Providers', desc: 'Connect via API key' },
]

function catalogByCategory(cat) {
  return catalog.filter(c => c.category === cat)
}

function getProviderStatus(providerType) {
  return providerStatuses.value.find(s => s.type === providerType)
}

function accountCount(catalogItem) {
  return accounts.value.filter(a => {
    if (catalogItem.providerSubtype) {
      return a.provider_type === catalogItem.type && a.metadata?.provider_subtype === catalogItem.providerSubtype
    }
    if (catalogItem.type === 'generic-openai' && !catalogItem.providerSubtype) {
      const knownSubtypes = catalog.filter(c => c.providerSubtype).map(c => c.providerSubtype)
      return a.provider_type === 'generic-openai' && !knownSubtypes.includes(a.metadata?.provider_subtype)
    }
    return a.provider_type === catalogItem.type
  }).length
}

const groupedAccounts = computed(() => {
  const groups = {}
  for (const a of accounts.value) {
    const key = a.provider_type
    if (!groups[key]) groups[key] = []
    groups[key].push(a)
  }
  return groups
})

const groupKeys = computed(() => Object.keys(groupedAccounts.value).sort())

function getCatalogInfo(providerType) {
  return catalog.find(c => c.type === providerType) || { label: providerType, icon: '?', color: 'gray' }
}

function supportsOAuth(providerType) {
  const c = catalog.find(c => c.type === providerType)
  return c?.auth?.includes('oauth')
}

function colorClasses(color) {
  // Tailwind não gera classes quando elas são construídas dinamicamente (bg-${...}).
  // Mantemos o mapeamento explícito para garantir que ícones/cores apareçam no build.
  switch (color) {
    case 'orange':
      return 'bg-orange-500/20 text-orange-400'
    case 'emerald':
      return 'bg-emerald-500/20 text-emerald-400'
    case 'blue':
      return 'bg-blue-500/20 text-blue-400'
    case 'cyan':
      return 'bg-cyan-500/20 text-cyan-400'
    case 'purple':
      return 'bg-purple-500/20 text-purple-400'
    case 'pink':
      return 'bg-pink-500/20 text-pink-400'
    case 'gray':
    default:
      return 'bg-gray-500/20 text-gray-400'
  }
}

// --- Actions ---

async function load() {
  accounts.value = await api.listAccounts()
  try { oauthProviders.value = await api.listOAuthProviders() } catch {}
  try { providerStatuses.value = await api.getProviderStatuses() } catch {}
  loading.value = false
}

function openAddForm(catalogItem) {
  addProviderType.value = catalogItem.type
  addAuthMode.value = catalogItem.auth.includes('oauth')
    ? 'oauth'
    : (catalogItem.auth[0] || 'apikey')
  addForm.value = {
    label: '',
    api_key: '',
    base_url: catalogItem.defaultBaseUrl || '',
    metadata: catalogItem.providerSubtype ? { provider_subtype: catalogItem.providerSubtype } : {},
  }
  oauthState.value = null
  oauthCallbackUrl.value = ''
  showAddForm.value = true
}

function closeAddForm() {
  showAddForm.value = false
  oauthState.value = null
}

async function createApiKeyAccount() {
  if (!addForm.value.api_key && addProviderType.value !== 'generic-openai') return
  const data = {
    provider_type: addProviderType.value,
    label: addForm.value.label || 'Account',
    api_key: addForm.value.api_key,
    auth_mode: 'apikey',
  }
  if (addForm.value.base_url || Object.keys(addForm.value.metadata).length > 0) {
    data.metadata = { ...addForm.value.metadata }
    if (addForm.value.base_url) data.metadata.base_url = addForm.value.base_url
  }
  await api.createAccount(data)
  closeAddForm()
  await load()
}

async function startOAuthFlow() {
  oauthState.value = { waiting: true, error: null }
  try {
    const result = await api.startOAuth(addProviderType.value)
    if (!result.auth_url) {
      oauthState.value = { error: 'Backend did not return auth_url. Check logs.', waiting: false }
      return
    }
    oauthState.value = { flowId: result.flow_id, authUrl: result.auth_url, waiting: false, error: null, polling: false }
  } catch (e) {
    const msg = e instanceof Error ? e.message : (typeof e === 'string' ? e : JSON.stringify(e))
    oauthState.value = { error: msg, waiting: false }
  }
}

async function waitForCallback() {
  if (!oauthState.value?.flowId) return
  oauthState.value.polling = true
  oauthState.value.error = null
  try {
    const account = await api.completeOAuth(oauthState.value.flowId, addProviderType.value, null, addForm.value.label)
    if (account.error) {
      oauthState.value.error = account.error
      oauthState.value.polling = false
      return
    }
    closeAddForm()
    await load()
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    oauthState.value.error = msg
    oauthState.value.polling = false
  }
}

async function submitCallbackUrl() {
  if (!oauthCallbackUrl.value || !oauthState.value?.flowId) return
  oauthState.value.polling = true
  oauthState.value.error = null
  try {
    const account = await api.completeOAuth(oauthState.value.flowId, addProviderType.value, oauthCallbackUrl.value, addForm.value.label)
    if (account.error) {
      oauthState.value.error = account.error
      oauthState.value.polling = false
      return
    }
    closeAddForm()
    await load()
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    oauthState.value.error = msg
    oauthState.value.polling = false
  }
}

async function toggleAccount(a) {
  await api.updateAccount(a.id, { is_active: !a.is_active })
  await load()
}

async function refreshAccountToken(a) {
  await api.refreshAccount(a.id)
  await load()
}

async function deleteAccount(a) {
  if (!confirm(`Delete account "${a.label}"?`)) return
  await api.deleteAccount(a.id)
  await load()
}

function formatExpiry(expiresAt) {
  if (!expiresAt) return ''
  const d = new Date(expiresAt)
  if (d.getFullYear() < 2000) return ''
  const now = new Date()
  const diff = d - now
  if (diff < 0) return 'Expired'
  if (diff < 3600000) return `${Math.round(diff / 60000)}min`
  if (diff < 86400000) return `${Math.round(diff / 3600000)}h`
  return `${Math.round(diff / 86400000)}d`
}

function isExpired(a) {
  if (!a.expires_at) return false
  const d = new Date(a.expires_at)
  return d.getFullYear() > 2000 && d < new Date()
}

function isCooldown(a) {
  if (!a.cooldown_until) return false
  return new Date(a.cooldown_until) > new Date()
}

function statusBadge(a) {
  if (!a.is_active) return { text: 'Inactive', cls: 'bg-gray-500/10 text-gray-500' }
  if (isExpired(a)) return { text: 'Expired', cls: 'bg-red-400/10 text-red-400' }
  if (isCooldown(a)) return { text: 'Cooldown', cls: 'bg-yellow-400/10 text-yellow-400' }
  return { text: 'Active', cls: 'bg-green-400/10 text-green-400' }
}

function catalogKey(c) {
  return c.prefix || c.label
}

async function toggleModels(c) {
  const key = catalogKey(c)
  if (expandedModels.value === key) {
    expandedModels.value = ''
    return
  }
  expandedModels.value = key
  if (!providerModels.value[key]) {
    try {
      const data = await api.getProviderModels(c.type)
      const allModels = data.models || []
      const prefix = c.prefix + '/'
      providerModels.value[key] = allModels.filter(m => m.id.startsWith(prefix))
    } catch {
      providerModels.value[key] = []
    }
  }
}

function copyToClipboard(text, modelId) {
  navigator.clipboard.writeText(text)
  if (modelId) {
    copiedModelId.value = modelId
    setTimeout(() => { if (copiedModelId.value === modelId) copiedModelId.value = '' }, 1500)
  }
}

function startEditLabel(a) {
  editingAccountId.value = a.id
  editLabel.value = a.label || ''
}

async function saveAccountLabel() {
  if (!editingAccountId.value) return
  try {
    await api.updateAccount(editingAccountId.value, { label: editLabel.value })
    editingAccountId.value = ''
    await load()
  } catch {
    editingAccountId.value = ''
  }
}

function cancelEditLabel() {
  editingAccountId.value = ''
}

onMounted(load)
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Providers</h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Manage AI providers and connected accounts
        </p>
      </div>
    </div>

    <div v-if="loading" class="text-gray-500 text-center py-8">Loading...</div>

    <template v-else>
      <div class="flex flex-col gap-6">
        <div class="w-full order-2">
          <!-- Catalog by category -->
      <div v-for="cat in categories" :key="cat.key" class="mb-8">
        <div class="flex items-center gap-2 mb-3">
          <h3 class="text-xs font-semibold uppercase tracking-wider"
              :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">{{ cat.label }}</h3>
          <span class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ cat.desc }}</span>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          <div v-for="c in catalogByCategory(cat.key)" :key="catalogKey(c)"
               class="border rounded-xl p-4 transition-colors group"
               :class="[
                 props.theme === 'light' ? 'bg-white' : 'bg-gray-900',
                 expandedModels === catalogKey(c)
                   ? 'border-blue-500/40'
                   : props.theme === 'light' ? 'border-gray-200 hover:border-gray-300' : 'border-gray-800 hover:border-gray-700'
               ]">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg flex items-center justify-center text-sm font-bold shrink-0 cursor-pointer"
                   :class="colorClasses(c.color)"
                   @click="toggleModels(c)">
                {{ c.icon }}
              </div>
              <div class="flex-1 min-w-0 cursor-pointer" @click="toggleModels(c)">
                <div class="flex items-center gap-2">
                  <p class="text-sm font-medium" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ c.label }}</p>
                  <!-- CLI runtime status badge -->
                  <template v-if="c.category === 'cli'">
                    <span v-if="getProviderStatus(c.type)?.available"
                          class="bg-green-400/10 text-green-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                      Detected
                    </span>
                    <span v-else
                          class="bg-red-400/10 text-red-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                      Not Found
                    </span>
                  </template>
                  <!-- Account count badge for API providers -->
                  <span v-else-if="accountCount(c) > 0"
                        class="bg-green-400/10 text-green-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                    {{ accountCount(c) }}
                  </span>
                </div>
                <p class="text-xs truncate" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
                  <span class="font-mono" :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'">{{ c.prefix }}/</span>
                  {{ c.desc }}
                </p>
              </div>
              <div class="flex items-center gap-2">
                <button @click.stop="toggleModels(c)"
                        class="text-[10px] px-1.5 py-0.5 rounded transition-colors"
                        :class="props.theme === 'light'
                          ? 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
                          : 'text-gray-500 hover:text-gray-300 hover:bg-gray-800'">
                  {{ expandedModels === catalogKey(c) ? 'Hide' : 'Models' }}
                </button>
                <button @click.stop="openAddForm(c)"
                        class="text-blue-400 hover:text-blue-300 text-xs px-2 py-0.5 rounded transition-colors"
                        :class="props.theme === 'light' ? 'hover:bg-gray-100' : 'hover:bg-gray-800'">
                  + Add
                </button>
              </div>
            </div>
            <!-- Models list (expanded) -->
            <div v-if="expandedModels === catalogKey(c) && providerModels[catalogKey(c)]"
                 class="mt-3 pt-3 border-t"
                 :class="props.theme === 'light' ? 'border-gray-100' : 'border-gray-800'">
              <div v-if="providerModels[catalogKey(c)].length === 0"
                   class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">No models registered</div>
              <div v-else class="flex flex-wrap gap-1.5">
                <span v-for="m in providerModels[catalogKey(c)]" :key="m.id"
                      class="relative text-[11px] px-2 py-1 rounded-md font-mono cursor-pointer transition-colors"
                      :class="copiedModelId === m.id
                        ? 'bg-green-500/20 text-green-400'
                        : props.theme === 'light'
                          ? 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                          : 'bg-gray-800 text-gray-300 hover:bg-gray-700'"
                      :title="m.id"
                      @click="copyToClipboard(m.id, m.id)">
                  {{ copiedModelId === m.id ? 'Copied!' : m.id }}
                </span>
              </div>
              <p class="text-[10px] mt-2" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">Click to copy model ID</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Add Account Dialog -->
      <div v-if="showAddForm"
           class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
        <div class="w-full max-w-xl border rounded-xl p-5"
             :class="props.theme === 'light' ? 'bg-white border-blue-200' : 'bg-gray-900 border-blue-500/30'">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-sm font-semibold uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
              Add Account &mdash; {{ getCatalogInfo(addProviderType).label }}
            </h3>
            <button @click="closeAddForm"
                    class="text-sm" :class="props.theme === 'light' ? 'text-gray-400 hover:text-gray-600' : 'text-gray-400 hover:text-white'">
              Cancel
            </button>
          </div>

        <!-- Auth mode toggle -->
        <div v-if="supportsOAuth(addProviderType)" class="flex gap-2 mb-4">
          <button @click="addAuthMode = 'apikey'"
                  :class="addAuthMode === 'apikey' ? 'bg-blue-600 text-white' : props.theme === 'light' ? 'bg-gray-100 text-gray-500 hover:text-gray-700' : 'bg-gray-800 text-gray-400 hover:text-white'"
                  class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors">
            API Key
          </button>
          <button @click="addAuthMode = 'oauth'"
                  :class="addAuthMode === 'oauth' ? 'bg-blue-600 text-white' : props.theme === 'light' ? 'bg-gray-100 text-gray-500 hover:text-gray-700' : 'bg-gray-800 text-gray-400 hover:text-white'"
                  class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors">
            OAuth
          </button>
        </div>

        <!-- API Key Form -->
        <div v-if="addAuthMode === 'apikey'" class="space-y-3">
          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Label</label>
            <input v-model="addForm.label" placeholder="e.g. Production, Test"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>
          <div>
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">API Key</label>
            <input v-model="addForm.api_key" type="password" placeholder="sk-..."
                   class="w-full border rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>
          <div v-if="addProviderType === 'generic-openai'">
            <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Base URL</label>
            <input v-model="addForm.base_url" placeholder="https://api.example.com"
                   class="w-full border rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light'
                     ? 'bg-gray-50 border-gray-200 text-gray-900'
                     : 'bg-gray-950 border-gray-700 text-white'" />
          </div>
          <button @click="createApiKeyAccount"
                  class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
            Create Account
          </button>
        </div>

        <!-- OAuth Flow -->
        <div v-if="addAuthMode === 'oauth'" class="space-y-3">
          <div v-if="!oauthState" class="space-y-3">
            <div>
              <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Label (optional)</label>
              <input v-model="addForm.label" placeholder="e.g. Personal, Production"
                     class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                     :class="props.theme === 'light'
                       ? 'bg-gray-50 border-gray-200 text-gray-900'
                       : 'bg-gray-950 border-gray-700 text-white'" />
            </div>
            <p class="text-sm" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              Click to authenticate via OAuth. A window will open for login.
            </p>
            <button @click="startOAuthFlow"
                    class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
              Connect via {{ getCatalogInfo(addProviderType).label }}
            </button>
          </div>

          <div v-else-if="oauthState.waiting" class="text-sm"
               :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
            Starting OAuth flow...
          </div>

          <div v-else-if="oauthState.authUrl && !oauthState.polling">
            <div class="border rounded-lg p-4 mb-4"
                 :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200' : 'bg-gray-950 border-gray-700'">
              <p class="text-sm mb-2" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-300'">Click the link to authenticate:</p>
              <a :href="oauthState.authUrl" target="_blank" rel="noopener"
                 class="text-blue-400 hover:text-blue-300 text-sm underline break-all">
                Open login page
              </a>
              <button @click="copyToClipboard(oauthState.authUrl)"
                      class="text-xs ml-3" :class="props.theme === 'light' ? 'text-gray-400 hover:text-gray-600' : 'text-gray-500 hover:text-gray-300'">
                (copy URL)
              </button>
            </div>

            <p class="text-xs mb-3" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">After authenticating, choose an option:</p>

            <div class="space-y-3">
              <button @click="waitForCallback"
                      class="w-full bg-blue-600 hover:bg-blue-700 text-white px-4 py-2.5 rounded-lg text-sm font-medium transition-colors">
                Wait for Automatic Callback
              </button>
              <div class="flex items-center gap-3">
                <div class="flex-1 border-t" :class="props.theme === 'light' ? 'border-gray-200' : 'border-gray-800'"></div>
                <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">or</span>
                <div class="flex-1 border-t" :class="props.theme === 'light' ? 'border-gray-200' : 'border-gray-800'"></div>
              </div>
              <div>
                <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
                  Paste callback URL (from address bar after login)
                </label>
                <div class="flex gap-2">
                  <input v-model="oauthCallbackUrl" placeholder="http://localhost:54545/callback?code=..."
                         class="flex-1 border rounded-lg px-3 py-2 text-xs font-mono focus:outline-none focus:border-blue-500"
                         :class="props.theme === 'light'
                           ? 'bg-gray-50 border-gray-200 text-gray-900'
                           : 'bg-gray-950 border-gray-700 text-white'" />
                  <button @click="submitCallbackUrl" :disabled="!oauthCallbackUrl"
                          class="px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                          :class="props.theme === 'light'
                            ? 'bg-gray-200 hover:bg-gray-300 text-gray-700'
                            : 'bg-gray-700 hover:bg-gray-600 text-white'">
                    Submit
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="oauthState.polling" class="flex items-center gap-3">
            <div class="w-4 h-4 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"></div>
            <span class="text-sm" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Waiting for authentication...</span>
          </div>

          <div v-if="oauthState?.error" class="bg-red-500/10 border border-red-500/20 rounded-lg p-3 mt-3">
            <p class="text-red-400 text-sm">{{ oauthState.error }}</p>
            <button @click="oauthState = null" class="text-red-400 hover:text-red-300 text-xs mt-1 underline">
              Try again
            </button>
          </div>
          </div>
        </div>
      </div>

      </div>

      <div class="w-full order-1">
        <!-- Connected Accounts -->
        <div class="mt-0 lg:mt-0">
        <h3 class="text-xs font-semibold uppercase tracking-wider mb-3"
            :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Connected Accounts ({{ accounts.length }})
        </h3>

        <div v-if="accounts.length === 0"
             class="text-center py-8 border rounded-xl text-sm"
             :class="props.theme === 'light' ? 'text-gray-400 bg-white border-gray-200' : 'text-gray-500 bg-gray-900 border-gray-800'">
          No connected accounts. Click a provider above to add one.
        </div>

        <div v-else class="space-y-4">
          <div v-for="provType in groupKeys" :key="provType">
            <div class="flex items-center gap-2 mb-2">
              <div class="w-6 h-6 rounded flex items-center justify-center text-[10px] font-bold"
                   :class="colorClasses(getCatalogInfo(provType).color)">
                {{ getCatalogInfo(provType).icon }}
              </div>
              <span class="text-sm font-medium" :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">{{ getCatalogInfo(provType).label }}</span>
              <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">({{ groupedAccounts[provType].length }})</span>
            </div>

            <div class="space-y-1.5">
              <div v-for="a in groupedAccounts[provType]" :key="a.id"
                   class="border rounded-lg px-4 py-3 flex items-center justify-between transition-colors"
                   :class="props.theme === 'light'
                     ? 'bg-white border-gray-200 hover:border-gray-300'
                     : 'bg-gray-900 border-gray-800 hover:border-gray-700'">
                <div class="flex items-center gap-3 min-w-0 flex-1">
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <template v-if="editingAccountId === a.id">
                        <input v-model="editLabel" @keyup.enter="saveAccountLabel" @keyup.escape="cancelEditLabel" @blur="saveAccountLabel"
                               class="border rounded px-2 py-0.5 text-sm font-medium w-40 focus:outline-none"
                               :class="props.theme === 'light'
                                 ? 'bg-white border-blue-400 text-gray-900'
                                 : 'bg-gray-950 border-blue-500 text-white'" />
                      </template>
                      <template v-else>
                        <span class="text-sm font-medium truncate cursor-pointer hover:text-blue-300 transition-colors"
                              :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'"
                              @dblclick="startEditLabel(a)" title="Double-click to edit">
                          {{ a.label || 'Unnamed' }}
                        </span>
                        <button @click="startEditLabel(a)"
                                class="text-[10px] transition-colors"
                                :class="props.theme === 'light' ? 'text-gray-300 hover:text-gray-500' : 'text-gray-600 hover:text-gray-400'"
                                title="Edit name">
                          &#9998;
                        </button>
                      </template>
                      <span class="text-[10px] font-mono uppercase"
                            :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ a.auth_mode }}</span>
                    </div>
                    <div class="flex items-center gap-2 mt-0.5">
                      <span v-if="a.priority > 0" class="text-[10px]"
                            :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">P:{{ a.priority }}</span>
                      <span v-if="a.last_error" class="text-red-400/70 text-[10px] truncate max-w-48" :title="a.last_error">
                        {{ a.last_error }}
                      </span>
                      <span v-if="a.auth_mode === 'oauth' && formatExpiry(a.expires_at)"
                            :class="isExpired(a) ? 'text-red-400' : props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'"
                            class="text-[10px]">
                        Expires: {{ formatExpiry(a.expires_at) }}
                      </span>
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-2 shrink-0 ml-3">
                  <span :class="statusBadge(a).cls" class="text-[10px] px-2 py-0.5 rounded-full font-medium whitespace-nowrap">
                    {{ statusBadge(a).text }}
                  </span>
                  <button v-if="a.auth_mode === 'oauth'" @click="refreshAccountToken(a)"
                          class="text-xs px-1.5 py-1 rounded transition-colors"
                          :class="props.theme === 'light'
                            ? 'text-gray-400 hover:text-blue-500 hover:bg-gray-100'
                            : 'text-gray-500 hover:text-blue-400 hover:bg-gray-800'"
                          title="Refresh token">
                    &#8635;
                  </button>
                  <button @click="toggleAccount(a)"
                          class="text-xs px-2 py-1 rounded transition-colors"
                          :class="props.theme === 'light'
                            ? 'text-gray-400 hover:text-gray-700 hover:bg-gray-100'
                            : 'text-gray-500 hover:text-white hover:bg-gray-800'">
                    {{ a.is_active ? 'Pause' : 'Activate' }}
                  </button>
                  <button @click="deleteAccount(a)"
                          class="text-xs px-1.5 py-1 rounded transition-colors"
                          :class="props.theme === 'light'
                            ? 'text-gray-400 hover:text-red-500 hover:bg-gray-100'
                            : 'text-gray-500 hover:text-red-400 hover:bg-gray-800'">
                    &#10005;
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
        </div>
      </div>
      </div>
    </template>
  </div>
</template>
