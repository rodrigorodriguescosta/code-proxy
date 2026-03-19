<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })
const stats = ref(null)
const loading = ref(true)
const period = ref('30d')

const periods = [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '60d', label: '60 days' },
  { value: '', label: 'All time' },
]

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

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left column -->
        <div class="lg:col-span-2 space-y-6">
          <!-- Top Models -->
          <div v-if="stats.top_models?.length" class="border rounded-xl p-5"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
            <h3 class="text-sm font-semibold mb-3 uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Top Models</h3>
            <div class="space-y-2">
              <div v-for="m in stats.top_models" :key="m.model"
                   class="flex justify-between items-center py-1.5 px-3 rounded-lg"
                   :class="props.theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-gray-800/30'">
                <span class="text-sm font-mono" :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">{{ m.model }}</span>
                <div class="flex items-center gap-4 text-xs">
                  <span class="text-gray-500">{{ m.count }} req</span>
                  <span class="text-blue-400">{{ fmtNum(m.input_tokens) }}in</span>
                  <span class="text-green-400">{{ fmtNum(m.output_tokens) }}out</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Per API Key -->
          <div v-if="stats.per_key?.length" class="border rounded-xl p-5"
               :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
            <h3 class="text-sm font-semibold mb-3 uppercase tracking-wider"
                :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Usage by API Key</h3>
            <div class="space-y-2">
              <div v-for="k in stats.per_key" :key="k.key_id"
                   class="flex items-center justify-between py-2 px-3 rounded-lg"
                   :class="props.theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-gray-800/30'">
                <div>
                  <p class="text-sm font-medium" :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">{{ k.key_name }}</p>
                  <p class="text-[10px] font-mono" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-600'">{{ k.key_masked }}</p>
                </div>
                <div class="flex items-center gap-4 text-xs">
                  <span class="text-gray-500">{{ fmtNum(k.requests) }} req</span>
                  <span class="text-blue-400">{{ fmtNum(k.input_tokens) }}in</span>
                  <span class="text-green-400">{{ fmtNum(k.output_tokens) }}out</span>
                  <span :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">{{ fmtCost(k.cost) }}</span>
                </div>
              </div>
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
              <p>OpenAI Base URL: <span class="text-blue-400">http://localhost:3456/v1</span></p>
              <p>API Key: <span class="text-blue-400">(create one in API Keys tab)</span></p>
              <p>Model: <span class="text-blue-400">cc/claude-opus-4-6:max</span></p>
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
