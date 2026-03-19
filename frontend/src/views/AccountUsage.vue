<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'
import BarChart from '../components/BarChart.vue'

const props = defineProps({ theme: String })

const accounts = ref([])
const allAccounts = ref([])
const loading = ref(true)
const chartMetric = ref('requests')

const period = ref('30d')
const periods = [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '60d', label: '60 days' },
  { value: '', label: 'All time' },
]

const metrics = [
  { value: 'requests', label: 'Requests' },
  { value: 'tokens', label: 'Tokens' },
  { value: 'cost', label: 'Cost' },
]

// Quota state
const quotaData = ref({})
const quotaLoading = ref({})
const quotaAutoRefresh = ref(true)
let quotaInterval = null

// Provider catalog for icons/colors
const providerInfo = {
  'claude-cli': { label: 'Claude Code', icon: '/providers/claude.png', color: '#D97757', textIcon: 'CC' },
  'anthropic-api': { label: 'Anthropic', icon: '/providers/anthropic.png', color: '#D97757', textIcon: 'AN' },
  'codex-cli': { label: 'OpenAI Codex', icon: '/providers/codex.png', color: '#3B82F6', textIcon: 'CX' },
  'openai-api': { label: 'OpenAI', icon: '/providers/openai.png', color: '#10A37F', textIcon: 'OA' },
  'antigravity': { label: 'Antigravity', icon: '/providers/antigravity.png', color: '#F59E0B', textIcon: 'AG' },
  'github-copilot': { label: 'GitHub Copilot', icon: '/providers/github.png', color: '#333333', textIcon: 'GH' },
  'gemini-cli': { label: 'Gemini CLI', icon: '/providers/gemini-cli.png', color: '#4285F4', textIcon: 'GC' },
  'gemini-api': { label: 'Gemini', icon: '/providers/gemini.png', color: '#4285F4', textIcon: 'GE' },
}

// Subscription providers that have quota bars
const quotaProviderTypes = ['claude-cli', 'codex-cli', 'antigravity', 'github-copilot', 'anthropic-api', 'openai-api', 'gemini-cli', 'gemini-api']

const quotaAccounts = computed(() => {
  return allAccounts.value.filter(a =>
    a.is_active && quotaProviderTypes.includes(a.provider_type) && a.auth_mode === 'oauth'
  )
})

const imgErrors = ref({})
function onImgError(key) {
  imgErrors.value[key] = true
}

function getInfo(providerType) {
  return providerInfo[providerType] || { label: providerType, icon: null, color: '#6B7280', textIcon: '?' }
}

const chartItems = computed(() => {
  return accounts.value.map(a => {
    const label = a.label || a.provider_type
    if (chartMetric.value === 'tokens') {
      return { label, value: (a.input_tokens || 0) + (a.output_tokens || 0), sub: `(${fmtNum(a.input_tokens)}in / ${fmtNum(a.output_tokens)}out)` }
    }
    if (chartMetric.value === 'cost') {
      return { label, value: a.estimated_cost || 0 }
    }
    return { label, value: a.requests || 0, sub: fmtCost(a.estimated_cost) }
  })
})

function chartValueFmt(v) {
  if (chartMetric.value === 'cost') return fmtCost(v)
  return fmtNum(v)
}

function fmtNum(n) {
  if (!n && n !== 0) return '0'
  return Number(n).toLocaleString('en-US')
}

function fmtCost(n) {
  if (!n && n !== 0) return '$0.00'
  return '~$' + Number(n).toFixed(4)
}

function timeAgo(iso) {
  if (!iso) return '-'
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'now'
  if (mins < 60) return mins + 'm'
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return hrs + 'h'
  return Math.floor(hrs / 24) + 'd'
}

// Quota helpers
function getQuotaColor(percentage) {
  const light = props.theme === 'light'
  if (percentage > 70) return { bar: 'bg-green-500', barBg: light ? 'bg-green-100' : 'bg-green-500/10', text: light ? 'text-green-700' : 'text-green-500', emoji: '🟢' }
  if (percentage >= 30) return { bar: 'bg-yellow-500', barBg: light ? 'bg-yellow-100' : 'bg-yellow-500/10', text: light ? 'text-yellow-700' : 'text-yellow-500', emoji: '🟡' }
  return { bar: 'bg-red-500', barBg: light ? 'bg-red-100' : 'bg-red-500/10', text: light ? 'text-red-700' : 'text-red-500', emoji: '🔴' }
}

function formatResetCountdown(resetAt) {
  if (!resetAt) return ''
  const now = new Date()
  const reset = new Date(resetAt)
  const diff = reset - now
  if (diff <= 0) return 'now'
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

function formatResetDate(resetAt) {
  if (!resetAt) return ''
  const d = new Date(resetAt)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  const tomorrow = new Date(now.getTime() + 86400000)
  const isTomorrow = d.toDateString() === tomorrow.toDateString()
  const timeStr = d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: false })
  if (isToday) return `Today, ${timeStr}`
  if (isTomorrow) return `Tomorrow, ${timeStr}`
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) + `, ${timeStr}`
}

async function fetchQuota(accountId) {
  quotaLoading.value[accountId] = true
  try {
    const result = await api.getAccountQuota(accountId)
    quotaData.value[accountId] = result
  } catch (e) {
    quotaData.value[accountId] = { error: e.message || 'Failed to fetch quota' }
  }
  quotaLoading.value[accountId] = false
}

async function fetchAllQuotas() {
  const promises = quotaAccounts.value.map(a => fetchQuota(a.id))
  await Promise.allSettled(promises)
}

async function refreshQuota(accountId) {
  await fetchQuota(accountId)
}

async function load() {
  loading.value = true
  try {
    const [usage, accts] = await Promise.all([
      api.getAccountUsage(period.value),
      api.listAccounts(),
    ])
    accounts.value = usage
    allAccounts.value = accts
  } catch (e) {
    console.error(e)
    accounts.value = []
  }
  loading.value = false

  // Fetch quotas for subscription accounts
  await fetchAllQuotas()
}

function selectPeriod(p) {
  period.value = p
  load()
}

function startAutoRefresh() {
  if (quotaInterval) clearInterval(quotaInterval)
  quotaInterval = setInterval(() => {
    if (quotaAutoRefresh.value && !document.hidden) {
      fetchAllQuotas()
    }
  }, 60000)
}

onMounted(() => {
  load()
  startAutoRefresh()
})

onUnmounted(() => {
  if (quotaInterval) clearInterval(quotaInterval)
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
          Account Usage
        </h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Tokens, estimated cost, and subscription quotas
        </p>
      </div>

      <div class="flex gap-1 p-1 rounded-lg" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-zinc-900'">
        <button
          v-for="p in periods"
          :key="p.value"
          @click="selectPeriod(p.value)"
          class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors"
          :class="
            period === p.value
              ? 'bg-blue-600 text-white'
              : props.theme === 'light'
                ? 'text-gray-500 hover:text-gray-900'
                : 'text-gray-400 hover:text-white'
          "
        >
          {{ p.label }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-gray-500">Loading...</div>

    <div v-else class="space-y-6">
      <!-- Subscription Quotas -->
      <div v-if="quotaAccounts.length > 0">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-semibold uppercase tracking-wider"
              :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Subscription Limits</h3>
          <div class="flex items-center gap-2">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" v-model="quotaAutoRefresh" class="w-3 h-3 rounded" />
              <span class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Auto-refresh</span>
            </label>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          <div v-for="a in quotaAccounts" :key="'quota-' + a.id"
               class="border rounded-xl p-4"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
            <!-- Provider header -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2.5">
                <div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 overflow-hidden"
                     :style="{ backgroundColor: getInfo(a.provider_type).color + '15' }">
                  <img v-if="getInfo(a.provider_type).icon && !imgErrors[a.id]"
                       :src="getInfo(a.provider_type).icon" :alt="getInfo(a.provider_type).label"
                       class="w-5 h-5 object-contain rounded"
                       @error="onImgError(a.id)" />
                  <span v-else class="text-[10px] font-bold" :style="{ color: getInfo(a.provider_type).color }">
                    {{ getInfo(a.provider_type).textIcon }}
                  </span>
                </div>
                <div>
                  <p class="text-sm font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
                    {{ a.label || getInfo(a.provider_type).label }}
                  </p>
                  <span v-if="quotaData[a.id]?.plan"
                        class="text-[10px] px-1.5 py-0.5 rounded-full font-medium bg-blue-500/10 text-blue-400">
                    {{ quotaData[a.id].plan }}
                  </span>
                </div>
              </div>
              <button @click="refreshQuota(a.id)"
                      :disabled="quotaLoading[a.id]"
                      class="p-1.5 rounded-lg transition-colors disabled:opacity-50"
                      :class="props.theme === 'light' ? 'hover:bg-gray-100 text-gray-400' : 'hover:bg-zinc-900 text-gray-500'"
                      title="Refresh quota">
                <span :class="quotaLoading[a.id] ? 'animate-spin inline-block' : ''">&#8635;</span>
              </button>
            </div>

            <!-- Loading state -->
            <div v-if="quotaLoading[a.id] && !quotaData[a.id]" class="space-y-3">
              <div class="h-3 rounded animate-pulse" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-zinc-900'"></div>
              <div class="h-2 rounded animate-pulse" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-zinc-900'"></div>
            </div>

            <!-- Error state -->
            <div v-else-if="quotaData[a.id]?.error" class="p-3 rounded-lg bg-red-500/10 border border-red-500/20">
              <p class="text-xs text-red-400">{{ quotaData[a.id].error }}</p>
            </div>

            <!-- Info message (no quota data) -->
            <div v-else-if="quotaData[a.id]?.message && (!quotaData[a.id]?.quotas || quotaData[a.id].quotas.length === 0)"
                 class="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
              <p class="text-xs text-blue-400">{{ quotaData[a.id].message }}</p>
            </div>

            <!-- Quota bars -->
            <div v-else-if="quotaData[a.id]?.quotas?.length > 0" class="space-y-4">
              <div v-for="(q, idx) in quotaData[a.id].quotas" :key="idx">
                <!-- Label + percentage -->
                <div class="flex items-center justify-between text-xs mb-1">
                  <span class="font-semibold" :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-200'">
                    {{ q.name }}
                  </span>
                  <div class="flex items-center gap-1">
                    <span class="text-[10px]">{{ getQuotaColor(q.percentage).emoji }}</span>
                    <span :class="getQuotaColor(q.percentage).text" class="font-medium">
                      {{ Math.round(q.percentage) }}%
                    </span>
                  </div>
                </div>

                <!-- Progress bar -->
                <div v-if="!q.unlimited" class="h-2 rounded-full overflow-hidden" :class="getQuotaColor(q.percentage).barBg">
                  <div class="h-full rounded-full transition-all duration-500"
                       :class="getQuotaColor(q.percentage).bar"
                       :style="{ width: Math.min(q.percentage, 100) + '%' }"></div>
                </div>
                <div v-else class="h-2 rounded-full overflow-hidden bg-blue-500/10">
                  <div class="h-full rounded-full bg-blue-500 w-full"></div>
                </div>

                <!-- Usage details + reset -->
                <div class="flex items-center justify-between text-[10px] mt-1"
                     :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
                  <span>{{ fmtNum(q.used) }} / {{ fmtNum(q.total) }}</span>
                  <span v-if="q.reset_at && formatResetCountdown(q.reset_at)">
                    Reset in {{ formatResetCountdown(q.reset_at) }}
                  </span>
                </div>
                <div v-if="q.reset_at" class="text-[9px] mt-0.5"
                     :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'">
                  {{ formatResetDate(q.reset_at) }}
                </div>
              </div>
            </div>

            <!-- No data yet -->
            <div v-else class="text-center py-4">
              <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">No quota data</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Chart -->
      <div v-if="accounts.length" class="border rounded-xl p-5"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-semibold uppercase tracking-wider"
              :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Usage by Account</h3>
          <div class="flex gap-1 p-1 rounded-lg" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-zinc-900'">
            <button v-for="m in metrics" :key="m.value" @click="chartMetric = m.value"
                    class="px-2.5 py-1 rounded-md text-xs font-medium transition-colors"
                    :class="chartMetric === m.value
                      ? (props.theme === 'light' ? 'bg-white text-gray-900 shadow-sm' : 'bg-zinc-800 text-white')
                      : (props.theme === 'light' ? 'text-gray-500 hover:text-gray-900' : 'text-gray-400 hover:text-white')">
              {{ m.label }}
            </button>
          </div>
        </div>
        <BarChart :items="chartItems" :theme="props.theme" :value-formatter="chartValueFmt" />
      </div>

      <!-- Table -->
      <div
        class="border rounded-xl p-5"
        :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'"
      >
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-3"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Details</h3>
        <div class="grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-2 text-xs pb-2 border-b mb-2"
             :class="props.theme === 'light' ? 'text-gray-400 border-gray-200' : 'text-gray-500 border-zinc-800/50'">
          <span>Account</span>
          <span class="text-right">Provider</span>
          <span class="text-right">Requests</span>
          <span class="text-right">Tokens</span>
          <span class="text-right">Cost</span>
          <span class="text-right">Last Used</span>
        </div>

        <div v-if="accounts.length" class="space-y-2">
          <div
            v-for="a in accounts"
            :key="a.account_id"
            class="grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-2 py-2 text-xs items-center"
            :class="props.theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-zinc-900/30'"
          >
            <div class="min-w-0">
              <span class="block font-medium truncate" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
                {{ a.label || 'Account' }}
              </span>
              <span class="block text-[10px] font-mono text-gray-500" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">
                {{ a.account_id }}
              </span>
            </div>
            <span class="text-right" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">
              {{ a.provider_type }}
            </span>
            <span class="text-right" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">
              {{ fmtNum(a.requests) }}
            </span>
            <span class="text-right whitespace-nowrap">
              <span class="text-blue-400">{{ fmtNum(a.input_tokens) }}</span>
              <span :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'"> / </span>
              <span class="text-green-400">{{ fmtNum(a.output_tokens) }}</span>
            </span>
            <span class="text-right" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">
              {{ fmtCost(a.estimated_cost) }}
            </span>
            <span class="text-right" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
              {{ timeAgo(a.last_used_at) }}
            </span>
          </div>
        </div>

        <div v-else class="text-xs text-center py-6 text-gray-500">
          No accounts found
        </div>
      </div>
    </div>
  </div>
</template>
