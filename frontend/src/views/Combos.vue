<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({ theme: String })

const combos = ref([])
const loading = ref(true)
const copiedId = ref(null)

// Form state
const showForm = ref(false)
const editingCombo = ref(null)
const formName = ref('')
const formModels = ref([])
const formNameError = ref('')
const saving = ref(false)

// Add model modal
const showModelSelect = ref(false)
const modelSearch = ref('')
const providerModels = ref([])
const loadingModels = ref(false)

const VALID_NAME = /^[a-zA-Z0-9_-]+$/

// Provider catalog for grouping (matches Providers.vue)
const providerCatalog = [
  { type: 'claude-cli', label: 'Claude Code', color: '#D97757', prefix: 'cc' },
  { type: 'codex-cli', label: 'OpenAI Codex', color: '#3B82F6', prefix: 'cx' },
  { type: 'gemini-cli', label: 'Gemini CLI', color: '#4285F4', prefix: 'gc' },
  { type: 'anthropic-api', label: 'Anthropic', color: '#D97757', prefix: 'anthropic' },
  { type: 'openai-api', label: 'OpenAI', color: '#10A37F', prefix: 'openai' },
  { type: 'gemini-api', label: 'Gemini', color: '#4285F4', prefix: 'gemini' },
  { type: 'antigravity', label: 'Antigravity', color: '#F59E0B', prefix: 'ag' },
  { type: 'github-copilot', label: 'GitHub Copilot', color: '#333333', prefix: 'github' },
]

async function load() {
  try {
    const data = await api.listCombos()
    combos.value = data.combos || []
  } catch (e) {
    console.error('Error loading combos:', e)
  } finally {
    loading.value = false
  }
}

async function loadModels() {
  loadingModels.value = true
  try {
    const statuses = await api.getProviderStatuses()
    const groups = []

    for (const status of statuses) {
      if (!status.available) continue
      try {
        const data = await api.getProviderModels(status.type)
        if (data.models?.length) {
          const catalog = providerCatalog.find(c => c.type === status.type)
          groups.push({
            type: status.type,
            name: catalog?.label || status.name || status.type,
            color: catalog?.color || '#6B7280',
            prefix: catalog?.prefix || status.type,
            models: data.models,
          })
        }
      } catch {}
    }

    providerModels.value = groups
  } catch (e) {
    console.error('Error loading models:', e)
  } finally {
    loadingModels.value = false
  }
}

// Filtered combos for the model picker (exclude self when editing)
const combosForPicker = computed(() => {
  if (!editingCombo.value) return combos.value
  return combos.value.filter(c => c.id !== editingCombo.value.id)
})

const filteredCombos = computed(() => {
  if (!modelSearch.value.trim()) return combosForPicker.value
  const q = modelSearch.value.toLowerCase()
  return combosForPicker.value.filter(c => c.name.toLowerCase().includes(q))
})

const filteredProviderModels = computed(() => {
  if (!modelSearch.value.trim()) return providerModels.value
  const q = modelSearch.value.toLowerCase()
  return providerModels.value
    .map(g => ({
      ...g,
      models: g.models.filter(m =>
        (m.id || '').toLowerCase().includes(q) ||
        (m.name || '').toLowerCase().includes(q)
      ),
    }))
    .filter(g => g.models.length > 0 || g.name.toLowerCase().includes(q))
})

function openCreate() {
  editingCombo.value = null
  formName.value = ''
  formModels.value = []
  formNameError.value = ''
  showForm.value = true
}

function openEdit(combo) {
  editingCombo.value = combo
  formName.value = combo.name
  formModels.value = [...combo.models]
  formNameError.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingCombo.value = null
}

function validateName(v) {
  if (!v.trim()) { formNameError.value = 'Name is required'; return false }
  if (!VALID_NAME.test(v)) { formNameError.value = 'Only letters, numbers, - and _ allowed'; return false }
  formNameError.value = ''
  return true
}

function onNameInput(e) {
  formName.value = e.target.value
  if (formName.value) validateName(formName.value)
  else formNameError.value = ''
}

async function saveCombo() {
  if (!validateName(formName.value)) return
  saving.value = true
  try {
    if (editingCombo.value) {
      await api.updateCombo(editingCombo.value.id, { name: formName.value.trim(), models: formModels.value })
    } else {
      await api.createCombo({ name: formName.value.trim(), models: formModels.value })
    }
    closeForm()
    await load()
  } catch (e) {
    alert(e.message || 'Failed to save combo')
  } finally {
    saving.value = false
  }
}

async function deleteCombo(combo) {
  if (!confirm(`Delete combo "${combo.name}"?`)) return
  try {
    await api.deleteCombo(combo.id)
    await load()
  } catch (e) {
    alert(e.message || 'Failed to delete combo')
  }
}

function copyName(combo) {
  navigator.clipboard.writeText(combo.name)
  copiedId.value = combo.id
  setTimeout(() => { copiedId.value = null }, 2000)
}

function moveUp(index) {
  if (index === 0) return
  const m = [...formModels.value];
  [m[index - 1], m[index]] = [m[index], m[index - 1]]
  formModels.value = m
}

function moveDown(index) {
  if (index >= formModels.value.length - 1) return
  const m = [...formModels.value];
  [m[index], m[index + 1]] = [m[index + 1], m[index]]
  formModels.value = m
}

function removeModel(index) {
  formModels.value = formModels.value.filter((_, i) => i !== index)
}

function openModelSelect() {
  modelSearch.value = ''
  showModelSelect.value = true
  if (providerModels.value.length === 0) loadModels()
}

function selectModel(value) {
  if (!formModels.value.includes(value)) {
    formModels.value.push(value)
  }
  showModelSelect.value = false
}

onMounted(load)
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Combos</h2>
        <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
          Model combos with fallback support
        </p>
      </div>
      <button @click="openCreate"
              class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center gap-1.5">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Create Combo
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12"
         :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
      Loading...
    </div>

    <!-- Empty State -->
    <div v-else-if="combos.length === 0"
         class="text-center py-16 border rounded-xl"
         :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
      <div class="inline-flex items-center justify-center w-16 h-16 rounded-full mb-4"
           :class="props.theme === 'light' ? 'bg-blue-50' : 'bg-blue-500/10'">
        <svg class="w-8 h-8 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L12 12.75 6.429 9.75m11.142 0l4.179 2.25L12 17.25 2.25 12l4.179-2.25m11.142 0l4.179 2.25L12 22.5l-9.75-5.25 4.179-2.25" />
        </svg>
      </div>
      <p class="font-medium mb-1" :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">No combos yet</p>
      <p class="text-sm mb-4" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
        Create model combos with automatic fallback
      </p>
      <button @click="openCreate"
              class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
        Create Combo
      </button>
    </div>

    <!-- Combos List -->
    <div v-else class="space-y-3">
      <div v-for="combo in combos" :key="combo.id"
           class="group border rounded-xl p-4 transition-colors"
           :class="props.theme === 'light' ? 'bg-white border-gray-200 hover:border-gray-300' : 'bg-zinc-900 border-zinc-800/40 hover:border-zinc-700'">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <!-- Icon -->
            <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0"
                 :class="props.theme === 'light' ? 'bg-blue-50' : 'bg-blue-500/10'">
              <svg class="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L12 12.75 6.429 9.75m11.142 0l4.179 2.25L12 17.25 2.25 12l4.179-2.25" />
              </svg>
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <code class="text-sm font-medium font-mono truncate"
                      :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">{{ combo.name }}</code>
                <button @click.stop="copyName(combo)"
                        class="p-0.5 rounded transition-colors opacity-0 group-hover:opacity-100"
                        :class="props.theme === 'light' ? 'hover:bg-gray-100 text-gray-400 hover:text-gray-700' : 'hover:bg-zinc-800 text-gray-500 hover:text-gray-300'"
                        title="Copy combo name">
                  <svg v-if="copiedId === combo.id" class="w-3.5 h-3.5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  <svg v-else class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
              <!-- Model badges -->
              <div class="flex items-center gap-1 mt-1 flex-wrap">
                <template v-if="combo.models.length === 0">
                  <span class="text-xs italic" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">No models</span>
                </template>
                <template v-else>
                  <code v-for="(model, i) in combo.models.slice(0, 3)" :key="i"
                        class="text-[10px] font-mono px-1.5 py-0.5 rounded"
                        :class="props.theme === 'light' ? 'bg-gray-100 text-gray-500' : 'bg-zinc-800 text-gray-400'">
                    {{ model }}
                  </code>
                  <span v-if="combo.models.length > 3"
                        class="text-[10px]"
                        :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
                    +{{ combo.models.length - 3 }} more
                  </span>
                </template>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
            <button @click="openEdit(combo)"
                    class="p-1.5 rounded-lg transition-colors"
                    :class="props.theme === 'light' ? 'text-gray-400 hover:bg-gray-100 hover:text-gray-700' : 'text-gray-500 hover:bg-zinc-800 hover:text-gray-300'"
                    title="Edit">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button @click="deleteCombo(combo)"
                    class="p-1.5 rounded-lg transition-colors text-red-500 hover:bg-red-500/10"
                    title="Delete">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <Teleport to="body">
      <div v-if="showForm" class="fixed inset-0 z-50 flex items-center justify-center">
        <div class="absolute inset-0 bg-black/50" @click="closeForm"></div>
        <div class="relative w-full max-w-md mx-4 rounded-xl border shadow-2xl p-6"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800'">
          <!-- Header -->
          <div class="flex items-center justify-between mb-5">
            <h3 class="text-lg font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
              {{ editingCombo ? 'Edit Combo' : 'Create Combo' }}
            </h3>
            <button @click="closeForm" class="p-1 rounded-lg transition-colors"
                    :class="props.theme === 'light' ? 'text-gray-400 hover:bg-gray-100' : 'text-gray-500 hover:bg-zinc-800'">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Name -->
          <div class="mb-4">
            <label class="block text-sm font-medium mb-1.5"
                   :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">Combo Name</label>
            <input :value="formName" @input="onNameInput"
                   placeholder="my-combo"
                   class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                   :class="[
                     props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-zinc-950 border-zinc-800 text-white',
                     formNameError ? 'border-red-500' : ''
                   ]" />
            <p class="text-[11px] mt-1" :class="formNameError ? 'text-red-500' : (props.theme === 'light' ? 'text-gray-400' : 'text-gray-500')">
              {{ formNameError || 'Only letters, numbers, - and _ allowed' }}
            </p>
          </div>

          <!-- Models -->
          <div class="mb-5">
            <label class="block text-sm font-medium mb-1.5"
                   :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">Models</label>

            <!-- Empty state -->
            <div v-if="formModels.length === 0"
                 class="text-center py-4 border border-dashed rounded-lg"
                 :class="props.theme === 'light' ? 'border-gray-200 bg-gray-50/50' : 'border-zinc-800 bg-zinc-950/30'">
              <svg class="w-6 h-6 mx-auto mb-1" :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L12 12.75 6.429 9.75" />
              </svg>
              <p class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">No models added yet</p>
            </div>

            <!-- Model list -->
            <div v-else class="space-y-1 max-h-48 overflow-y-auto">
              <div v-for="(model, index) in formModels" :key="index"
                   class="flex items-center gap-1.5 px-2 py-1.5 rounded-md group/item"
                   :class="props.theme === 'light' ? 'bg-gray-50 hover:bg-gray-100' : 'bg-zinc-950/50 hover:bg-zinc-800/50'">
                <!-- Index -->
                <span class="text-[10px] font-medium w-4 text-center shrink-0"
                      :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">{{ index + 1 }}</span>
                <!-- Model name -->
                <code class="flex-1 text-xs font-mono truncate"
                      :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">{{ model }}</code>
                <!-- Move up/down -->
                <button @click="moveUp(index)" :disabled="index === 0"
                        class="p-0.5 rounded transition-colors"
                        :class="index === 0 ? 'opacity-20 cursor-not-allowed' : (props.theme === 'light' ? 'text-gray-400 hover:text-gray-700' : 'text-gray-500 hover:text-gray-300')">
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" />
                  </svg>
                </button>
                <button @click="moveDown(index)" :disabled="index >= formModels.length - 1"
                        class="p-0.5 rounded transition-colors"
                        :class="index >= formModels.length - 1 ? 'opacity-20 cursor-not-allowed' : (props.theme === 'light' ? 'text-gray-400 hover:text-gray-700' : 'text-gray-500 hover:text-gray-300')">
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                <!-- Remove -->
                <button @click="removeModel(index)"
                        class="p-0.5 rounded transition-colors text-gray-400 hover:text-red-500">
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Add Model button -->
            <button @click="openModelSelect"
                    class="w-full mt-2 py-2 border border-dashed rounded-lg text-xs flex items-center justify-center gap-1 transition-colors"
                    :class="props.theme === 'light'
                      ? 'border-gray-200 text-gray-400 hover:text-blue-600 hover:border-blue-300'
                      : 'border-zinc-800 text-gray-500 hover:text-blue-400 hover:border-blue-500/30'">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
              </svg>
              Add Model
            </button>
          </div>

          <!-- Actions -->
          <div class="flex gap-3">
            <button @click="closeForm"
                    class="flex-1 py-2 rounded-lg text-sm font-medium transition-colors"
                    :class="props.theme === 'light' ? 'bg-gray-100 text-gray-700 hover:bg-gray-200' : 'bg-zinc-800 text-gray-300 hover:bg-zinc-700'">
              Cancel
            </button>
            <button @click="saveCombo"
                    :disabled="!formName.trim() || !!formNameError || saving"
                    class="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white py-2 rounded-lg text-sm font-medium transition-colors">
              {{ saving ? 'Saving...' : (editingCombo ? 'Save' : 'Create') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Add Model Modal -->
    <Teleport to="body">
      <div v-if="showModelSelect" class="fixed inset-0 z-[60] flex items-center justify-center">
        <div class="absolute inset-0 bg-black/50" @click="showModelSelect = false"></div>
        <div class="relative w-full max-w-sm mx-4 rounded-xl border shadow-2xl p-4"
             :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800'">
          <!-- Header -->
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-base font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">
              Add Model to Combo
            </h3>
            <button @click="showModelSelect = false" class="p-1 rounded-lg transition-colors"
                    :class="props.theme === 'light' ? 'text-gray-400 hover:bg-gray-100' : 'text-gray-500 hover:bg-zinc-800'">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Search -->
          <div class="relative mb-3">
            <svg class="w-4 h-4 absolute left-2.5 top-1/2 -translate-y-1/2"
                 :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'"
                 fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input v-model="modelSearch" placeholder="Search..."
                   class="w-full pl-8 pr-3 py-1.5 border rounded-lg text-xs focus:outline-none focus:border-blue-500"
                   :class="props.theme === 'light' ? 'bg-gray-50 border-gray-200 text-gray-900' : 'bg-zinc-950 border-zinc-800 text-white'" />
          </div>

          <!-- Model List -->
          <div class="max-h-72 overflow-y-auto space-y-3">
            <!-- Loading -->
            <div v-if="loadingModels" class="text-center py-6 text-xs"
                 :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">
              Loading models...
            </div>

            <template v-else>
              <!-- Combos section -->
              <div v-if="filteredCombos.length > 0">
                <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 py-0.5"
                     :class="props.theme === 'light' ? 'bg-white' : 'bg-zinc-900'">
                  <svg class="w-3.5 h-3.5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L12 12.75 6.429 9.75" />
                  </svg>
                  <span class="text-xs font-medium text-blue-500">Combos</span>
                  <span class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">({{ filteredCombos.length }})</span>
                </div>
                <div class="flex flex-wrap gap-1.5">
                  <button v-for="c in filteredCombos" :key="c.id"
                          @click="selectModel(c.name)"
                          class="px-2 py-1 rounded-full text-xs font-medium border transition-all cursor-pointer"
                          :class="props.theme === 'light'
                            ? 'bg-white border-gray-200 text-gray-700 hover:border-blue-400 hover:bg-blue-50'
                            : 'bg-zinc-900 border-zinc-700 text-gray-300 hover:border-blue-500/50 hover:bg-blue-500/5'">
                    {{ c.name }}
                  </button>
                </div>
              </div>

              <!-- Provider groups -->
              <div v-for="group in filteredProviderModels" :key="group.type">
                <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 py-0.5"
                     :class="props.theme === 'light' ? 'bg-white' : 'bg-zinc-900'">
                  <div class="w-2 h-2 rounded-full" :style="{ backgroundColor: group.color }"></div>
                  <span class="text-xs font-medium" :class="props.theme === 'light' ? 'text-gray-700' : 'text-gray-300'">{{ group.name }}</span>
                  <span class="text-[10px]" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">({{ group.models.length }})</span>
                </div>
                <div class="flex flex-wrap gap-1.5">
                  <button v-for="model in group.models" :key="model.id"
                          @click="selectModel(group.prefix + '/' + model.id)"
                          class="px-2 py-1 rounded-full text-xs font-medium border transition-all cursor-pointer"
                          :class="props.theme === 'light'
                            ? 'bg-white border-gray-200 text-gray-700 hover:border-blue-400 hover:bg-blue-50'
                            : 'bg-zinc-900 border-zinc-700 text-gray-300 hover:border-blue-500/50 hover:bg-blue-500/5'">
                    {{ model.name || model.id }}
                  </button>
                </div>
              </div>

              <!-- No results -->
              <div v-if="filteredCombos.length === 0 && filteredProviderModels.length === 0"
                   class="text-center py-6">
                <svg class="w-6 h-6 mx-auto mb-1" :class="props.theme === 'light' ? 'text-gray-300' : 'text-gray-600'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <p class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">No models found</p>
              </div>
            </template>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
