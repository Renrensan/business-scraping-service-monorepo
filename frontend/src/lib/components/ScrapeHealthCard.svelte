<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { Activity, AlertTriangle, PauseCircle, CheckCircle2, BellOff, Bell } from 'lucide-svelte';

  type Health = {
    status: 'ok' | 'warning' | 'paused';
    issues: string[];
    jobs: {
      pending: number;
      processing: number;
      completed_24h: number;
      failed_24h: number;
      cancelled_24h: number;
      new_leads_24h: number;
      recent_failures: { query: string; status: string; error_message: string; finished_at?: string }[];
    };
    keywords: { total: number; active: number; no_results: number; backed_off: number; running: number };
    quota_paused_until?: string;
    quota_reset_at: string;
    places_requests_today: number;
    places_daily_budget: number;
    auto_enabled: boolean;
    next_run_at: string;
    last_run_at?: string;
    has_api_key: boolean;
    alerts_enabled: boolean;
  };

  let health = $state<Health | null>(null);
  let loadError = $state('');

  async function load() {
    try {
      health = await api.get<Health>('/api/v1/system/scrape-health');
      loadError = '';
    } catch (e: any) {
      loadError = e?.message || 'Failed to load scraper health';
    }
  }

  onMount(() => {
    load();
    const t = setInterval(load, 30000);
    return () => clearInterval(t);
  });

  const fmt = (s?: string) => (s ? new Date(s).toLocaleString() : '—');
</script>

<div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-5 space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Activity class="w-4 h-4 text-zinc-400" />
      <h2 class="text-sm font-medium text-white">Scraper health</h2>
    </div>
    {#if health}
      <span class="inline-flex items-center gap-1.5 text-[11px] px-2.5 py-0.5 rounded-full border
        {health.status === 'ok' ? 'bg-emerald-950 text-emerald-300 border-emerald-800' : health.status === 'paused' ? 'bg-red-950 text-red-300 border-red-800' : 'bg-amber-950 text-amber-300 border-amber-800'}">
        {#if health.status === 'ok'}<CheckCircle2 class="w-3 h-3" /> Healthy
        {:else if health.status === 'paused'}<PauseCircle class="w-3 h-3" /> Paused
        {:else}<AlertTriangle class="w-3 h-3" /> Needs attention{/if}
      </span>
    {/if}
  </div>

  {#if loadError}
    <p class="text-xs text-red-300">{loadError}</p>
  {:else if health}
    {#if health.issues.length}
      <ul class="space-y-1">
        {#each health.issues as issue}
          <li class="text-xs text-amber-200 flex gap-1.5"><AlertTriangle class="w-3.5 h-3.5 shrink-0 mt-0.5" />{issue}</li>
        {/each}
      </ul>
    {/if}

    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
      <div class="bg-black/40 border border-zinc-900 rounded-xl p-3">
        <div class="text-zinc-500">Queue</div>
        <div class="text-white font-mono mt-0.5">{health.jobs.processing} running · {health.jobs.pending} waiting</div>
      </div>
      <div class="bg-black/40 border border-zinc-900 rounded-xl p-3">
        <div class="text-zinc-500">Last 24h</div>
        <div class="text-white font-mono mt-0.5">
          {health.jobs.completed_24h} ok · <span class={health.jobs.failed_24h ? 'text-red-300' : ''}>{health.jobs.failed_24h} failed</span>
        </div>
        <div class="text-zinc-500 mt-0.5">+{health.jobs.new_leads_24h} new leads</div>
      </div>
      <div class="bg-black/40 border border-zinc-900 rounded-xl p-3">
        <div class="text-zinc-500">Places requests today</div>
        <div class="text-white font-mono mt-0.5">
          {health.places_requests_today}{health.places_daily_budget ? ` / ${health.places_daily_budget}` : ''}
        </div>
        <div class="text-zinc-500 mt-0.5">resets {fmt(health.quota_reset_at)}</div>
      </div>
      <div class="bg-black/40 border border-zinc-900 rounded-xl p-3">
        <div class="text-zinc-500">Auto scraping</div>
        <div class="text-white mt-0.5">{health.auto_enabled ? `next ${fmt(health.next_run_at)}` : 'off'}</div>
        <div class="text-zinc-500 mt-0.5">last {fmt(health.last_run_at)}</div>
      </div>
    </div>

    <div class="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-zinc-500">
      <span>{health.keywords.total} keywords · {health.keywords.active} active</span>
      {#if health.keywords.no_results}<span class="text-red-300">{health.keywords.no_results} with no results</span>{/if}
      {#if health.keywords.backed_off}<span>{health.keywords.backed_off} running less often (nothing new)</span>{/if}
      <span class="inline-flex items-center gap-1">
        {#if health.alerts_enabled}<Bell class="w-3 h-3" /> WhatsApp alerts on{:else}<BellOff class="w-3 h-3" /> WhatsApp alerts off{/if}
      </span>
    </div>

    {#if health.jobs.recent_failures.length}
      <details class="text-xs">
        <summary class="text-zinc-400 cursor-pointer">Recent problems ({health.jobs.recent_failures.length})</summary>
        <ul class="mt-2 space-y-1.5">
          {#each health.jobs.recent_failures as f}
            <li class="text-zinc-400">
              <span class="text-white">{f.query}</span>
              <span class="text-zinc-600">· {f.status.toLowerCase()} · {fmt(f.finished_at)}</span>
              {#if f.error_message}<div class="text-zinc-500 truncate" title={f.error_message}>{f.error_message}</div>{/if}
            </li>
          {/each}
        </ul>
      </details>
    {/if}
  {:else}
    <p class="text-xs text-zinc-500">Loading…</p>
  {/if}
</div>
