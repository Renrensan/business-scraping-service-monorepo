<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Sparkles, MapPin, Search, Layers, Mail } from 'lucide-svelte';

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
  let isSubmitting = $state(false);

  async function handleSubmit() {
    if (!query.trim()) {
      toast.error('Please enter a search query (e.g. "Coffee Shops")');
      return;
    }
    if (!location.trim()) {
      toast.error('Please specify a target city or location (e.g. "Austin, TX")');
      return;
    }

    try {
      isSubmitting = true;
      await api.post('/api/v1/jobs', {
        query: query.trim(),
        location: location.trim(),
        category: category.trim(),
        max_results: Number(maxResults),
        enrich_contacts: enrichContacts
      });

      toast.success('Extraction job created and dispatched!');
      onJobCreated();
      onClose();
      query = '';
      location = '';
      category = '';
    } catch (err: any) {
      toast.error(err.message || 'Failed to start scrape job');
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-xs">
    <div class="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg overflow-hidden shadow-2xl">
      <div class="flex items-center justify-between px-6 py-4 border-b border-slate-800">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center">
            <Sparkles class="w-4 h-4" />
          </div>
          <h2 class="text-sm font-bold text-white">New Extraction & Enrichment Job</h2>
        </div>
        <button onclick={onClose} class="text-slate-400 hover:text-white">
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4 text-xs">
        <div>
          <label class="block font-medium text-slate-300 mb-1.5">Target Business Keyword / Query *</label>
          <div class="relative">
            <Search class="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
            <input
              type="text"
              bind:value={query}
              placeholder="e.g. Specialty Coffee, Dental Clinic, Real Estate Agent"
              required
              class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
          <div>
            <label class="block font-medium text-slate-300 mb-1.5">City / Location *</label>
            <div class="relative">
              <MapPin class="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
              <input
                type="text"
                bind:value={location}
                placeholder="e.g. Seattle, WA"
                required
                class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
              />
            </div>
          </div>

          <div>
            <label class="block font-medium text-slate-300 mb-1.5">Category Tag (Optional)</label>
            <div class="relative">
              <Layers class="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
              <input
                type="text"
                bind:value={category}
                placeholder="e.g. Coffee Shop"
                class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-9 pr-3.5 py-2 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs"
              />
            </div>
          </div>
        </div>

        <div>
          <div class="flex justify-between items-center mb-1.5">
            <label class="font-medium text-slate-300">Max Results Limit</label>
            <span class="text-indigo-400 font-mono font-bold">{maxResults} listings</span>
          </div>
          <input
            type="range"
            min="10"
            max="300"
            step="10"
            bind:value={maxResults}
            class="w-full accent-indigo-500 cursor-pointer"
          />
        </div>

        <div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800 flex items-center justify-between">
          <div class="flex items-center gap-2.5">
            <div class="w-7 h-7 rounded-md bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
              <Mail class="w-3.5 h-3.5" />
            </div>
            <div>
              <div class="font-medium text-slate-200">Deep Contact & Social Enrichment</div>
              <div class="text-[11px] text-slate-400">Crawl websites for emails, LinkedIn, Instagram, FB, X</div>
            </div>
          </div>
          <input
            type="checkbox"
            bind:checked={enrichContacts}
            class="w-4 h-4 rounded accent-indigo-600 bg-slate-800 cursor-pointer"
          />
        </div>

        <div class="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-800">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700 transition"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium shadow-md shadow-indigo-600/20 disabled:opacity-50 transition"
          >
            {isSubmitting ? 'Dispatching...' : 'Launch Ingestion Job'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
