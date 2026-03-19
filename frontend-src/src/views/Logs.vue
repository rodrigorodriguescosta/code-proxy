<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })

const logs = ref([])
const total = ref(0)
const loading = ref(true)
const page = ref(0)
const limit = 50

async function load() {
  loading.value = true
  const res = await api.listLogs(limit, page.value * limit)
  logs.value = res.data || []
  total.value = res.total || 0
  loading.value = false
}

function nextPage() {
  if ((page.value + 1) * limit < total.value) {
    page.value++
    load()
  }
}

function prevPage() {
  if (page.value > 0) {
    page.value--
    load()
  }
}

function fmtNum(n) {
  if (!n) return '0'
  return n.toLocaleString('en-US')
}

function formatDuration(ms) {
  if (!ms) return '-'
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
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

function effortBadge(effort) {
  switch (effort) {
    case 'high': return { text: 'MAX', cls: 'bg-red-400/10 text-red-400' }
    case 'medium': return { text: 'MED', cls: 'bg-yellow-400/10 text-yellow-400' }
    case 'low': return { text: 'LOW', cls: 'bg-green-400/10 text-green-400' }
    default: return { text: '-', cls: 'text-gray-600' }
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Request Logs</h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Detailed history of all requests
        </p>
      </div>
      <button @click="load"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
              :class="props.theme === 'light'
                ? 'bg-gray-100 hover:bg-gray-200 text-gray-600'
                : 'bg-gray-800 hover:bg-gray-700 text-gray-300'">
        Refresh
      </button>
    </div>

    <div class="border rounded-xl overflow-hidden"
         :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-gray-900 border-gray-800'">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b" :class="props.theme === 'light' ? 'border-gray-100' : 'border-gray-800'">
            <th class="text-left px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Model</th>
            <th class="text-left px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Effort</th>
            <th class="text-right px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">In</th>
            <th class="text-right px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Out</th>
            <th class="text-right px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Duration</th>
            <th class="text-left px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Key</th>
            <th class="text-right px-4 py-3 text-xs uppercase tracking-wider font-medium"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">When</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="7" class="px-4 py-8 text-center text-xs"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">Loading...</td>
          </tr>
          <tr v-else-if="logs.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-xs"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">No logs recorded</td>
          </tr>
          <tr v-for="log in logs" :key="log.id"
              class="border-b transition-colors"
              :class="props.theme === 'light'
                ? 'border-gray-50 hover:bg-gray-50/50'
                : 'border-gray-800/50 hover:bg-gray-800/20'">
            <td class="px-4 py-2.5">
              <span class="font-mono text-xs" :class="props.theme === 'light' ? 'text-gray-800' : 'text-white'">{{ log.model }}</span>
            </td>
            <td class="px-4 py-2.5">
              <span :class="effortBadge(log.effort).cls"
                    class="text-xs px-1.5 py-0.5 rounded font-mono">
                {{ effortBadge(log.effort).text }}
              </span>
            </td>
            <td class="px-4 py-2.5 text-right">
              <span class="text-blue-400 text-xs font-mono">{{ fmtNum(log.input_tokens) }}</span>
            </td>
            <td class="px-4 py-2.5 text-right">
              <span class="text-green-400 text-xs font-mono">{{ fmtNum(log.output_tokens) }}</span>
            </td>
            <td class="px-4 py-2.5 text-right text-xs font-mono"
                :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">{{ formatDuration(log.duration_ms) }}</td>
            <td class="px-4 py-2.5">
              <span class="text-xs truncate block max-w-[100px]"
                    :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
                {{ log.api_key_name || '-' }}
              </span>
            </td>
            <td class="px-4 py-2.5 text-right text-xs"
                :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">{{ timeAgo(log.created_at) }}</td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="total > limit" class="flex items-center justify-between px-4 py-3 border-t"
           :class="props.theme === 'light' ? 'border-gray-100' : 'border-gray-800'">
        <span class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
          {{ page * limit + 1 }}-{{ Math.min((page + 1) * limit, total) }} of {{ fmtNum(total) }}
        </span>
        <div class="flex gap-1">
          <button @click="prevPage" :disabled="page === 0"
                  class="px-3 py-1 rounded text-xs disabled:opacity-30 transition-colors"
                  :class="props.theme === 'light'
                    ? 'bg-gray-100 text-gray-500 hover:text-gray-700'
                    : 'bg-gray-800 text-gray-400 hover:text-white'">
            Previous
          </button>
          <button @click="nextPage" :disabled="(page + 1) * limit >= total"
                  class="px-3 py-1 rounded text-xs disabled:opacity-30 transition-colors"
                  :class="props.theme === 'light'
                    ? 'bg-gray-100 text-gray-500 hover:text-gray-700'
                    : 'bg-gray-800 text-gray-400 hover:text-white'">
            Next
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
