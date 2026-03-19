<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import BarChart from '../components/BarChart.vue'

const props = defineProps({ theme: String })
const stats = ref(null)
const loading = ref(true)
const period = ref('30d')
const chartMetric = ref('requests')
const expandedKey = ref(null)

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

const modelChartItems = computed(() => {
  if (!stats.value?.top_models) return []
  return stats.value.top_models.map(m => {
    if (chartMetric.value === 'tokens') {
      return { label: m.model, value: m.input_tokens + m.output_tokens, sub: `(${fmtNum(m.input_tokens)}in / ${fmtNum(m.output_tokens)}out)` }
    }
    if (chartMetric.value === 'cost') {
      const cost = modelCost(m)
      return { label: m.model, value: cost }
    }
    return { label: m.model, value: m.count, sub: `${fmtNum(m.input_tokens + m.output_tokens)} tok` }
  })
})

const keyChartItems = computed(() => {
  if (!stats.value?.per_key) return []
  return stats.value.per_key.map(k => {
    if (chartMetric.value === 'tokens') {
      return { label: k.key_name, value: k.input_tokens + k.output_tokens, sub: `(${fmtNum(k.input_tokens)}in / ${fmtNum(k.output_tokens)}out)` }
    }
    if (chartMetric.value === 'cost') {
      return { label: k.key_name, value: k.cost || 0 }
    }
    return { label: k.key_name, value: k.requests, sub: fmtCost(k.cost) }
  })
})

// Group per_key_models by key_id
const keyModelsMap = computed(() => {
  const map = {}
  if (!stats.value?.per_key_models) return map
  for (const km of stats.value.per_key_models) {
    if (!map[km.key_id]) map[km.key_id] = []
    map[km.key_id].push(km)
  }
  return map
})

function keyModelChartItems(keyId) {
  const models = keyModelsMap.value[keyId] || []
  return models.map(m => {
    if (chartMetric.value === 'tokens') {
      return { label: m.model, value: m.input_tokens + m.output_tokens, sub: `(${fmtNum(m.input_tokens)}in / ${fmtNum(m.output_tokens)}out)` }
    }
    if (chartMetric.value === 'cost') {
      return { label: m.model, value: m.cost || 0 }
    }
    return { label: m.model, value: m.requests, sub: fmtCost(m.cost) }
  })
}

function toggleKeyDetail(keyId) {
  expandedKey.value = expandedKey.value === keyId ? null : keyId
}

function modelCost(m) {
  return (m.input_tokens / 1_000_000) * 3 + (m.output_tokens / 1_000_000) * 15
}

function chartValueFmt(v) {
  if (chartMetric.value === 'cost') return fmtCost(v)
  return fmtNum(v)
}

async function load() {
  loading.value = true
  try {
    stats.value = await api.getStats(period.value)
  } catch (e) {
    console.error(e)
  }
  loading.value = false
}

function selectPeriod(p) {
  period.value = p
  load()
}

function fmtNum(n) {
  if (!n) return '0'
  return n.toLocaleString('en-US')
}

function fmtCost(n) {
  if (!n) return '$0.00'
  return '~$' + n.toFixed(4)
}

function timeAgo(t) {
  const diff = Date.now() - new Date(t).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'now'
  if (mins < 60) return mins + 'm'
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return hrs + 'h'
  return Math.floor(hrs / 24) + 'd'
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Usage & Analytics</h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          API usage, token consumption, and request logs
        </p>
      </div>
      <!-- Period selector -->
      <div class="flex gap-1 p-1 rounded-lg" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-gray-800'">
        <button v-for="p in periods" :key="p.value" @click="selectPeriod(p.value)"
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors"
                :class="period === p.value
                  ? 'bg-blue-600 text-white'
                  : props.theme === 'light'
                    ? 'text-gray-500 hover:text-gray-900'
                    : 'text-gray-400 hover:text-white'">
          {{ p.label }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-gray-500">Loading...</div>

    <template v-else-if="stats">
      <!-- Stats Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div class="border rounded-xl p-5"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
          <p class="text-xs uppercase tracking-wider font-medium" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">Total Requests</p>
          <p class="text-3xl font-bold mt-2" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ fmtNum(stats.total_requests) }}</p>
          <p class="text-xs mt-1" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ fmtNum(stats.today_requests) }} today</p>
        </div>
        <div class="border rounded-xl p-5"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
          <p class="text-xs uppercase tracking-wider font-medium" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">Input Tokens</p>
          <p class="text-3xl font-bold text-orange-400 mt-2">{{ fmtNum(stats.total_input_tokens) }}</p>
        </div>
        <div class="border rounded-xl p-5"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
          <p class="text-xs uppercase tracking-wider font-medium" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">Output Tokens</p>
          <p class="text-3xl font-bold mt-2" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ fmtNum(stats.total_output_tokens) }}</p>
        </div>
        <div class="border rounded-xl p-5"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
          <p class="text-xs uppercase tracking-wider font-medium" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">Estimated Cost</p>
          <p class="text-3xl font-bold mt-2" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ fmtCost(stats.estimated_cost) }}</p>
          <p class="text-xs mt-1" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">Estimate, not actual billing</p>
        </div>
      </div>

      <!-- Metric toggle -->
      <div class="flex gap-1 p-1 rounded-lg w-fit mb-4" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-gray-800'">
        <button v-for="m in metrics" :key="m.value" @click="chartMetric = m.value"
                class="px-3 py-1 rounded-md text-xs font-medium transition-colors"
                :class="chartMetric === m.value
                  ? (props.theme === 'light' ? 'bg-white text-gray-900 shadow-sm' : 'bg-gray-700 text-white')
                  : (props.theme === 'light' ? 'text-gray-500 hover:text-gray-900' : 'text-gray-400 hover:text-white')">
          {{ m.label }}
        </button>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left column -->
        <div class="lg:col-span-2 space-y-6">
          <!-- Top Models Chart -->
          <div v-if="stats.top_models?.length" class="border rounded-xl p-5"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
            <h3 class="text-sm font-semibold mb-4 uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Usage by Model</h3>
            <BarChart :items="modelChartItems" :theme="props.theme" :value-formatter="chartValueFmt" />
          </div>

          <!-- Per API Key Chart + Detail -->
          <div v-if="stats.per_key?.length" class="border rounded-xl p-5"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
            <h3 class="text-sm font-semibold mb-4 uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Usage by API Key</h3>
            <BarChart :items="keyChartItems" :theme="props.theme" :value-formatter="chartValueFmt" />

            <!-- Per-key detail: click a key to see model breakdown -->
            <div class="mt-4 pt-4 border-t" :class="props.theme === 'light' ? 'border-gray-100' : 'border-gray-800'">
              <p class="text-[10px] uppercase tracking-wider mb-2"
                 :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">
                Click a key to see model breakdown
              </p>
              <div class="flex flex-wrap gap-1.5">
                <button v-for="k in stats.per_key" :key="k.key_id"
                        @click="toggleKeyDetail(k.key_id)"
                        class="text-xs px-2.5 py-1 rounded-full transition-colors"
                        :class="expandedKey === k.key_id
                          ? 'bg-blue-600 text-white'
                          : props.theme === 'light'
                            ? 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                            : 'bg-gray-800 text-gray-400 hover:bg-gray-700'">
                  {{ k.key_name }}
                </button>
              </div>
              <!-- Expanded key model breakdown -->
              <div v-if="expandedKey && keyModelsMap[expandedKey]?.length" class="mt-3">
                <p class="text-xs font-medium mb-2" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-300'">
                  Models for "{{ stats.per_key.find(k => k.key_id === expandedKey)?.key_name }}"
                </p>
                <BarChart :items="keyModelChartItems(expandedKey)" :theme="props.theme" :value-formatter="chartValueFmt" :bar-height="24" />
              </div>
              <p v-else-if="expandedKey" class="mt-3 text-xs text-gray-500">No model data for this key</p>
            </div>
          </div>

          <!-- Quick Start -->
          <div class="border rounded-xl p-5"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
            <h3 class="text-sm font-semibold mb-3 uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Quick Start</h3>
            <div class="rounded-lg p-4 font-mono text-sm space-y-1"
                 :class="props.theme === 'light' ? 'bg-gray-50 text-gray-700' : 'bg-gray-950 text-gray-300'">
              <p :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'"># Configure in Cursor:</p>
              <p>OpenAI Base URL: <span class="text-blue-400">https://your-public-url/v1</span></p>
              <p>API Key: <span class="text-blue-400">(create one in API Keys tab)</span></p>
              <p>Model: <span class="text-blue-400">cc/claude-opus-4-6:max</span></p>
              <p class="pt-2" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'"># Note: Cursor requires a public URL (not localhost)</p>
              <p :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'"># Use a VPS, Cloudflare tunnel, or similar</p>
            </div>
          </div>
        </div>

        <!-- Right column - Recent Requests -->
        <div class="border rounded-xl p-5 h-fit"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
          <h3 class="text-sm font-semibold mb-4 uppercase tracking-wider"
              :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Recent Requests</h3>
          <div>
            <div class="grid grid-cols-[1fr_auto_auto_auto] gap-2 text-xs pb-2 border-b mb-2"
                 :class="props.theme === 'light' ? 'text-gray-400 border-gray-200' : 'text-gray-500 border-gray-800'">
              <span>Model</span>
              <span class="text-right">Tokens</span>
              <span class="text-right">Cost</span>
              <span class="text-right w-14">When</span>
            </div>
            <template v-if="stats.recent_requests?.length">
              <div v-for="(r, i) in stats.recent_requests" :key="i"
                   class="grid grid-cols-[1fr_auto_auto_auto_auto] gap-2 py-1.5 text-xs items-center">
                <div class="min-w-0">
                  <span class="font-mono truncate block" :class="props.theme === 'light' ? 'text-gray-800' : 'text-white'">{{ r.model }}</span>
                  <span v-if="r.key_name" class="text-[10px] block font-mono" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ r.key_name }}</span>
                  <span v-if="r.key_masked" class="text-[10px] block font-mono" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">{{ r.key_masked }}</span>
                </div>
                <span class="text-right whitespace-nowrap">
                  <span class="text-blue-400">{{ fmtNum(r.input_tokens) }}</span>
                  <span :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'"> / </span>
                  <span class="text-green-400">{{ fmtNum(r.output_tokens) }}</span>
                </span>
                <span class="text-right" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">
                  {{ fmtCost(r.estimated_cost) }}
                </span>
                <span class="text-right w-14" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">{{ timeAgo(r.created_at) }}</span>
              </div>
            </template>
            <p v-else class="text-xs text-center py-4" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">No requests yet</p>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
