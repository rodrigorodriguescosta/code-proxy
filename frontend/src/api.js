const BASE = ''

// Dashboard session token
let dashboardToken = localStorage.getItem('dashboard_token') || ''

export function setDashboardToken(token) {
  dashboardToken = token
  if (token) {
    localStorage.setItem('dashboard_token', token)
  } else {
    localStorage.removeItem('dashboard_token')
  }
}

async function request(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...options.headers }
  if (dashboardToken) {
    headers['X-Dashboard-Token'] = dashboardToken
  }
  const res = await fetch(BASE + path, { headers, ...options })
  let data
  try {
    data = await res.json()
  } catch {
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
    return {}
  }
  if (!res.ok) {
    const msg = typeof data?.error === 'string' ? data.error : (data?.error?.message || JSON.stringify(data))
    throw new Error(msg)
  }
  return data
}

export const api = {
  // Stats
  getStats: (period) => request('/api/stats' + (period ? `?period=${period}` : '')),

  // API Keys
  listKeys: () => request('/api/keys'),
  createKey: (name) => request('/api/keys', { method: 'POST', body: JSON.stringify({ name }) }),
  toggleKey: (id, active) => request(`/api/keys/${id}`, { method: 'PUT', body: JSON.stringify({ is_active: active }) }),
  deleteKey: (id) => request(`/api/keys/${id}`, { method: 'DELETE' }),

  // Providers
  listProviders: () => request('/api/providers'),
  createProvider: (data) => request('/api/providers', { method: 'POST', body: JSON.stringify(data) }),
  updateProvider: (id, data) => request(`/api/providers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteProvider: (id) => request(`/api/providers/${id}`, { method: 'DELETE' }),
  getProviderModels: (type) => request(`/api/providers/${type}/models`),
  getProviderStatuses: () => request('/api/providers/status'),

  // Settings
  getSettings: () => request('/api/settings'),
  updateSettings: (data) => request('/api/settings', { method: 'PUT', body: JSON.stringify(data) }),

  // Logs
  listLogs: (limit = 50, offset = 0) => request(`/api/logs?limit=${limit}&offset=${offset}`),

  // Tunnel
  getTunnelStatus: () => request('/api/tunnel/status'),
  enableTunnel: (token) => request('/api/tunnel/enable', { method: 'POST', body: JSON.stringify(token ? { token } : {}) }),
  disableTunnel: () => request('/api/tunnel/disable', { method: 'POST' }),
  getTunnelToken: () => request('/api/tunnel/token'),
  setTunnelToken: (token) => request('/api/tunnel/token', { method: 'PUT', body: JSON.stringify({ token }) }),
  deleteTunnelToken: () => request('/api/tunnel/token', { method: 'DELETE' }),

  // Accounts
  listAccounts: (providerType) => request('/api/accounts' + (providerType ? `?provider_type=${providerType}` : '')),
  createAccount: (data) => request('/api/accounts', { method: 'POST', body: JSON.stringify(data) }),
  updateAccount: (id, data) => request(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteAccount: (id) => request(`/api/accounts/${id}`, { method: 'DELETE' }),
  refreshAccount: (id) => request(`/api/accounts/${id}/refresh`, { method: 'POST' }),
  testAccount: (id) => request(`/api/accounts/${id}/test`, { method: 'POST' }),
  getAccountQuota: (id) => request(`/api/accounts/${id}/quota`),
  getAccountUsage: (period) => request('/api/accounts/usage' + (period ? `?period=${period}` : '')),

  // OAuth
  listOAuthProviders: () => request('/api/accounts/oauth/providers'),
  startOAuth: (providerType) => request('/api/accounts/oauth/start', { method: 'POST', body: JSON.stringify({ provider_type: providerType }) }),
  completeOAuth: (flowId, providerType, callbackUrl, label) => request('/api/accounts/oauth/callback', {
    method: 'POST',
    body: JSON.stringify({
      flow_id: flowId,
      provider_type: providerType,
      ...(callbackUrl ? { callback_url: callbackUrl } : {}),
      ...(label ? { label } : {}),
    }),
  }),

  // Export / Import
  exportData: (includeLogs) => request('/api/export' + (includeLogs ? '?logs=true' : '')),
  importData: (data, mode = 'merge') => request(`/api/import?mode=${mode}`, { method: 'POST', body: JSON.stringify(data) }),

  // Dashboard Auth
  authStatus: () => request('/api/auth/status'),
  authLogin: (password) => request('/api/auth/login', { method: 'POST', body: JSON.stringify({ password }) }),
  setPassword: (currentPassword, newPassword) => request('/api/auth/password', {
    method: 'PUT',
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  }),
  removePassword: (currentPassword) => request('/api/auth/password', {
    method: 'DELETE',
    body: JSON.stringify({ current_password: currentPassword }),
  }),
}
