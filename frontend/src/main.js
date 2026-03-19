import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/Dashboard.vue') },
    { path: '/keys', component: () => import('./views/ApiKeys.vue') },
    { path: '/providers', component: () => import('./views/Providers.vue') },
    { path: '/tunnel', component: () => import('./views/Tunnel.vue') },
    { path: '/logs', component: () => import('./views/Logs.vue') },
    { path: '/settings', component: () => import('./views/Settings.vue') },
    { path: '/accounts', component: () => import('./views/AccountUsage.vue') },
    { path: '/about', component: () => import('./views/About.vue') },
  ],
})

createApp(App).use(router).mount('#app')
