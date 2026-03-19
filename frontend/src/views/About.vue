<script setup>
const props = defineProps({ theme: String })

const version = 'v1.0.0'
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-2xl font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">About</h2>
      <p class="text-sm mt-1" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-500'">
        Code Proxy {{ version }}
      </p>
    </div>

    <div class="space-y-6 max-w-2xl">
      <!-- Project -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-12 h-12 rounded-xl bg-blue-500/20 flex items-center justify-center">
            <span class="text-blue-400 text-lg font-bold">CP</span>
          </div>
          <div>
            <h3 class="text-lg font-bold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Code Proxy</h3>
            <p class="text-xs" :class="props.theme === 'light' ? 'text-gray-400' : 'text-gray-500'">{{ version }} &middot; MIT License</p>
          </div>
        </div>
        <p class="text-sm leading-relaxed" :class="props.theme === 'light' ? 'text-gray-600' : 'text-gray-400'">
          A unified proxy that lets you use Claude Code, OpenAI Codex, Gemini and other AI coding agents
          through any OpenAI-compatible client — like Cursor, VS Code, or any tool that speaks the OpenAI API.
          Route CLI subscriptions and API keys through a single endpoint, track usage, and manage access for your team.
        </p>
      </div>

      <!-- Developer -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-4"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Developer</h3>
        <div class="flex items-start gap-4">
          <img src="https://github.com/rodrigorodriguescosta.png" alt="Rodrigo Rodrigues"
               class="w-14 h-14 rounded-full ring-2 ring-blue-500/30" />
          <div>
            <p class="text-sm font-semibold" :class="props.theme === 'light' ? 'text-gray-900' : 'text-white'">Rodrigo Rodrigues</p>
            <p class="text-xs mt-0.5" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              Software Engineer &middot; Full-stack Go + Vue.js
            </p>
            <p class="text-xs mt-2 leading-relaxed" :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              Creator of Code Proxy and Sigeflex ERP. Building tools that make AI coding agents
              accessible and manageable for development teams.
            </p>
            <div class="flex gap-3 mt-3">
              <a href="https://github.com/rodrigorodriguescosta" target="_blank"
                 class="text-xs flex items-center gap-1.5 transition-colors"
                 :class="props.theme === 'light' ? 'text-gray-500 hover:text-gray-900' : 'text-gray-400 hover:text-white'">
                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
                GitHub
              </a>
              <a href="https://github.com/rodrigorodriguescosta/code-proxy" target="_blank"
                 class="text-xs flex items-center gap-1.5 transition-colors"
                 :class="props.theme === 'light' ? 'text-gray-500 hover:text-gray-900' : 'text-gray-400 hover:text-white'">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
                Source Code
              </a>
            </div>
          </div>
        </div>
      </div>

      <!-- Important Notes -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-4"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Important Notes</h3>

        <div class="space-y-4">
          <!-- Cursor localhost warning -->
          <div class="rounded-lg p-4 border"
               :class="props.theme === 'light'
                 ? 'bg-amber-50 border-amber-200'
                 : 'bg-amber-500/5 border-amber-500/20'">
            <p class="text-sm font-semibold mb-1.5"
               :class="props.theme === 'light' ? 'text-amber-800' : 'text-amber-400'">
              Cursor does not support localhost
            </p>
            <p class="text-xs leading-relaxed"
               :class="props.theme === 'light' ? 'text-amber-700' : 'text-amber-300/80'">
              Cursor requires a <strong>public URL</strong> for the OpenAI Base URL — <code class="font-mono">http://localhost:3456</code> will not work.
              You need to expose Code Proxy via a <strong>public IP</strong> (VPS) or a <strong>tunnel</strong> (Cloudflare, ngrok, etc).
            </p>
          </div>

          <!-- Tunnel URL problem -->
          <div class="rounded-lg p-4 border"
               :class="props.theme === 'light'
                 ? 'bg-blue-50 border-blue-200'
                 : 'bg-blue-500/5 border-blue-500/20'">
            <p class="text-sm font-semibold mb-1.5"
               :class="props.theme === 'light' ? 'text-blue-800' : 'text-blue-400'">
              Cloudflare Quick Tunnel URLs change on every restart
            </p>
            <p class="text-xs leading-relaxed mb-2"
               :class="props.theme === 'light' ? 'text-blue-700' : 'text-blue-300/80'">
              The built-in Cloudflare "Quick Tunnel" generates a random <code class="font-mono">*.trycloudflare.com</code> URL
              that changes every time you restart the tunnel. This means you'll have to update Cursor's settings each time.
            </p>
            <p class="text-xs font-semibold mb-1"
               :class="props.theme === 'light' ? 'text-blue-800' : 'text-blue-300'">
              Solutions for a fixed URL:
            </p>
            <ul class="text-xs space-y-1.5 list-disc list-inside"
                :class="props.theme === 'light' ? 'text-blue-700' : 'text-blue-300/80'">
              <li>
                <strong>Cloudflare Named Tunnel</strong> — create a tunnel in
                <a href="https://one.dash.cloudflare.com" target="_blank" class="underline">Cloudflare Zero Trust</a>,
                point it to your custom domain (e.g. <code class="font-mono">proxy.yourdomain.com</code>),
                and paste the token in the <router-link to="/tunnel" class="underline">Tunnel settings</router-link>
              </li>
              <li>
                <strong>VPS with public IP</strong> — deploy on a server with a fixed IP and optionally add a domain with DNS
              </li>
              <li>
                <strong>Reverse proxy</strong> — use nginx/caddy on a VPS pointing to your local tunnel or port-forwarded machine
              </li>
            </ul>
          </div>

          <!-- CLI providers note -->
          <div class="rounded-lg p-4 border"
               :class="props.theme === 'light'
                 ? 'bg-gray-50 border-gray-200'
                 : 'bg-zinc-900/50 border-zinc-800/50'">
            <p class="text-sm font-semibold mb-1.5"
               :class="props.theme === 'light' ? 'text-gray-800' : 'text-gray-300'">
              CLI providers require the CLI tool installed
            </p>
            <p class="text-xs leading-relaxed"
               :class="props.theme === 'light' ? 'text-gray-500' : 'text-gray-400'">
              For <code class="font-mono">cc/*</code> models you need <code class="font-mono">claude</code> CLI installed and authenticated.
              For <code class="font-mono">codex/*</code> models you need <code class="font-mono">codex</code> CLI.
              API providers (<code class="font-mono">anthropic/*</code>, <code class="font-mono">openai/*</code>, etc.) only need an API key or OAuth — no CLI required.
            </p>
          </div>
        </div>
      </div>

      <!-- Tech Stack -->
      <div class="border rounded-xl p-6"
           :class="props.theme === 'light' ? 'bg-white border-gray-200' : 'bg-zinc-900 border-zinc-800/40'">
        <h3 class="text-sm font-semibold uppercase tracking-wider mb-3"
            :class="props.theme === 'light' ? 'text-gray-700' : 'text-white'">Built With</h3>
        <div class="flex flex-wrap gap-2">
          <span v-for="tech in ['Go', 'Vue 3', 'Tailwind CSS', 'SQLite', 'Cloudflare Tunnel', 'OpenAI API']"
                :key="tech"
                class="text-xs px-2.5 py-1 rounded-full"
                :class="props.theme === 'light'
                  ? 'bg-gray-100 text-gray-600'
                  : 'bg-zinc-900 text-gray-400'">
            {{ tech }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
