<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { ScraperKeyword, KeywordScrapeResult } from '$lib/types';
  import { X, Play, RefreshCw, Layers, Mail, CheckCircle2, RotateCcw } from 'lucide-svelte';

  let {
    keyword = null,
    isOpen = false,
    onClose,
    onScraped
  }: {
    keyword: ScraperKeyword | null;
    isOpen: boolean;
    onClose: () => void;
    onScraped: () => void;
  } = $props();

  let maxPages = $state<1 | 2 | 3 | 5>(1);
  let enrichContacts = $state(true);
  let isRunning = $state(false);
  let isResetting = $state(false);

  async function handleResetToPage1() {
    if (!keyword) return;
    try {
      isResetting = true;
      await api.post(`/api/v1/keywords/${keyword.id}/reset`, {});
      toast.success(`Reset "${keyword.keyword}" back to Page 1`);
      keyword.current_page = 1;
      keyword.next_page_token = '';
      keyword.status = 'ready';
      onScraped();
    } catch (err: any) {
      toast.error('Failed to reset: ' + (err?.message || 'Unknown error'));
    } finally {
      isResetting = false;
    }
  }

  async function handleScrape() {
    if (!keyword) return;

    try {
      isRunning = true;
      await api.post<KeywordScrapeResult>(`/api/v1/keywords/${keyword.id}/scrape`, {
        max_pages: Number(maxPages),
        enrich_contacts: enrichContacts
      });

      toast.success(`Scrape started in background for "${keyword.keyword}" (Page ${keyword.current_page || 1})`);
      onScraped();
      onClose();
    } catch (err: any) {
      toast.error('Failed to start scrape: ' + (err?.message || 'Unknown error'));
    } finally {
      isRunning = false;
    }
  }
</script>

{#if isOpen && keyword}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Play class="w-4 h-4 fill-white text-white" />
          <h3 class="text-sm font-semibold text-white">Scrape Keyword</h3>
        </div>
        <button
          onclick={onClose}
          disabled={isRunning}
          class="text-zinc-500 hover:text-white p-1 rounded-lg hover:bg-zinc-900 transition cursor-pointer"
          aria-label="Close modal"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-6 space-y-4">
        <!-- Target Info -->
        <div class="p-3 rounded-xl bg-black border border-zinc-800 text-xs space-y-2">
          <div class="text-zinc-500 text-[11px] font-medium">TARGET QUERY</div>
          <div class="text-white font-semibold text-sm">{keyword.keyword}</div>
          <div class="text-xs flex items-center justify-between pt-1.5 border-t border-zinc-900">
            {#if keyword.status === 'completed' || !keyword.next_page_token}
              <div class="flex items-center gap-1.5 text-emerald-400">
                <CheckCircle2 class="w-3.5 h-3.5" />
                <span>Will start from <strong class="text-white">Page 1</strong> (New Cycle)</span>
              </div>
            {:else}
              <div class="flex items-center gap-1.5 text-zinc-300">
                <span>Continuing from <strong class="text-white">Page {keyword.current_page || 1}</strong></span>
              </div>
              <button
                type="button"
                onclick={handleResetToPage1}
                disabled={isResetting}
                class="flex items-center gap-1 text-[11px] text-amber-400 hover:text-amber-300 font-medium cursor-pointer transition disabled:opacity-50"
              >
                <RotateCcw class="w-3 h-3 {isResetting ? 'animate-spin' : ''}" />
                <span>Reset to Page 1</span>
              </button>
            {/if}
          </div>
        </div>

        <!-- Pages Selection -->
        <div class="space-y-2">
          <span class="block text-xs font-medium text-zinc-300">
            How many pages to scrape in this run?
          </span>
          <div class="grid grid-cols-2 gap-2 text-xs">
            {#each [
              { pages: 1, leads: '~20 leads', desc: '1 Page' },
              { pages: 2, leads: '~40 leads', desc: '2 Pages' },
              { pages: 3, leads: '~60 leads', desc: '3 Pages' },
              { pages: 5, leads: '~100 leads', desc: '5 Pages' }
            ] as opt}
              <button
                type="button"
                onclick={() => (maxPages = opt.pages as any)}
                class="p-3 rounded-xl border text-left transition cursor-pointer flex flex-col justify-between {maxPages === opt.pages ? 'bg-white text-black font-semibold border-white shadow-xs' : 'bg-black text-zinc-400 border-zinc-800 hover:border-zinc-700 hover:text-white'}"
              >
                <div class="font-bold text-xs">{opt.desc}</div>
                <div class="text-[11px] mt-0.5 {maxPages === opt.pages ? 'text-zinc-700' : 'text-zinc-500'}">{opt.leads}</div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Enrichment Toggle -->
        <label class="flex items-center justify-between p-3 rounded-xl bg-black border border-zinc-800 cursor-pointer hover:border-zinc-700 transition">
          <div class="flex items-center gap-2.5">
            <Mail class="w-4 h-4 text-zinc-400" />
            <div>
              <div class="text-xs font-medium text-white">Extract Emails & Socials</div>
              <div class="text-[11px] text-zinc-500">Crawls company website for contact data</div>
            </div>
          </div>
          <input
            type="checkbox"
            bind:checked={enrichContacts}
            class="rounded border-zinc-700 text-white focus:ring-0 focus:ring-offset-0 bg-zinc-900 cursor-pointer"
          />
        </label>
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-zinc-800 bg-black flex items-center justify-end gap-2.5">
        <button
          type="button"
          onclick={onClose}
          disabled={isRunning}
          class="px-4 py-2 rounded-xl bg-zinc-900 text-zinc-400 hover:text-white text-xs font-medium transition cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleScrape}
          disabled={isRunning}
          class="px-5 py-2 rounded-xl bg-white text-black font-semibold text-xs hover:bg-zinc-200 transition shadow-sm flex items-center gap-2 disabled:opacity-50 cursor-pointer"
        >
          {#if isRunning}
            <RefreshCw class="w-3.5 h-3.5 animate-spin text-black" />
            <span>Scraping Google Maps...</span>
          {:else}
            <Play class="w-3.5 h-3.5 fill-black text-black" />
            <span>Start Scrape</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
