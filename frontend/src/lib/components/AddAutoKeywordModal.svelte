<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Plus, Search } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose,
    onKeywordsAdded
  }: {
    isOpen: boolean;
    onClose: () => void;
    onKeywordsAdded: () => void;
  } = $props();

  let keywordsText = $state('');
  let isSubmitting = $state(false);

  async function handleSubmit() {
    const lines = keywordsText
      .split('\n')
      .map(s => s.trim())
      .filter(s => s.length > 0);

    if (lines.length === 0) {
      toast.error('Please enter at least one target keyword');
      return;
    }

    try {
      isSubmitting = true;
      const res = await api.post<{ added_count: number }>('/api/v1/keywords', {
        keywords: lines
      });

      toast.success(`Added ${res.added_count || lines.length} keyword(s) successfully!`);
      keywordsText = '';
      onKeywordsAdded();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to add keywords');
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-lg overflow-y-auto shadow-2xl">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Search class="w-4 h-4 text-white" />
          <h3 class="text-sm font-semibold text-white">
            Add Target Keywords
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
        
        <!-- Keywords Input -->
        <div class="space-y-1.5">
          <label for="auto-keywords-input" class="block text-xs text-zinc-400 font-medium">
            Search Queries / Target Keywords (one per line)
          </label>
          <textarea
            id="auto-keywords-input"
            rows="5"
            bind:value={keywordsText}
            placeholder="e.g. Specialty Coffee in Orchard, Singapore&#10;Dental Clinics in Jurong East&#10;Coworking Spaces in Central Singapore"
            class="w-full p-3 bg-black border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition leading-relaxed font-sans"
          ></textarea>
          <p class="text-[11px] text-zinc-500">
            Enter search queries to scrape on-demand or schedule background runs later.
          </p>
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
            class="px-5 py-2.5 rounded-xl bg-white text-black font-semibold text-xs hover:bg-zinc-200 transition shadow-sm flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
          >
            <Plus class="w-4 h-4" />
            <span>{isSubmitting ? 'Adding...' : 'Add Keywords'}</span>
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
