<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { ScraperKeyword } from '$lib/types';
  import { X, RotateCcw, Save } from 'lucide-svelte';

  let {
    keyword = null,
    isOpen = false,
    onClose,
    onUpdated
  }: {
    keyword: ScraperKeyword | null;
    isOpen: boolean;
    onClose: () => void;
    onUpdated: () => void;
  } = $props();

  let editedQuery = $state('');
  let isSaving = $state(false);
  let isResetting = $state(false);

  $effect(() => {
    if (keyword) {
      editedQuery = keyword.keyword;
    }
  });

  async function handleSave() {
    if (!keyword) return;
    if (!editedQuery.trim()) {
      toast.error('Search query cannot be empty');
      return;
    }

    try {
      isSaving = true;
      await api.put(`/api/v1/keywords/${keyword.id}`, {
        keyword: editedQuery.trim(),
        is_active: keyword.is_active ?? true
      });
      toast.success('Keyword updated');
      onUpdated();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to save changes');
    } finally {
      isSaving = false;
    }
  }

  async function handleResetCursor() {
    if (!keyword) return;
    try {
      isResetting = true;
      await api.post(`/api/v1/keywords/${keyword.id}/reset`, {});
      toast.success(`Reset "${keyword.keyword}" cursor back to Page 1`);
      onUpdated();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to reset cursor');
    } finally {
      isResetting = false;
    }
  }
</script>

{#if isOpen && keyword}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-white">Edit Keyword</h3>
        <button
          onclick={onClose}
          class="text-zinc-500 hover:text-white p-1 rounded-lg hover:bg-zinc-900 transition cursor-pointer"
          aria-label="Close modal"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="p-6 space-y-4">
        
        <!-- Keyword Search Query Input -->
        <div class="space-y-1.5">
          <label for="edit-keyword-query" class="block text-xs text-zinc-400 font-medium">
            Search Keyword / Query
          </label>
          <input
            id="edit-keyword-query"
            type="text"
            bind:value={editedQuery}
            class="w-full p-3 bg-black border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition"
          />
        </div>

        <!-- Cursor Reset Option -->
        <div class="p-3.5 rounded-xl bg-black border border-zinc-800 flex items-center justify-between text-xs">
          <div>
            <div class="text-white font-medium">Cursor Position</div>
            <div class="text-zinc-500 text-[11px] mt-0.5">Currently on Page {keyword.current_page || 1}</div>
          </div>
          <button
            type="button"
            onclick={handleResetCursor}
            disabled={isResetting}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-white border border-zinc-800 text-xs font-medium transition cursor-pointer disabled:opacity-50"
          >
            <RotateCcw class="w-3 h-3 text-amber-400 {isResetting ? 'animate-spin' : ''}" />
            <span>Reset to Page 1</span>
          </button>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-end gap-2.5 pt-3 border-t border-zinc-900">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl bg-zinc-900 text-zinc-400 hover:text-white text-xs font-medium transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSaving}
            class="px-5 py-2 rounded-xl bg-white text-black font-semibold text-xs hover:bg-zinc-200 transition shadow-sm flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
          >
            <Save class="w-3.5 h-3.5" />
            <span>{isSaving ? 'Saving...' : 'Save Changes'}</span>
          </button>
        </div>

      </form>
    </div>
  </div>
{/if}

