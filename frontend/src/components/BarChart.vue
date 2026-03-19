<script setup>
import { computed } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },   // [{ label, value, color?, sub? }]
  theme: { type: String, default: 'dark' },
  maxBars: { type: Number, default: 8 },
  valueFormatter: { type: Function, default: (v) => v.toLocaleString('en-US') },
  barHeight: { type: Number, default: 28 },
})

const maxValue = computed(() => {
  if (!props.items.length) return 1
  return Math.max(...props.items.slice(0, props.maxBars).map(i => i.value)) || 1
})

const visibleItems = computed(() => props.items.slice(0, props.maxBars))

const colors = ['#3b82f6', '#f97316', '#10b981', '#8b5cf6', '#ef4444', '#06b6d4', '#f59e0b', '#ec4899']
</script>

<template>
  <div class="space-y-1.5">
    <div v-for="(item, idx) in visibleItems" :key="idx" class="flex items-center gap-2" :style="{ height: barHeight + 'px' }">
      <div class="w-28 min-w-[7rem] truncate text-xs text-right pr-1"
           :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'"
           :title="item.label">
        {{ item.label }}
      </div>
      <div class="flex-1 h-full flex items-center">
        <div class="h-5 rounded-sm transition-all duration-500 relative group"
             :style="{
               width: Math.max((item.value / maxValue) * 100, 2) + '%',
               backgroundColor: item.color || colors[idx % colors.length],
               opacity: 0.85,
             }">
          <div class="absolute inset-0 rounded-sm opacity-0 group-hover:opacity-100 transition-opacity"
               :style="{ backgroundColor: item.color || colors[idx % colors.length] }"></div>
        </div>
        <span class="ml-2 text-xs font-medium whitespace-nowrap"
              :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">
          {{ valueFormatter(item.value) }}
        </span>
        <span v-if="item.sub" class="ml-1 text-[10px] whitespace-nowrap"
              :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
          {{ item.sub }}
        </span>
      </div>
    </div>
    <p v-if="!items.length" class="text-xs text-center py-4 text-gray-500">No data</p>
  </div>
</template>
