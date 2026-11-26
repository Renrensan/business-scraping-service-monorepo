<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { ScraperKeyword, ScheduleInfo } from '$lib/types';
  import { X, Clock, Power, Save } from 'lucide-svelte';

  let {
    keyword = null,
    isOpen = false,
    scheduleInfo = null,
    onClose,
    onUpdated
  }: {
    keyword: ScraperKeyword | null;
    isOpen: boolean;
    scheduleInfo: ScheduleInfo | null;
    onClose: () => void;
    onUpdated: () => void;
  } = $props();

  // Form fields start from the props and are editable; they reset when the props change.
  const toPages = (n: number | undefined): 1 | 2 | 3 => (n === 1 || n === 2 ? n : 3);
  let isScheduled = $derived(keyword?.is_active ?? true);
  let intervalHours = $derived(scheduleInfo?.interval_hours || 24);
  let maxPagesPerRun = $derived(toPages(scheduleInfo?.max_pages_per_run));
  let enrichContacts = $derived(scheduleInfo?.enrich_contacts ?? true);
  let isSaving = $state(false);

  async function handleSave() {
    if (!keyword) return;

    try {
      isSaving = true;

      // 1. Update Keyword active state
      await api.put(`/api/v1/keywords/${keyword.id}`, {
        keyword: keyword.keyword,
        is_active: isScheduled
      });

      // 2. Save Schedule Global frequency & auto_enabled if active
      await api.post('/api/v1/system/schedule', {
        interval_hours: Number(intervalHours),
        auto_enabled: isScheduled ? true : (scheduleInfo?.auto_enabled ?? false),
        max_pages_per_run: Number(maxPagesPerRun),
        page_size: 20,
        max_results: maxPagesPerRun * 20,
        enrich_contacts: enrichContacts,
        engine: 'places'
      });

      toast.success(`Schedule updated for "${keyword.keyword}"`);
      onUpdated();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to update schedule');
    } finally {
      isSaving = false;
    }
  }
</script>

{#if isOpen && keyword}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-xs p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Clock class="w-4 h-4 text-emerald-400" />
          <h3 class="text-sm font-semibold text-white">Keyword Scheduler</h3>
        </div>
        <button
          onclick={onClose}
          class="text-zinc-500 hover:text-white p-1 rounded-lg hover:bg-zinc-900 transition cursor-pointer"
          aria-label="Close modal"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-6 space-y-4">
        <!-- Target Query Info -->
        <div class="p-3 rounded-xl bg-black border border-zinc-800 text-xs">
          <div class="text-zinc-500 text-[11px] font-medium">TARGET QUERY</div>
          <div class="text-white font-semibold text-sm mt-0.5">{keyword.keyword}</div>
        </div>

        <!-- Master Include Toggle -->
        <div class="flex items-center justify-between p-3.5 rounded-xl bg-black border border-zinc-800">
          <div>
            <div class="text-xs text-white font-semibold">Enable Background Schedule</div>
            <div class="text-[11px] text-zinc-500 mt-0.5">Scrapes automatically on recurring interval</div>
          </div>
          <button
            type="button"
            onclick={() => (isScheduled = !isScheduled)}
            aria-label="Toggle background schedule"
            class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-hidden {isScheduled ? 'bg-emerald-500' : 'bg-zinc-800'}"
            role="switch"
            aria-checked={isScheduled}
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out {isScheduled ? 'translate-x-5' : 'translate-x-0'}"
            ></span>
          </button>
        </div>

        {#if isScheduled}
          <!-- Frequency -->
          <div class="space-y-1.5">
            <span class="block text-xs text-zinc-400 font-medium">How often to run this scraper?</span>
            <div class="grid grid-cols-4 gap-1.5 text-xs">
              {#each [
                { hours: 1, label: 'Every 1h' },
                { hours: 4, label: 'Every 4h' },
                { hours: 12, label: 'Every 12h' },
                { hours: 24, label: 'Every 24h' }
              ] as opt}
                <button
                  type="button"
                  onclick={() => (intervalHours = opt.hours)}
                  class="p-2 rounded-xl border text-center transition cursor-pointer font-medium {intervalHours === opt.hours ? 'bg-white text-black font-semibold border-white shadow-xs' : 'bg-black text-zinc-400 border-zinc-800 hover:text-white'}"
                >
                  {opt.label}
                </button>
              {/each}
            </div>
          </div>

          <!-- Pages per cycle -->
          <div class="space-y-1.5">
            <span class="block text-xs text-zinc-400 font-medium">Pages per cycle</span>
            <div class="grid grid-cols-3 gap-1.5 text-xs">
              {#each [
                { pages: 1, label: '1 Page (20 leads)' },
                { pages: 2, label: '2 Pages (40 leads)' },
                { pages: 3, label: '3 Pages (60 leads)' }
              ] as opt}
                <button
                  type="button"
                  onclick={() => (maxPagesPerRun = opt.pages as any)}
                  class="p-2 rounded-xl border text-center transition cursor-pointer font-medium {maxPagesPerRun === opt.pages ? 'bg-white text-black font-semibold border-white shadow-xs' : 'bg-black text-zinc-400 border-zinc-800 hover:text-white'}"
                >
                  {opt.label}
                </button>
              {/each}
            </div>
          </div>

          <!-- Enrich Emails Checkbox -->
          <label class="flex items-center justify-between p-3 rounded-xl bg-black border border-zinc-800 cursor-pointer hover:border-zinc-700 transition">
            <span class="text-xs font-medium text-white">Extract Emails & Socials</span>
            <input
              type="checkbox"
              bind:checked={enrichContacts}
              class="rounded border-zinc-700 text-white focus:ring-0 focus:ring-offset-0 bg-zinc-900 cursor-pointer"
            />
          </label>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-zinc-800 bg-black flex items-center justify-end gap-2.5">
        <button
          type="button"
          onclick={onClose}
          disabled={isSaving}
          class="px-4 py-2 rounded-xl bg-zinc-900 text-zinc-400 hover:text-white text-xs font-medium transition cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleSave}
          disabled={isSaving}
          class="px-5 py-2 rounded-xl bg-white text-black font-semibold text-xs hover:bg-zinc-200 transition shadow-sm flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
        >
          <Save class="w-3.5 h-3.5" />
          <span>{isSaving ? 'Saving...' : 'Save Schedule'}</span>
        </button>
      </div>
    </div>
  </div>
{/if}
