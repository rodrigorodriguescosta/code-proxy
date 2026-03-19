<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import BarChart from '../components/BarChart.vue'

const props = defineProps({ theme: String })

const accounts = ref([])
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

async function load() {
  loading.value = true
  try {
    accounts.value = await api.getAccountUsage(period.value)
  } catch (e) {
    console.error(e)
    accounts.value = []
  }
  loading.value = false
}

function selectPeriod(p) {
  period.value = p
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
          Account Usage
        </h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Tokens and estimated cost per connected account
        </p>
      </div>

      <div class="flex gap-1 p-1 rounded-lg" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-gray-800'">
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
      <!-- Chart -->
      <div v-if="accounts.length" class="border rounded-xl p-5"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-semibold uppercase tracking-wider"
              :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Usage by Account</h3>
          <div class="flex gap-1 p-1 rounded-lg" :class="props.theme === 'light' ? 'bg-gray-100' : 'bg-gray-800'">
            <button v-for="m in metrics" :key="m.value" @click="chartMetric = m.value"
                    class="px-2.5 py-1 rounded-md text-xs font-medium transition-colors"
                    :class="chartMetric === m.value
                      ? (props.theme === 'light' ? 'bg-white text-gray-900 shadow-sm' : 'bg-gray-700 text-white')
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
        :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'"
      >
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-3"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Details</h3>
        <div class="grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-2 text-xs pb-2 border-b mb-2"
             :class="props.theme === 'light' ? 'text-gray-400 border-gray-200' : 'text-gray-500 border-gray-800'">
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
            :class="props.theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-gray-800/30'"
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
