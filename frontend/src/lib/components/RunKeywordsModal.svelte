<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ScraperKeyword, ListKeywordResponse } from '$lib/types';
  import { toast } from 'svelte-sonner';
  import { X, Play, RefreshCw, CheckSquare, Square } from 'lucide-svelte';

  let {
    isOpen = false,
    enrichContacts = true,
    maxPages = 3,
    onClose,
    onDone
  }: {
    isOpen: boolean;
    enrichContacts: boolean;
    maxPages: number;
    onClose: () => void;
    onDone: () => void;
  } = $props();

  let keywords = $state<ScraperKeyword[]>([]);
  let selectedIds = $state<Set<string>>(new Set());
  let isLoading = $state(false);
  let isRunning = $state(false);

  $effect(() => {
    if (isOpen) {
      loadKeywords();
      selectedIds = new Set();
    }
  });

  async function loadKeywords() {
    isLoading = true;
    try {
      const res = await api.get<ListKeywordResponse>('/api/v1/keywords', {
        page: 1,
        page_size: 100
      });
      keywords = res.items || [];
      // Pre-select all by default
      selectedIds = new Set(keywords.map(k => k.id));
    } catch {
      toast.error('Failed to load keywords');
    } finally {
      isLoading = false;
    }
  }

  function toggleAll() {
    if (selectedIds.size === keywords.length) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(keywords.map(k => k.id));
    }
  }

  function toggleKeyword(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedIds = next;
  }

  async function runSelected() {
    if (selectedIds.size === 0) {
      toast.error('Select at least one keyword');
      return;
    }

    isRunning = true;
    try {
      const selectedKeywords = keywords
        .filter(k => selectedIds.has(k.id))
        .map(k => k.keyword);

      await api.post('/api/v1/system/trigger-scrape', {
        locations: ['Bali, Indonesia'],
        queries: selectedKeywords,
        categories: [''],
        max_results: maxPages * 20,
        enrich_contacts: enrichContacts,
        engine: 'places'
      });

      toast.success(`Started scrape for ${selectedIds.size} keyword${selectedIds.size !== 1 ? 's' : ''}`);
      onClose();
      onDone();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to start scrape');
    } finally {
      isRunning = false;
    }
  }

  const allSelected = $derived(keywords.length > 0 && selectedIds.size === keywords.length);
  const someSelected = $derived(selectedIds.size > 0 && selectedIds.size < keywords.length);
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-md shadow-2xl flex flex-col max-h-[85vh]">

      <!-- Header -->
      <div class="px-5 py-4 border-b border-zinc-800 flex items-center justify-between shrink-0">
        <div>
          <h3 class="text-sm font-semibold text-white">Run Scrape Now</h3>
          <p class="text-xs text-zinc-500 mt-0.5">Choose which keywords to scrape</p>
        </div>
        <button
          onclick={onClose}
          class="p-1.5 rounded-lg text-zinc-500 hover:text-white hover:bg-zinc-900 transition cursor-pointer"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Select all row -->
      {#if !isLoading && keywords.length > 0}
        <button
          onclick={toggleAll}
          class="px-5 py-3 border-b border-zinc-800 flex items-center gap-3 text-xs text-zinc-400 hover:bg-zinc-900/50 transition cursor-pointer shrink-0"
        >
          {#if allSelected}
            <CheckSquare class="w-4 h-4 text-white shrink-0" />
          {:else if someSelected}
            <div class="w-4 h-4 rounded border border-zinc-500 bg-zinc-800 flex items-center justify-center shrink-0">
              <div class="w-2 h-0.5 bg-zinc-300 rounded"></div>
            </div>
          {:else}
            <Square class="w-4 h-4 shrink-0" />
          {/if}
          <span class="font-medium text-white">Select all</span>
          <span class="ml-auto text-zinc-500">{selectedIds.size} / {keywords.length} selected</span>
        </button>
      {/if}

      <!-- Keyword list -->
      <div class="overflow-y-auto flex-1">
        {#if isLoading}
          <div class="flex items-center justify-center py-12">
            <RefreshCw class="w-5 h-5 animate-spin text-zinc-500" />
          </div>
        {:else if keywords.length === 0}
          <div class="py-12 text-center text-xs text-zinc-500">
            No keywords yet.
            <a href="/keywords" class="text-white underline ml-1">Add keywords</a>
          </div>
        {:else}
          <div class="divide-y divide-zinc-900">
            {#each keywords as kw (kw.id)}
              <button
                onclick={() => toggleKeyword(kw.id)}
                class="w-full px-5 py-3 flex items-center gap-3 text-left hover:bg-zinc-900/40 transition cursor-pointer"
              >
                {#if selectedIds.has(kw.id)}
                  <CheckSquare class="w-4 h-4 text-white shrink-0" />
                {:else}
                  <Square class="w-4 h-4 text-zinc-600 shrink-0" />
                {/if}
                <div class="min-w-0 flex-1">
                  <div class="text-xs font-medium text-white truncate">{kw.keyword}</div>
                  <div class="text-[11px] text-zinc-500 mt-0.5">{kw.total_leads_scraped} leads scraped</div>
                </div>
                <span class="shrink-0 text-[11px] px-2 py-0.5 rounded-full
                  {kw.status === 'completed' ? 'bg-emerald-950 text-emerald-400 border border-emerald-800' :
                   kw.status === 'paginating' ? 'bg-amber-950 text-amber-300 border border-amber-800' :
                   kw.status === 'no_results' ? 'bg-red-950 text-red-300 border border-red-800' :
                   'bg-zinc-900 text-zinc-400 border border-zinc-800'}">
                  {kw.status === 'paginating' ? `p.${kw.current_page}` : kw.status === 'no_results' ? 'no results' : kw.status}
                </span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-5 py-4 border-t border-zinc-800 flex items-center justify-between gap-3 shrink-0">
        <span class="text-xs text-zinc-500">
          {#if selectedIds.size === 0}
            Nothing selected
          {:else}
            {selectedIds.size} keyword{selectedIds.size !== 1 ? 's' : ''} · up to {selectedIds.size * maxPages * 20} places
          {/if}
        </span>
        <div class="flex items-center gap-2">
          <button
            onclick={onClose}
            class="px-4 py-2 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-300 hover:text-white text-xs font-medium transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            onclick={runSelected}
            disabled={isRunning || selectedIds.size === 0}
            class="flex items-center gap-2 px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black text-xs font-semibold transition cursor-pointer disabled:opacity-50"
          >
            <Play class="w-3.5 h-3.5 fill-black text-black" />
            {isRunning ? 'Starting...' : 'Run now'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
