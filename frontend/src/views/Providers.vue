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
const imgErrors = ref({})
const testingAccount = ref('')
const testResults = ref({})

// CLI Tools (auto-detected from environment, proxy communicates with local CLIs)
const cliCatalog = [
  { type: 'claude-cli', label: 'Claude Code', icon: '/providers/claude.png', color: '#D97757', textIcon: 'CC', desc: 'Detected from local machine', auth: [], prefix: 'cli-cc', category: 'cli' },
  { type: 'codex-cli', label: 'OpenAI Codex', icon: '/providers/codex.png', color: '#3B82F6', textIcon: 'CX', desc: 'Detected from local machine', auth: [], prefix: 'cli-codex', category: 'cli' },
  { type: 'gemini-cli', label: 'Gemini CLI', icon: '/providers/gemini-cli.png', color: '#4285F4', textIcon: 'GC', desc: 'Detected from local machine', auth: [], prefix: 'cli-gc', category: 'cli' },
]

// OAuth Providers (subscription-based tokens via OAuth, like 9router)
const oauthCatalog = [
  { type: 'claude-cli', label: 'Claude Code', icon: '/providers/claude.png', color: '#D97757', textIcon: 'CC', desc: 'Claude Max subscription', auth: ['oauth'], oauthName: 'claude', prefix: 'cc', category: 'oauth' },
  { type: 'codex-cli', label: 'OpenAI Codex', icon: '/providers/codex.png', color: '#3B82F6', textIcon: 'CX', desc: 'Codex Pro subscription', auth: ['oauth'], oauthName: 'codex', prefix: 'codex', category: 'oauth' },
  { type: 'gemini-cli', label: 'Gemini CLI', icon: '/providers/gemini-cli.png', color: '#4285F4', textIcon: 'GC', desc: 'Google AI subscription', auth: ['oauth'], oauthName: 'gemini', prefix: 'gc', category: 'oauth' },
  { type: 'antigravity', label: 'Antigravity', icon: '/providers/antigravity.png', color: '#F59E0B', textIcon: 'AG', desc: 'Google Cloud Code', auth: ['oauth'], oauthName: 'antigravity', prefix: 'ag', category: 'oauth' },
  { type: 'github-copilot', label: 'GitHub Copilot', icon: '/providers/github.png', color: '#333333', textIcon: 'GH', desc: 'GitHub Copilot subscription', auth: ['oauth'], oauthName: 'github', prefix: 'github', category: 'oauth' },
]

// API Key Providers (pay-per-token)
const apikeyCatalog = [
  { type: 'anthropic-api', label: 'Anthropic', icon: '/providers/anthropic.png', color: '#D97757', textIcon: 'AN', desc: 'Claude Sonnet, Opus, Haiku', auth: ['apikey'], prefix: 'anthropic', category: 'apikey' },
  { type: 'openai-api', label: 'OpenAI', icon: '/providers/openai.png', color: '#10A37F', textIcon: 'OA', desc: 'GPT-5, o3, o4-mini', auth: ['apikey'], prefix: 'openai', category: 'apikey' },
  { type: 'gemini-api', label: 'Gemini', icon: '/providers/gemini.png', color: '#4285F4', textIcon: 'GE', desc: 'Gemini 2.5 Pro/Flash', auth: ['apikey'], prefix: 'gemini', category: 'apikey' },
  { type: 'generic-openai', label: 'DeepSeek', icon: '/providers/deepseek.png', color: '#4D6BFE', textIcon: 'DS', desc: 'DeepSeek Chat/Coder/Reasoner', auth: ['apikey'], defaultBaseUrl: 'https://api.deepseek.com', prefix: 'deepseek', category: 'apikey' },
  { type: 'generic-openai', label: 'Groq', icon: '/providers/groq.png', color: '#F55036', textIcon: 'GQ', desc: 'Llama, Mixtral via Groq', auth: ['apikey'], defaultBaseUrl: 'https://api.groq.com/openai', providerSubtype: 'groq', prefix: 'groq', category: 'apikey' },
  { type: 'generic-openai', label: 'xAI (Grok)', icon: '/providers/xai.png', color: '#1DA1F2', textIcon: 'XA', desc: 'Grok models', auth: ['apikey'], defaultBaseUrl: 'https://api.x.ai', providerSubtype: 'xai', prefix: 'xai', category: 'apikey' },
  { type: 'generic-openai', label: 'Mistral', icon: '/providers/mistral.png', color: '#FF7000', textIcon: 'MI', desc: 'Mistral, Codestral', auth: ['apikey'], defaultBaseUrl: 'https://api.mistral.ai', providerSubtype: 'mistral', prefix: 'mistral', category: 'apikey' },
  { type: 'generic-openai', label: 'OpenRouter', icon: '/providers/openrouter.png', color: '#F97316', textIcon: 'OR', desc: 'Multi-provider gateway', auth: ['apikey'], defaultBaseUrl: 'https://openrouter.ai/api', providerSubtype: 'openrouter', prefix: 'openrouter', category: 'apikey' },
  { type: 'generic-openai', label: 'Together AI', icon: '/providers/together.png', color: '#0F6FFF', textIcon: 'TG', desc: 'Llama, Mixtral via Together', auth: ['apikey'], defaultBaseUrl: 'https://api.together.xyz', providerSubtype: 'together', prefix: 'together', category: 'apikey' },
  { type: 'generic-openai', label: 'Fireworks AI', icon: '/providers/fireworks.png', color: '#7B2EF2', textIcon: 'FW', desc: 'Fast inference', auth: ['apikey'], defaultBaseUrl: 'https://api.fireworks.ai/inference', providerSubtype: 'fireworks', prefix: 'fireworks', category: 'apikey' },
  { type: 'generic-openai', label: 'Cerebras', icon: '/providers/cerebras.png', color: '#FF4F00', textIcon: 'CB', desc: 'Ultra-fast inference', auth: ['apikey'], defaultBaseUrl: 'https://api.cerebras.ai', providerSubtype: 'cerebras', prefix: 'cerebras', category: 'apikey' },
  { type: 'generic-openai', label: 'Perplexity', icon: '/providers/perplexity.png', color: '#20808D', textIcon: 'PP', desc: 'Search-augmented AI', auth: ['apikey'], defaultBaseUrl: 'https://api.perplexity.ai', providerSubtype: 'perplexity', prefix: 'perplexity', category: 'apikey' },
  { type: 'generic-openai', label: 'NVIDIA NIM', icon: '/providers/nvidia.png', color: '#76B900', textIcon: 'NV', desc: 'NVIDIA inference', auth: ['apikey'], defaultBaseUrl: 'https://integrate.api.nvidia.com', providerSubtype: 'nvidia', prefix: 'nvidia', category: 'apikey' },
  { type: 'generic-openai', label: 'Cohere', icon: '/providers/cohere.png', color: '#39594D', textIcon: 'CO', desc: 'Command models', auth: ['apikey'], defaultBaseUrl: 'https://api.cohere.com', providerSubtype: 'cohere', prefix: 'cohere', category: 'apikey' },
  { type: 'generic-openai', label: 'Ollama (Local)', icon: null, color: '#6B7280', textIcon: 'OL', desc: 'Local models via Ollama', auth: ['apikey'], defaultBaseUrl: 'http://localhost:11434', providerSubtype: 'ollama', prefix: 'ollama', category: 'apikey' },
  { type: 'generic-openai', label: 'Custom Endpoint', icon: null, color: '#6B7280', textIcon: '?', desc: 'Any OpenAI-compatible API', auth: ['apikey'], prefix: 'generic', category: 'apikey' },
]

const allCatalog = [...cliCatalog, ...oauthCatalog, ...apikeyCatalog]

const categories = [
  { key: 'cli', label: 'CLI Tools', desc: 'Detected from local machine', catalog: cliCatalog },
  { key: 'oauth', label: 'OAuth Providers', desc: 'Connect via subscription tokens', catalog: oauthCatalog },
  { key: 'apikey', label: 'API Key Providers', desc: 'Connect via API key', catalog: apikeyCatalog },
]

function getProviderStatus(providerType) {
  return providerStatuses.value.find(s => s.type === providerType)
}

function accountsForCatalog(catalogItem) {
  return accounts.value.filter(a => {
    // CLI section: no accounts shown (they're auto-detected)
    if (catalogItem.category === 'cli') return false
    // OAuth section: only show oauth accounts for this provider type
    if (catalogItem.category === 'oauth') {
      return a.provider_type === catalogItem.type && a.auth_mode === 'oauth'
    }
    // API Key section
    if (catalogItem.providerSubtype) {
      return a.provider_type === catalogItem.type && a.metadata?.provider_subtype === catalogItem.providerSubtype
    }
    if (catalogItem.type === 'generic-openai' && !catalogItem.providerSubtype) {
      const knownSubtypes = allCatalog.filter(c => c.providerSubtype).map(c => c.providerSubtype)
      return a.provider_type === 'generic-openai' && !knownSubtypes.includes(a.metadata?.provider_subtype)
    }
    // For apikey category, only show apikey accounts
    if (catalogItem.category === 'apikey') {
      return a.provider_type === catalogItem.type && a.auth_mode !== 'oauth'
    }
    return a.provider_type === catalogItem.type
  })
}

function accountCount(catalogItem) {
  return accountsForCatalog(catalogItem).length
}

function getCatalogInfo(providerType) {
  return allCatalog.find(c => c.type === providerType) || { label: providerType, icon: null, color: '#6B7280', textIcon: '?' }
}

function supportsOAuth(providerType) {
  const c = allCatalog.find(c => c.type === providerType)
  return c?.auth?.includes('oauth')
}

function catalogKey(c) {
  return c.prefix || c.label
}

function onImgError(key) {
  imgErrors.value[key] = true
}

// Provider stats: connected / errors
function getProviderStats(catalogItem) {
  const accts = accountsForCatalog(catalogItem)
  const connected = accts.filter(a => a.is_active && !isExpired(a) && !isCooldown(a)).length
  const errors = accts.filter(a => a.last_error || isExpired(a) || isCooldown(a)).length
  return { connected, errors, total: accts.length }
}

// CLI status from backend provider statuses
function getCliStatus(catalogItem) {
  const status = providerStatuses.value.find(s => s.type === catalogItem.type)
  if (!status) return { detected: false }
  return { detected: status.available || false, version: status.version || '' }
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
  addAuthMode.value = catalogItem.auth.includes('oauth') && !catalogItem.auth.includes('apikey')
    ? 'oauth'
    : catalogItem.auth.includes('oauth') ? 'oauth' : 'apikey'
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

async function doTestAccount(a) {
  testingAccount.value = a.id
  testResults.value[a.id] = null
  try {
    const result = await api.testAccount(a.id)
    testResults.value[a.id] = result
  } catch (e) {
    testResults.value[a.id] = { valid: false, error: e.message || 'Test failed' }
  }
  testingAccount.value = ''
  await load() // Refresh to pick up any status changes
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
      <!-- Categories: CLI Tools, OAuth Providers, API Key Providers -->
      <div v-for="cat in categories" :key="cat.key" class="mb-8">
        <div class="flex items-center gap-2 mb-3">
          <h3 class="text-lg font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ cat.label }}</h3>
          <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ cat.desc }}</span>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          <div v-for="c in cat.catalog" :key="cat.key + '-' + catalogKey(c)"
               class="border rounded-xl p-4 transition-all group"
               :class="[
                 props.theme === 'light' ? 'bg-white' : 'bg-gray-900',
                 expandedModels === catalogKey(c)
                   ? 'border-blue-500/40 ring-1 ring-blue-500/20'
                   : props.theme === 'light' ? 'border-gray-200 hover:border-gray-300 hover:shadow-sm' : 'border-gray-800 hover:border-gray-700'
               ]">
            <!-- Card header -->
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 overflow-hidden cursor-pointer"
                   :style="{ backgroundColor: c.color + '15' }"
                   @click="toggleModels(c)">
                <img v-if="c.icon && !imgErrors[catalogKey(c)]"
                     :src="c.icon" :alt="c.label"
                     class="w-7 h-7 object-contain rounded"
                     @error="onImgError(catalogKey(c))" />
                <span v-else class="text-sm font-bold" :style="{ color: c.color }">{{ c.textIcon }}</span>
              </div>
              <div class="flex-1 min-w-0 cursor-pointer"
                   @click="toggleModels(c)">
                <div class="flex items-center gap-2">
                  <p class="text-sm font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ c.label }}</p>
                </div>
                <div class="flex items-center gap-1.5 mt-0.5">
                  <!-- CLI: show detected / not detected -->
                  <template v-if="cat.key === 'cli'">
                    <span v-if="getCliStatus(c).detected"
                          class="bg-green-400/10 text-green-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                      Detected{{ getCliStatus(c).version ? ' (' + getCliStatus(c).version + ')' : '' }}
                    </span>
                    <span v-else
                          class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">Not detected</span>
                  </template>
                  <!-- OAuth / API Key: show connected accounts -->
                  <template v-else>
                    <template v-if="getProviderStats(c).connected > 0">
                      <span class="bg-green-400/10 text-green-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                        {{ getProviderStats(c).connected }} Connected
                      </span>
                    </template>
                    <template v-if="getProviderStats(c).errors > 0">
                      <span class="bg-red-400/10 text-red-400 text-[10px] px-1.5 py-0.5 rounded-full font-medium">
                        {{ getProviderStats(c).errors }} Error
                      </span>
                    </template>
                    <span v-if="getProviderStats(c).total === 0"
                          class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ c.desc }}</span>
                  </template>
                </div>
              </div>
              <!-- Add button: only for OAuth and API Key sections -->
              <button v-if="cat.key !== 'cli'"
                      @click.stop="openAddForm(c)"
                      class="text-xs px-2.5 py-1 rounded-lg font-medium transition-colors shrink-0 opacity-0 group-hover:opacity-100"
                      :class="props.theme === 'light'
                        ? 'text-blue-600 hover:bg-blue-50'
                        : 'text-blue-400 hover:bg-blue-500/10'">
                + Add
              </button>
            </div>

            <!-- Connected accounts inline (not for CLI) -->
            <div v-if="cat.key !== 'cli' && accountsForCatalog(c).length > 0" class="mt-3 pt-3 border-t space-y-1.5"
                 :class="props.theme === 'light' ? 'border-gray-100' : 'border-gray-800'">
              <div v-for="a in accountsForCatalog(c)" :key="a.id"
                   class="flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg transition-colors"
                   :class="props.theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-gray-800/50'">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5">
                    <template v-if="editingAccountId === a.id">
                      <input v-model="editLabel" @keyup.enter="saveAccountLabel" @keyup.escape="cancelEditLabel" @blur="saveAccountLabel"
                             class="border rounded px-2 py-0.5 text-xs font-medium w-32 focus:outline-none"
                             :class="props.theme === 'light' ? 'bg-white border-blue-400 text-gray-900' : 'bg-gray-950 border-blue-500 text-white'" />
                    </template>
                    <template v-else>
                      <span class="text-xs font-medium truncate cursor-pointer"
                            :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'"
                            @dblclick="startEditLabel(a)">{{ a.label || 'Unnamed' }}</span>
                    </template>
                    <span :class="statusBadge(a).cls" class="text-[9px] px-1.5 py-0.5 rounded-full font-medium whitespace-nowrap">
                      {{ statusBadge(a).text }}
                    </span>
                  </div>
                  <!-- Test result -->
                  <div v-if="testResults[a.id]" class="mt-0.5">
                    <span v-if="testResults[a.id].valid"
                          class="text-[10px] text-green-400 font-medium">
                      &#10003; Connected ({{ testResults[a.id].latency_ms }}ms)
                    </span>
                    <span v-else
                          class="text-[10px] text-red-400 font-medium" :title="testResults[a.id].error">
                      &#10007; {{ testResults[a.id].error }}
                    </span>
                  </div>
                  <div v-else-if="a.last_error" class="mt-0.5">
                    <span class="text-[10px] text-red-400/70 truncate block max-w-[200px]" :title="a.last_error">{{ a.last_error }}</span>
                  </div>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <button @click="doTestAccount(a)"
                          :disabled="testingAccount === a.id"
                          class="text-[10px] px-1.5 py-0.5 rounded transition-colors disabled:opacity-50"
                          :class="testingAccount === a.id
                            ? 'text-blue-400 animate-pulse'
                            : props.theme === 'light'
                              ? 'text-gray-400 hover:text-blue-500 hover:bg-gray-100'
                              : 'text-gray-500 hover:text-blue-400 hover:bg-gray-800'"
                          :title="testingAccount === a.id ? 'Testing...' : 'Test connection'">
                    {{ testingAccount === a.id ? '...' : 'Test' }}
                  </button>
                  <button v-if="a.auth_mode === 'oauth'" @click="refreshAccountToken(a)"
                          class="text-[10px] px-1 py-0.5 rounded transition-colors"
                          :class="props.theme === 'light' ? 'text-gray-400 hover:text-blue-500 hover:bg-gray-100' : 'text-gray-500 hover:text-blue-400 hover:bg-gray-800'"
                          title="Refresh token">&#8635;</button>
                  <button @click="toggleAccount(a)"
                          class="text-[10px] px-1.5 py-0.5 rounded transition-colors"
                          :class="props.theme === 'light' ? 'text-gray-400 hover:text-gray-700 hover:bg-gray-100' : 'text-gray-500 hover:text-white hover:bg-gray-800'">
                    {{ a.is_active ? 'Pause' : 'On' }}
                  </button>
                  <button @click="deleteAccount(a)"
                          class="text-[10px] px-1 py-0.5 rounded transition-colors"
                          :class="props.theme === 'light' ? 'text-gray-400 hover:text-red-500 hover:bg-gray-100' : 'text-gray-500 hover:text-red-400 hover:bg-gray-800'">
                    &#10005;
                  </button>
                </div>
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
                      @click.stop="copyToClipboard(m.id, m.id)">
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

          <!-- Auth mode toggle (only for providers that support both) -->
          <div v-if="supportsOAuth(addProviderType) && getCatalogInfo(addProviderType).auth?.includes('apikey')" class="flex gap-2 mb-4">
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
                     :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-gray-950 border-gray-700 text-white'" />
            </div>
            <div>
              <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">API Key</label>
              <input v-model="addForm.api_key" type="password" placeholder="sk-..."
                     class="w-full border rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
                     :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-gray-950 border-gray-700 text-white'" />
            </div>
            <div v-if="addProviderType === 'generic-openai'">
              <label class="block text-xs mb-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">Base URL</label>
              <input v-model="addForm.base_url" placeholder="https://api.example.com"
                     class="w-full border rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
                     :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-gray-950 border-gray-700 text-white'" />
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
                       :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-gray-950 border-gray-700 text-white'" />
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
                   class="text-blue-400 hover:text-blue-300 text-sm underline break-all">Open login page</a>
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
                    <input v-model="oauthCallbackUrl" placeholder="http://localhost:.../callback?code=..."
                           class="flex-1 border rounded-lg px-3 py-2 text-xs font-mono focus:outline-none focus:border-blue-500"
                           :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-gray-950 border-gray-700 text-white'" />
                    <button @click="submitCallbackUrl" :disabled="!oauthCallbackUrl"
                            class="px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                            :class="props.theme === 'light' ? 'bg-gray-200 hover:bg-gray-300 text-gray-700' : 'bg-gray-700 hover:bg-gray-600 text-white'">
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
              <button @click="oauthState = null" class="text-red-400 hover:text-red-300 text-xs mt-1 underline">Try again</button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
