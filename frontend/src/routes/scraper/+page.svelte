<script lang="ts">
  import { errorMessage } from '$lib/utils/errors';
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ScheduleInfo, ScraperKeyword, ListKeywordResponse, ScrapeJob } from '$lib/types';
  import { toast } from 'svelte-sonner';
  import { 
    Play, Clock, Plus, Search, Check, Copy,
    CheckCircle2, RefreshCw, Pencil, Trash2, X, Sparkles, AlertCircle, RotateCcw, MapPin } from 'lucide-svelte';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import AddAutoKeywordModal from '$lib/components/AddAutoKeywordModal.svelte';
  import ScrapeKeywordModal from '$lib/components/ScrapeKeywordModal.svelte';
  import KeywordScheduleModal from '$lib/components/KeywordScheduleModal.svelte';
  import EditScraperModal from '$lib/components/EditScraperModal.svelte';
  import Badge from '$lib/components/Badge.svelte';

  // Schedule Info (Global)
  let scheduleInfo = $state<ScheduleInfo | null>(null);

  // Keywords State
  let keywords = $state<ScraperKeyword[]>([]);
  let totalKeywords = $state(0);
  let totalPages = $state(1);
  let currentPage = $state(1);
  let pageSize = $state(50);
  let searchQuery = $state('');
  let isLoadingKeywords = $state(true);

  // Active Tab: Keywords vs Recent Runs
  let activeTab = $state<'keywords' | 'history'>('keywords');
  let recentJobs = $state.raw<ScrapeJob[]>([]);
  let isLoadingHistory = $state(false);

  // Modals
  let showAddModal = $state(false);
  let scrapeModalKw = $state<ScraperKeyword | null>(null);
  let scheduleModalKw = $state<ScraperKeyword | null>(null);
  let editModalKw = $state<ScraperKeyword | null>(null);
  let confirmDeleteKw = $state<ScraperKeyword | null>(null);

  let copiedQueryId = $state<string | null>(null);
  let copiedTokenId = $state<string | null>(null);
  let searchTimeout: ReturnType<typeof setTimeout> | undefined;
  let pollingInterval: ReturnType<typeof setInterval> | null = null;
  let prevProcessingIds = new Set<string>();

  function formatTimeAgo(dateStr?: string): string {
    if (!dateStr) return 'Never';
    const date = new Date(dateStr);
    const now = new Date();
    const diffSec = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (diffSec < 60) return 'Just now';
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return `${Math.floor(diffSec / 86400)}d ago`;
  }

  onMount(() => {
    loadData();
  });

  onDestroy(() => {
    if (pollingInterval) clearInterval(pollingInterval);
  });

  async function loadData() {
    await Promise.all([loadScheduleInfo(), loadKeywords(currentPage, false)]);
    if (activeTab === 'history') loadHistory();
  }

  async function loadScheduleInfo() {
    try {
      const res = await api.get<ScheduleInfo>('/api/v1/system/schedule');
      if (res) scheduleInfo = res;
    } catch (err) {
      console.error('Failed to load schedule info:', err);
    }
  }

  async function loadKeywords(page = 1, silent = false) {
    try {
      if (!silent) isLoadingKeywords = true;
      currentPage = page;
      const res = await api.get<ListKeywordResponse>('/api/v1/keywords', {
        search: searchQuery,
        page: currentPage,
        page_size: pageSize
      });

      if (res) {
        const newKeywords = res.items || [];
        
        // Detect completed background scrapes
        for (const id of prevProcessingIds) {
          const kw = newKeywords.find(k => k.id === id);
          if (kw && kw.status !== 'processing') {
            api.get<{ items: ScrapeJob[] }>('/api/v1/jobs', { page: 1, page_size: 10 }).then(jobRes => {
              const latestJob = jobRes?.items?.find(j => j.query === kw.keyword);
              if (latestJob && latestJob.status === 'FAILED') {
                const errMsg = latestJob.error_message || 'Scrape failed';
                if (errMsg.includes('RESOURCE_EXHAUSTED') || errMsg.includes('Quota exceeded')) {
                  toast.error(`"${kw.keyword}" failed: Google Places API daily quota exceeded!`, { duration: 9000 });
                } else {
                  toast.error(`"${kw.keyword}" failed: ${errMsg}`, { duration: 7000 });
                }
              } else if (kw.status === 'no_results') {
                toast.error(`"${kw.keyword}" returned no results from Google. Try a different wording (e.g. add "private" or the business type).`, { duration: 9000 });
              } else {
                toast.success(`Scrape finished for "${kw.keyword}"! Total leads: ${kw.total_leads_scraped.toLocaleString()} (Page ${kw.current_page || 1})`);
              }
            }).catch(() => {
              toast.success(`Scrape finished for "${kw.keyword}"! Total leads: ${kw.total_leads_scraped.toLocaleString()} (Page ${kw.current_page || 1})`);
            });
          }
        }

        keywords = newKeywords;
        totalKeywords = res.total_count || 0;
        totalPages = res.total_pages || 1;

        // Update processing IDs set
        prevProcessingIds = new Set(newKeywords.filter(k => k.status === 'processing').map(k => k.id));

        // Background polling logic
        if (prevProcessingIds.size > 0) {
          startPolling();
        } else {
          stopPolling();
        }
      }
    } catch (err) {
      console.error('Failed to load keywords:', err);
    } finally {
      if (!silent) isLoadingKeywords = false;
    }
  }

  function startPolling() {
    if (pollingInterval) return;
    pollingInterval = setInterval(() => {
      loadKeywords(currentPage, true);
      if (activeTab === 'history') loadHistory();
    }, 2500);
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      pollingInterval = null;
    }
  }

  async function loadHistory() {
    try {
      isLoadingHistory = true;
      const res = await api.get<{ items: ScrapeJob[] }>('/api/v1/jobs', { page: 1, page_size: 50 });
      if (res?.items) {
        recentJobs = res.items;
      }
    } catch (err) {
      console.error('Failed to load job history:', err);
    } finally {
      isLoadingHistory = false;
    }
  }

  function handleSearchInput() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      loadKeywords(1, false);
    }, 300);
  }

  async function copyToClipboard(text: string, id: string, type: 'query' | 'token') {
    try {
      await navigator.clipboard.writeText(text);
      if (type === 'query') {
        copiedQueryId = id;
        setTimeout(() => (copiedQueryId = null), 2000);
        toast.success(`Copied query "${text}"`);
      } else {
        copiedTokenId = id;
        setTimeout(() => (copiedTokenId = null), 2000);
        toast.success('Copied cursor token to clipboard');
      }
    } catch {
      toast.error('Failed to copy to clipboard');
    }
  }

  let resettingKwId = $state<string | null>(null);

  async function handleDirectReset(kw: ScraperKeyword) {
    try {
      resettingKwId = kw.id;
      await api.post(`/api/v1/keywords/${kw.id}/reset`, {});
      toast.success(`Reset "${kw.keyword}" back to Page 1`);
      await loadKeywords(currentPage, true);
    } catch (err) {
      toast.error('Reset failed: ' + (errorMessage(err, 'Unknown error')));
    } finally {
      resettingKwId = null;
    }
  }

  async function executeDelete() {
    if (!confirmDeleteKw) return;
    const kw = confirmDeleteKw;
    confirmDeleteKw = null;
    try {
      await api.delete(`/api/v1/keywords/${kw.id}`);
      toast.success(`Deleted "${kw.keyword}"`);
      await loadKeywords(currentPage, false);
    } catch (err) {
      toast.error('Delete failed: ' + (errorMessage(err, 'Unknown error')));
    }
  }
</script>

<svelte:head>
  <title>Scraper | MŌMOS</title>
</svelte:head>

<div class="space-y-6 max-w-7xl mx-auto pb-16">
  
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-zinc-900">
    <div>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Scraper</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Manage search queries, run on-demand Google Maps scrapes, and configure background schedules</p>
    </div>

    <div class="flex items-center gap-2.5">
      <!-- Tabs: Keywords vs Run History -->
      <div class="flex items-center bg-zinc-950 p-1 rounded-xl border border-zinc-800 text-xs">
        <button
          onclick={() => (activeTab = 'keywords')}
          class="px-3 py-1.5 rounded-lg font-medium transition cursor-pointer {activeTab === 'keywords' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'}"
        >
          Target Keywords ({totalKeywords})
        </button>
        <button
          onclick={() => { activeTab = 'history'; loadHistory(); }}
          class="px-3 py-1.5 rounded-lg font-medium transition cursor-pointer {activeTab === 'history' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'}"
        >
          Recent Runs
        </button>
      </div>

      <!-- Add Keywords Button -->
      <button
        onclick={() => (showAddModal = true)}
        class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black flex items-center gap-1.5 text-xs font-semibold shadow-sm transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>Add Keywords</span>
      </button>
    </div>
  </div>

  {#if activeTab === 'keywords'}
    <!-- Search Bar -->
    <div class="relative">
      <Search class="w-4 h-4 text-zinc-500 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
      <input
        type="text"
        bind:value={searchQuery}
        oninput={handleSearchInput}
        placeholder="Search keywords or filter by location..."
        class="w-full pl-10 pr-10 py-2.5 bg-zinc-950 border border-zinc-800 rounded-xl text-xs text-white placeholder-zinc-500 focus:outline-hidden focus:border-zinc-600 transition"
      />
      {#if searchQuery}
        <button
          onclick={() => { searchQuery = ''; loadKeywords(1, false); }}
          class="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-white cursor-pointer"
          aria-label="Clear search"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      {/if}
    </div>

    <!-- Keyword Table -->
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl overflow-hidden shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead class="bg-black border-b border-zinc-800 text-[11px] uppercase tracking-wider text-zinc-400 font-semibold">
            <tr>
              <th class="py-3.5 px-5">Target Search Query</th>
              <th class="py-3.5 px-4">Cursor / Page</th>
              <th class="py-3.5 px-4">Cursor Token</th>
              <th class="py-3.5 px-4 text-center">Scheduler</th>
              <th class="py-3.5 px-4 text-center">Total Leads</th>
              <th class="py-3.5 px-4">Last Scraped</th>
              <th class="py-3.5 px-5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-900 text-zinc-300">
            {#if isLoadingKeywords && keywords.length === 0}
              <tr>
                <td colspan="7" class="py-16 text-center text-zinc-500">
                  <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-zinc-400" />
                  <span>Loading keywords...</span>
                </td>
              </tr>
            {:else if keywords.length === 0}
              <tr>
                <td colspan="7" class="py-16 text-center px-4">
                  <div class="max-w-xs mx-auto space-y-3">
                    <p class="text-sm text-zinc-300 font-semibold">No keywords found</p>
                    <p class="text-xs text-zinc-500">
                      {searchQuery ? 'No keywords match your search query.' : 'Add your first target keyword to begin scraping Google Maps leads.'}
                    </p>
                    <button
                      onclick={() => (showAddModal = true)}
                      class="px-4 py-2 rounded-xl bg-white text-black text-xs font-semibold hover:bg-zinc-200 transition cursor-pointer shadow-sm inline-flex items-center gap-1.5"
                    >
                      <Plus class="w-3.5 h-3.5" />
                      <span>Add Keywords</span>
                    </button>
                  </div>
                </td>
              </tr>
            {:else}
              {#each keywords as kw (kw.id)}
                <tr class="transition group {kw.status === 'processing' ? 'bg-amber-500/5 hover:bg-amber-500/10' : 'hover:bg-zinc-900/40'}">
                  
                  <!-- Target Search Query -->
                  <td class="py-3.5 px-5 font-medium text-white max-w-xs truncate">
                    <div class="flex items-center gap-2">
                      <span class="truncate block font-semibold text-xs">{kw.keyword}</span>
                      <button
                        onclick={() => copyToClipboard(kw.keyword, kw.id, 'query')}
                        title="Copy search query"
                        class="opacity-60 group-hover:opacity-100 text-zinc-400 hover:text-white transition p-1 rounded hover:bg-zinc-800 cursor-pointer shrink-0"
                      >
                        {#if copiedQueryId === kw.id}
                          <Check class="w-3.5 h-3.5 text-emerald-400" />
                        {:else}
                          <Copy class="w-3.5 h-3.5" />
                        {/if}
                      </button>
                    </div>
                  </td>

                  <!-- Cursor Position (Page Number) -->
                  <td class="py-3.5 px-4 whitespace-nowrap">
                    {#if kw.status === 'processing'}
                      <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-300 border border-amber-500/30 animate-pulse">
                        <RefreshCw class="w-3 h-3 animate-spin text-amber-400" />
                        Scraping (Page {kw.current_page || 1})...
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium {kw.current_page > 1 ? 'bg-amber-950 text-amber-300 border border-amber-800' : 'bg-zinc-900 border border-zinc-800 text-zinc-300'}">
                        {#if kw.current_page > 1}
                          <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
                        {/if}
                        Page {kw.current_page || 1}
                      </span>
                    {/if}
                  </td>

                  <!-- Cursor Token -->
                  <td class="py-3.5 px-4 whitespace-nowrap">
                    {#if kw.mode === 'area'}
                      <div class="flex flex-col gap-0.5" title="Area cells searched in this cycle. Busy areas are split into smaller cells.">
                        <span class="text-[11px] text-zinc-300">
                          <MapPin class="w-3 h-3 inline -mt-0.5 text-zinc-500" /> {kw.area_name || 'area removed'}{kw.included_type ? ` · ${kw.included_type}` : ''}
                        </span>
                        <span class="text-[11px] {kw.status === 'no_results' ? 'text-red-300' : 'text-zinc-500'}">
                          {#if kw.status === 'no_results'}No results in this area
                          {:else if kw.cells_total > 0}Cells {kw.cells_done}/{kw.cells_total}{kw.status === 'completed' ? ' · cycle done' : ''}
                          {:else}Not started{/if}
                        </span>
                      </div>
                    {:else if kw.next_page_token}
                      <div class="flex items-center gap-1.5">
                        <span class="text-xs font-mono text-zinc-400 max-w-[110px] truncate bg-black px-2 py-0.5 rounded border border-zinc-800" title={kw.next_page_token}>
                          {kw.next_page_token.substring(0, 10)}...
                        </span>
                        <button
                          onclick={() => copyToClipboard(kw.next_page_token, kw.id, 'token')}
                          title="Copy next page cursor token"
                          class="text-zinc-500 hover:text-white p-1 cursor-pointer"
                        >
                          {#if copiedTokenId === kw.id}
                            <Check class="w-3.5 h-3.5 text-emerald-400" />
                          {:else}
                            <Copy class="w-3.5 h-3.5" />
                          {/if}
                        </button>
                      </div>
                    {:else if kw.status === 'no_results'}
                      <span class="inline-flex items-center gap-1 text-[11px] text-red-300 font-medium bg-red-950/40 px-2.5 py-0.5 rounded-full border border-red-900/60" title="Google returned no places for this keyword. Try different wording.">
                        <span>No Results</span>
                      </span>
                    {:else if kw.status === 'completed'}
                      <span class="inline-flex items-center gap-1 text-[11px] text-emerald-400 font-medium bg-emerald-950/40 px-2.5 py-0.5 rounded-full border border-emerald-900/60">
                        <CheckCircle2 class="w-3 h-3 text-emerald-400" />
                        <span>End of Results</span>
                      </span>
                    {:else}
                      <span class="text-zinc-600 text-xs">— (Start)</span>
                    {/if}
                  </td>

                  <!-- Scheduler Status -->
                  <td class="py-3.5 px-4 text-center whitespace-nowrap">
                    {#if kw.is_active}
                      <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] bg-emerald-950 text-emerald-400 border border-emerald-800 font-medium">
                        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                        Scheduled
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] bg-zinc-900 border border-zinc-800 text-zinc-500 font-medium">
                        Manual Only
                      </span>
                    {/if}
                  </td>

                  <!-- Total Leads -->
                  <td class="py-3.5 px-4 text-center whitespace-nowrap">
                    <span class="font-mono text-white font-semibold bg-zinc-900 px-2.5 py-0.5 rounded border border-zinc-800">
                      {kw.total_leads_scraped.toLocaleString()}
                    </span>
                    {#if kw.last_scraped_at}
                      <div class="text-[10px] mt-1 {kw.new_places_last_run > 0 ? 'text-emerald-400' : 'text-zinc-600'}" title="New places found by the last run">
                        +{kw.new_places_last_run} new
                      </div>
                    {/if}
                    {#if kw.next_eligible_at && new Date(kw.next_eligible_at) > new Date()}
                      <div class="text-[10px] text-zinc-500" title="Recent runs found nothing new, so the scheduler runs this keyword less often. 'Scrape' still runs it now.">
                        next auto run {new Date(kw.next_eligible_at).toLocaleDateString()}
                      </div>
                    {/if}
                  </td>

                  <!-- Last Scraped -->
                  <td class="py-3.5 px-4 whitespace-nowrap text-zinc-400 text-xs">
                    {formatTimeAgo(kw.last_scraped_at)}
                  </td>

                  <!-- Actions -->
                  <td class="py-3.5 px-5 text-right whitespace-nowrap">
                    <div class="flex items-center justify-end gap-1.5">
                      <!-- Scrape Now Button -->
                      {#if kw.status === 'processing'}
                        <button
                          disabled
                          title="Scraping currently running in background"
                          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-amber-500/10 text-amber-300 border border-amber-500/30 font-semibold text-xs transition cursor-not-allowed opacity-80 mr-1"
                        >
                          <RefreshCw class="w-3.5 h-3.5 animate-spin text-amber-400" />
                          <span>Scraping...</span>
                        </button>
                      {:else}
                        <button
                          onclick={() => (scrapeModalKw = kw)}
                          title="Scrape Now"
                          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs transition shadow-sm cursor-pointer mr-1"
                        >
                          <Play class="w-3.5 h-3.5 fill-black text-black" />
                          <span>Scrape</span>
                        </button>
                      {/if}

                      <!-- Schedule (Clock icon) -->
                      <button
                        onclick={() => (scheduleModalKw = kw)}
                        title="Configure Auto-Schedule"
                        class="p-2 rounded-xl border text-xs transition cursor-pointer {kw.is_active ? 'bg-emerald-950/60 border-emerald-800/80 text-emerald-400 hover:bg-emerald-900/60' : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-white hover:bg-zinc-800'}"
                      >
                        <Clock class="w-3.5 h-3.5" />
                      </button>

                      <!-- Reset to Page 1 (RotateCcw icon) -->
                      <button
                        onclick={() => handleDirectReset(kw)}
                        disabled={kw.status === 'processing' || resettingKwId === kw.id}
                        title="Reset page back to Page 1"
                        class="p-2 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-amber-400 hover:bg-zinc-800 transition cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                      >
                        <RotateCcw class="w-3.5 h-3.5 {resettingKwId === kw.id ? 'animate-spin text-amber-400' : ''}" />
                      </button>

                      <!-- Edit (Pencil icon) -->
                      <button
                        onclick={() => (editModalKw = kw)}
                        title="Edit query / reset cursor"
                        class="p-2 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-white hover:bg-zinc-800 transition cursor-pointer"
                      >
                        <Pencil class="w-3.5 h-3.5" />
                      </button>

                      <!-- Delete (Trash icon) -->
                      <button
                        onclick={() => (confirmDeleteKw = kw)}
                        title="Delete keyword"
                        class="p-2 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-500 hover:text-rose-400 hover:bg-zinc-800 transition cursor-pointer"
                      >
                        <Trash2 class="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            {/if}
          </tbody>
        </table>
      </div>

      <!-- Pagination Footer -->
      {#if totalPages > 1}
        <div class="px-5 py-3 border-t border-zinc-800 flex items-center justify-between bg-black text-xs text-zinc-400">
          <span>Page {currentPage} of {totalPages} ({totalKeywords} keywords)</span>
          <div class="flex items-center gap-2">
            <button
              disabled={currentPage <= 1}
              onclick={() => loadKeywords(currentPage - 1)}
              class="px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 hover:text-white disabled:opacity-40 cursor-pointer font-medium"
            >
              Previous
            </button>
            <button
              disabled={currentPage >= totalPages}
              onclick={() => loadKeywords(currentPage + 1)}
              class="px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 hover:text-white disabled:opacity-40 cursor-pointer font-medium"
            >
              Next
            </button>
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <!-- Recent Runs Live History Table -->
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl overflow-hidden shadow-sm">
      <div class="px-5 py-3.5 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-xs font-semibold text-white">Execution History</span>
          <span class="text-xs px-2 py-0.5 rounded-full bg-zinc-900 text-zinc-400 border border-zinc-800">
            {recentJobs.length} Runs
          </span>
        </div>
        <button
          onclick={loadHistory}
          class="text-zinc-400 hover:text-white text-xs flex items-center gap-1.5 transition font-medium cursor-pointer"
        >
          <RefreshCw class="w-3.5 h-3.5 {isLoadingHistory ? 'animate-spin' : ''}" />
          <span>Refresh</span>
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-black border-b border-zinc-800 text-[11px] uppercase tracking-wider text-zinc-400 font-semibold">
            <tr>
              <th class="px-5 py-3.5">Query & Location</th>
              <th class="px-4 py-3.5">Status</th>
              <th class="px-4 py-3.5">Leads Scraped</th>
              <th class="px-4 py-3.5">Emails Extracted</th>
              <th class="px-4 py-3.5">Timestamp</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-900 text-zinc-300">
            {#if isLoadingHistory && recentJobs.length === 0}
              <tr>
                <td colspan="5" class="px-5 py-12 text-center text-zinc-500">
                  <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-zinc-400" />
                  <span>Loading recent runs...</span>
                </td>
              </tr>
            {:else if recentJobs.length === 0}
              <tr>
                <td colspan="5" class="px-5 py-12 text-center text-zinc-500">
                  No scrape jobs executed yet. Trigger a scrape to see live logs here.
                </td>
              </tr>
            {:else}
              {#each recentJobs as job (job.id)}
                <tr class="hover:bg-zinc-900/40 transition">
                  <td class="px-5 py-3.5">
                    <div class="font-medium text-white flex items-center gap-1.5">
                      <Search class="w-3.5 h-3.5 text-zinc-400" />
                      <span>{job.query}</span>
                    </div>
                    <div class="text-xs text-zinc-500 mt-0.5">
                      <span>{job.location}</span>
                    </div>
                    {#if job.status === 'FAILED' && job.error_message}
                      <div class="text-[11px] text-rose-400 mt-1 flex items-center gap-1 font-mono">
                        <AlertCircle class="w-3 h-3 shrink-0 text-rose-400" />
                        <span class="truncate max-w-sm" title={job.error_message}>
                          {job.error_message.includes('RESOURCE_EXHAUSTED') ? 'Quota exceeded (SearchText per day)' : job.error_message}
                        </span>
                      </div>
                    {/if}
                  </td>

                  <td class="px-4 py-3.5 whitespace-nowrap">
                    <Badge variant={job.status === 'COMPLETED' ? 'success' : job.status === 'PROCESSING' ? 'warning' : job.status === 'FAILED' ? 'error' : 'default'}>
                      {#if job.status === 'COMPLETED'}
                        <CheckCircle2 class="w-3 h-3 text-emerald-400" />
                      {:else if job.status === 'PROCESSING'}
                        <RefreshCw class="w-3 h-3 animate-spin text-amber-400" />
                      {/if}
                      <span>{job.status}</span>
                    </Badge>
                  </td>

                  <td class="px-4 py-3.5 whitespace-nowrap font-mono font-bold text-white">
                    {job.total_scraped} <span class="text-xs text-zinc-500 font-normal">/ {job.max_results}</span>
                  </td>

                  <td class="px-4 py-3.5 whitespace-nowrap font-mono text-emerald-400 font-bold">
                    {job.total_enriched}
                  </td>

                  <td class="px-4 py-3.5 whitespace-nowrap text-zinc-400 font-mono text-xs">
                    {job.created_at ? new Date(job.created_at).toLocaleString() : 'Pending'}
                  </td>
                </tr>
              {/each}
            {/if}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>

<!-- Add Keywords Modal -->
<AddAutoKeywordModal
  isOpen={showAddModal}
  onClose={() => (showAddModal = false)}
  onKeywordsAdded={() => {
    loadKeywords(1);
    loadScheduleInfo();
  }}
/>

<!-- Scrape Keyword Modal (Configurable pages per run) -->
<ScrapeKeywordModal
  isOpen={!!scrapeModalKw}
  keyword={scrapeModalKw}
  onClose={() => (scrapeModalKw = null)}
  onScraped={() => {
    loadKeywords(currentPage);
    loadHistory();
  }}
/>

<!-- Keyword Schedule Modal (Clock button) -->
<KeywordScheduleModal
  isOpen={!!scheduleModalKw}
  keyword={scheduleModalKw}
  scheduleInfo={scheduleInfo}
  onClose={() => (scheduleModalKw = null)}
  onUpdated={() => {
    loadKeywords(currentPage);
    loadScheduleInfo();
  }}
/>

<!-- Edit Keyword Modal (Pencil button) -->
<EditScraperModal
  isOpen={!!editModalKw}
  keyword={editModalKw}
  onClose={() => (editModalKw = null)}
  onUpdated={() => {
    loadKeywords(currentPage);
    loadScheduleInfo();
  }}
/>

<!-- Delete confirmation -->
<ConfirmModal
  isOpen={!!confirmDeleteKw}
  title="Delete keyword?"
  message={confirmDeleteKw ? `"${confirmDeleteKw.keyword}" will be permanently removed.` : ''}
  confirmLabel="Delete"
  variant="danger"
  onConfirm={executeDelete}
  onCancel={() => (confirmDeleteKw = null)}
/>
