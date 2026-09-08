<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Sparkles, MapPin, Search, Layers, Mail, Key, ShieldCheck, Cpu } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose,
    onJobCreated
  }: {
    isOpen: boolean;
    onClose: () => void;
    onJobCreated: () => void;
  } = $props();

  let query = $state('');
  let location = $state('');
  let category = $state('');
  let maxResults = $state(50);
  let enrichContacts = $state(true);
  let engine = $state<'places' | 'web'>('places');
  let apiKey = $state('');
  let isSubmitting = $state(false);

  async function handleSubmit() {
    if (!query.trim()) {
      toast.error('Please enter a search query (e.g. "Specialty Coffee")');
      return;
    }
    if (!location.trim()) {
      toast.error('Please specify a target city or location (e.g. "Seattle, WA")');
      return;
    }

    try {
      isSubmitting = true;
      await api.post('/api/v1/jobs', {
        query: query.trim(),
        location: location.trim(),
        category: category.trim(),
        max_results: Number(maxResults),
        enrich_contacts: enrichContacts,
        engine,
        api_key: apiKey.trim() || undefined
      });

      toast.success('Manual extraction job dispatched successfully!');
      onJobCreated();
      onClose();
      query = '';
      location = '';
      category = '';
      apiKey = '';
    } catch (err: any) {
      toast.error(err.message || 'Failed to start scrape job');
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-lg overflow-hidden shadow-2xl">
      <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-lg bg-zinc-900 text-white flex items-center justify-center border border-zinc-800">
            <Sparkles class="w-3.5 h-3.5" />
          </div>
          <h2 class="text-base font-display font-light text-white">Manual Extraction & Ingestion Job</h2>
        </div>
        <button onclick={onClose} class="text-zinc-500 hover:text-white transition cursor-pointer">
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4 text-xs font-mono">
        <div>
          <label for="job-query" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Target Query / Keyword *</label>
          <div class="relative">
            <Search class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
            <input
              id="job-query"
              type="text"
              bind:value={query}
              placeholder="e.g. Specialty Coffee, Dental Clinic, Law Firm"
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
          <div>
            <label for="job-location" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">City / Location *</label>
            <div class="relative">
              <MapPin class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
              <input
                id="job-location"
                type="text"
                bind:value={location}
                placeholder="e.g. Seattle, WA"
                required
                class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
              />
            </div>
          </div>

          <div>
            <label for="job-category" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Category (Optional)</label>
            <div class="relative">
              <Layers class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
              <input
                id="job-category"
                type="text"
                bind:value={category}
                placeholder="e.g. Cafe"
                class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
              />
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
          <div>
            <label for="job-engine" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Extraction Engine</label>
            <div class="relative">
              <Cpu class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
              <select
                id="job-engine"
                bind:value={engine}
                class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs cursor-pointer appearance-none"
              >
                <option value="places">Google Places API (New)</option>
                <option value="web">Direct Maps Web Scraper</option>
              </select>
            </div>
          </div>

          <div>
            <div class="flex justify-between items-center mb-1.5 font-mono">
              <label for="job-max-results" class="text-zinc-400 uppercase text-[10px] tracking-wider">Listings Target</label>
              <span class="text-white font-mono font-bold">{maxResults} leads</span>
            </div>
            <input
              id="job-max-results"
              type="range"
              min="10"
              max="300"
              step="10"
              bind:value={maxResults}
              class="w-full accent-white cursor-pointer mt-1"
            />
          </div>
        </div>

        {#if engine === 'places'}
          <div>
            <label for="job-api-key" class="block text-zinc-400 mb-1.5 uppercase text-[10px] tracking-wider">Custom Places API Key (Optional)</label>
            <div class="relative">
              <Key class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
              <input
                id="job-api-key"
                type="password"
                bind:value={apiKey}
                placeholder="Uses server GOOGLE_PLACES_API_KEY if left blank"
                class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs"
              />
            </div>
          </div>
        {/if}

        <div class="p-3.5 rounded-xl bg-black border border-zinc-800 flex items-center justify-between">
          <div class="flex items-center gap-2.5">
            <div class="w-7 h-7 rounded-md bg-zinc-900 text-emerald-400 flex items-center justify-center border border-zinc-800">
              <Mail class="w-3.5 h-3.5" />
            </div>
            <div>
              <div class="font-medium text-white">Deep Contact & Social Enrichment</div>
              <div class="text-[10px] text-zinc-500">Crawl domains for verified emails & social profiles</div>
            </div>
          </div>
          <input
            type="checkbox"
            bind:checked={enrichContacts}
            class="w-4 h-4 rounded accent-white bg-zinc-900 cursor-pointer"
          />
        </div>

        <!-- Deduplication Engine Guarantee Badge -->
        <div class="p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/80 flex items-start gap-2.5">
          <ShieldCheck class="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
          <div class="text-[11px] text-zinc-400 leading-relaxed">
            <span class="text-zinc-200 font-medium">Smart Deduplication Active:</span> Matches on Place ID, domain, phone, or name/city will automatically merge and update existing records without creating duplicates.
          </div>
        </div>

        <div class="flex items-center justify-end gap-2.5 pt-3 border-t border-zinc-800">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 hover:text-white transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="px-5 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold shadow-sm disabled:opacity-50 transition cursor-pointer"
          >
            {isSubmitting ? 'Dispatching...' : 'Dispatch Worker'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

