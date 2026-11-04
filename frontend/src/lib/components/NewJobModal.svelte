<script lang="ts">
  import { api } from '$lib/api/client';
  import type { ListKeywordResponse, ScraperKeyword } from '$lib/types';
  import { toast } from 'svelte-sonner';
  import { X, Play, Sparkles, MapPin, KeyRound } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose,
    onJobCreated
  }: {
    isOpen: boolean;
    onClose: () => void;
    onJobCreated: () => void;
  } = $props();

  let searchQuery = $state('Coffee shops in Singapore');
  let location = $state('Singapore');
  let maxResults = $state<number>(60);
  let enrichContacts = $state(true);
  let isSubmitting = $state(false);
  let availableKeywords = $state<ScraperKeyword[]>([]);

  $effect(() => {
    if (isOpen) {
      loadKeywords();
    }
  });

  async function loadKeywords() {
    try {
      const res = await api.get<ListKeywordResponse>('/api/v1/keywords', { page: 1, page_size: 50 });
      if (res?.items) {
        availableKeywords = res.items;
      }
    } catch {
      // Non-blocking
    }
  }

  function appendKeyword(kw: string) {
    const current = searchQuery.trim();
    if (!current) {
      searchQuery = kw;
    } else if (!current.includes(kw)) {
      searchQuery = `${current}\n${kw}`;
    }
  }

  async function handleSubmit() {
    const lines = searchQuery
      .split('\n')
      .map(s => s.trim())
      .filter(s => s.length > 0);

    if (lines.length === 0) {
      toast.error('Please enter at least one search query');
      return;
    }

    try {
      isSubmitting = true;

      for (const q of lines) {
        await api.post('/api/v1/jobs', {
          query: q,
          location: location || 'Singapore',
          category: 'Local Business',
          max_results: maxResults,
          enrich_contacts: enrichContacts,
          engine: 'places'
        });
      }

      toast.success(
        lines.length === 1 
          ? `Started scraping "${lines[0]}"!` 
          : `Started scraping ${lines.length} queries!`
      );

      onJobCreated();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to start scraping job');
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
    <div class="bg-black border border-zinc-800 rounded-2xl w-full max-w-lg overflow-y-auto shadow-2xl max-h-[90vh]">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Play class="w-4 h-4 text-emerald-400 fill-emerald-400" />
          <h3 class="text-sm font-semibold text-white">
            Scrape Now
          </h3>
        </div>
        <button
          onclick={onClose}
          class="text-zinc-500 hover:text-white p-1 rounded-lg hover:bg-zinc-900 transition cursor-pointer"
          aria-label="Close modal"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4">
        
        <!-- Search Query Input -->
        <div class="space-y-1.5">
          <label for="search-queries-input" class="block text-xs text-zinc-400 font-medium">
            Search Queries (one per line)
          </label>
          <div class="relative">
            <textarea
              id="search-queries-input"
              rows="3"
              bind:value={searchQuery}
              placeholder="e.g. Specialty Coffee in Orchard&#10;Dental Clinics in Tampines"
              class="w-full p-3 bg-zinc-950 border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition leading-relaxed font-sans"
            ></textarea>
          </div>

          <!-- Quick-pick existing keywords -->
          {#if availableKeywords.length > 0}
            <div class="pt-1.5 space-y-1">
              <span class="text-[11px] text-zinc-500 flex items-center gap-1 font-medium">
                <KeyRound class="w-3 h-3 text-zinc-500" />
                <span>Quick pick from target keywords:</span>
              </span>
              <div class="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto pr-1">
                {#each availableKeywords as kw}
                  <button
                    type="button"
                    onclick={() => appendKeyword(kw.keyword)}
                    class="px-2 py-0.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-300 hover:text-white text-[11px] font-medium transition cursor-pointer"
                  >
                    + {kw.keyword}
                  </button>
                {/each}
              </div>
            </div>
          {/if}
        </div>

        <!-- Location Filter -->
        <div class="space-y-1.5">
          <label for="location-input" class="block text-xs text-zinc-400 font-medium">
            Location Context
          </label>
          <div class="relative flex items-center">
            <MapPin class="w-4 h-4 text-zinc-500 absolute left-3.5 pointer-events-none" />
            <input
              id="location-input"
              type="text"
              bind:value={location}
              placeholder="e.g. Singapore, Bali, Jakarta"
              class="w-full pl-10 pr-3 py-2.5 bg-zinc-950 border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition"
            />
          </div>
        </div>

        <!-- Max Leads -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-zinc-400 font-medium">Max leads to scrape</span>
            <span class="text-xs text-zinc-500 font-mono">{maxResults} places</span>
          </div>
          <div class="grid grid-cols-4 gap-2 text-center text-xs">
            {#each ([20, 60, 100, 200] as const) as count}
              <button
                type="button"
                onclick={() => maxResults = count}
                class="py-2 px-1 rounded-lg border transition cursor-pointer font-medium
                  {maxResults === count ? 'bg-white text-black border-white' : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-white'}"
              >
                {count}
              </button>
            {/each}
          </div>
        </div>

        <!-- Extract Emails Toggle -->
        <div class="flex items-center justify-between p-3.5 rounded-xl bg-zinc-950 border border-zinc-800">
          <div>
            <div class="text-xs text-white font-medium flex items-center gap-1.5">
              <span>Extract Emails & Socials</span>
              <Sparkles class="w-3.5 h-3.5 text-amber-400" />
            </div>
            <div class="text-[11px] text-zinc-500 mt-0.5">Visits business websites in real-time to find emails</div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={enrichContacts} class="sr-only peer" />
            <div class="w-10 h-5 bg-zinc-800 peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-white peer-checked:after:bg-black"></div>
          </label>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-end gap-3 pt-3 border-t border-zinc-900">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl bg-zinc-900 text-zinc-400 hover:text-white text-xs font-medium transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="px-5 py-2.5 rounded-xl bg-white text-black font-semibold text-xs hover:bg-zinc-200 transition shadow-sm flex items-center gap-2 disabled:opacity-50 cursor-pointer"
          >
            <Play class="w-3.5 h-3.5 fill-black text-black" />
            <span>{isSubmitting ? 'Starting...' : 'Start Scraping Now'}</span>
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
