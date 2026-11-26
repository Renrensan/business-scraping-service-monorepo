<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Plus, Search, MapPin, Type } from 'lucide-svelte';
  import type { Area } from '$lib/types';

  let {
    isOpen = false,
    onClose,
    onKeywordsAdded
  }: {
    isOpen: boolean;
    onClose: () => void;
    onKeywordsAdded: () => void;
  } = $props();

  type Mode = 'area' | 'plain';
  let mode = $state<Mode>('area');
  let isSubmitting = $state(false);

  // Plain mode
  let keywordsText = $state('');

  // Area mode
  let term = $state('');
  let includedType = $state('');
  let typeTouched = $state(false);
  let areas = $state.raw<Area[]>([]);
  let selected = $state<Set<string>>(new Set());
  let areaFilter = $state('');
  let newAreaName = $state('');
  let isAddingArea = $state(false);
  let suggestTimer: ReturnType<typeof setTimeout> | undefined;

  const filteredAreas = $derived(
    areas.filter((a) => a.name.toLowerCase().includes(areaFilter.trim().toLowerCase()))
  );
  const allFilteredSelected = $derived(
    filteredAreas.length > 0 && filteredAreas.every((a) => selected.has(a.id))
  );

  $effect(() => {
    if (isOpen && areas.length === 0) loadAreas();
  });

  async function loadAreas() {
    try {
      areas = (await api.get<Area[]>('/api/v1/areas')) || [];
    } catch (err: any) {
      toast.error(err?.message || 'Failed to load areas');
    }
  }

  // Pre-fill the place type from the term, until the admin edits it themselves.
  function onTermInput() {
    if (typeTouched) return;
    clearTimeout(suggestTimer);
    suggestTimer = setTimeout(async () => {
      if (!term.trim()) {
        includedType = '';
        return;
      }
      try {
        const res = await api.get<{ included_type: string }>('/api/v1/keywords/type-suggestion', { term: term.trim() });
        if (!typeTouched) includedType = res?.included_type || '';
      } catch {
        /* suggestion is optional */
      }
    }, 250);
  }

  function toggleArea(id: string) {
    const next = new Set(selected);
    next.has(id) ? next.delete(id) : next.add(id);
    selected = next;
  }

  function toggleAllFiltered() {
    const next = new Set(selected);
    if (allFilteredSelected) filteredAreas.forEach((a) => next.delete(a.id));
    else filteredAreas.forEach((a) => next.add(a.id));
    selected = next;
  }

  async function addArea() {
    const name = newAreaName.trim();
    if (!name) return;
    try {
      isAddingArea = true;
      const area = await api.post<Area>('/api/v1/areas', { name });
      if (area) {
        if (!areas.some((a) => a.id === area.id)) {
          areas = [...areas, area].sort((a, b) => a.name.localeCompare(b.name));
        }
        selected = new Set(selected).add(area.id);
        toast.success(`Area "${area.name}" added`);
      }
      newAreaName = '';
    } catch (err: any) {
      toast.error(err?.message || 'Could not find that place');
    } finally {
      isAddingArea = false;
    }
  }

  async function handleSubmit() {
    if (mode === 'plain') return submitPlain();
    return submitArea();
  }

  async function submitArea() {
    if (!term.trim()) {
      toast.error('Enter what to search for, e.g. villa');
      return;
    }
    if (selected.size === 0) {
      toast.error('Pick at least one area');
      return;
    }
    try {
      isSubmitting = true;
      const res = await api.post<{ count: number; existing: number }>('/api/v1/keywords', {
        term: term.trim(),
        included_type: includedType.trim(),
        area_ids: [...selected]
      });
      const existing = res?.existing ? ` (${res.existing} already existed)` : '';
      toast.success(`Added ${res?.count ?? 0} keyword(s)${existing}`);
      term = '';
      includedType = '';
      typeTouched = false;
      selected = new Set();
      onKeywordsAdded();
      onClose();
    } catch (err: any) {
      toast.error(err?.message || 'Failed to add keywords');
    } finally {
      isSubmitting = false;
    }
  }

  async function submitPlain() {
    const lines = keywordsText
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    if (lines.length === 0) {
      toast.error('Please enter at least one target keyword');
      return;
    }

    try {
      isSubmitting = true;
      const res = await api.post<{ count: number }>('/api/v1/keywords', { keywords: lines });
      toast.success(`Added ${res?.count ?? lines.length} keyword(s) successfully!`);
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
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto shadow-2xl">
      <!-- Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Search class="w-4 h-4 text-white" />
          <h3 class="text-sm font-semibold text-white">Add Target Keywords</h3>
        </div>
        <button
          onclick={onClose}
          class="text-zinc-500 hover:text-white p-1 rounded-lg hover:bg-zinc-900 transition cursor-pointer"
          aria-label="Close modal"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Mode tabs -->
      <div class="px-6 pt-4">
        <div class="inline-flex rounded-xl bg-zinc-900 p-1 text-xs">
          <button
            type="button"
            onclick={() => (mode = 'area')}
            class="px-3 py-1.5 rounded-lg transition cursor-pointer flex items-center gap-1.5 {mode === 'area' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'}"
          >
            <MapPin class="w-3.5 h-3.5" /> By area
          </button>
          <button
            type="button"
            onclick={() => (mode = 'plain')}
            class="px-3 py-1.5 rounded-lg transition cursor-pointer flex items-center gap-1.5 {mode === 'plain' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'}"
          >
            <Type class="w-3.5 h-3.5" /> Plain text
          </button>
        </div>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4">
        {#if mode === 'area'}
          <!-- Term -->
          <div class="space-y-1.5">
            <label for="kw-term" class="block text-xs text-zinc-400 font-medium">Search for</label>
            <input
              id="kw-term"
              bind:value={term}
              oninput={onTermInput}
              placeholder="e.g. villa, coworking space, surf school"
              class="w-full px-3 py-2.5 bg-black border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition"
            />
          </div>

          <!-- Type -->
          <div class="space-y-1.5">
            <label for="kw-type" class="block text-xs text-zinc-400 font-medium">
              Place type <span class="text-zinc-600">(optional)</span>
            </label>
            <input
              id="kw-type"
              bind:value={includedType}
              oninput={() => (typeTouched = true)}
              placeholder="none: search by text only"
              class="w-full px-3 py-2.5 bg-black border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition font-mono"
            />
            <p class="text-[11px] text-zinc-500">
              Suggested from the term. A Google Places type such as <code>lodging</code>, <code>spa</code> or <code>restaurant</code>;
              it narrows results to that kind of place. Leave empty if unsure.
            </p>
          </div>

          <!-- Areas -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="block text-xs text-zinc-400 font-medium">Areas <span class="text-zinc-600">({selected.size} selected)</span></span>
              <button type="button" onclick={toggleAllFiltered} class="text-[11px] text-zinc-400 hover:text-white cursor-pointer">
                {allFilteredSelected ? 'Clear' : 'Select all'}{areaFilter ? ' shown' : ''}
              </button>
            </div>
            <input
              bind:value={areaFilter}
              placeholder="Filter areas…"
              class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-xs text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition"
            />
            <div class="flex flex-wrap gap-1.5 max-h-44 overflow-y-auto p-0.5">
              {#each filteredAreas as area (area.id)}
                <button
                  type="button"
                  onclick={() => toggleArea(area.id)}
                  title={area.source === 'preset' ? 'Approximate preset area' : area.source === 'lookup' ? 'Found on Google Maps' : 'Manual area'}
                  class="px-2.5 py-1 rounded-full text-[11px] border transition cursor-pointer {selected.has(area.id) ? 'bg-white text-black border-white font-medium' : 'bg-zinc-900 text-zinc-300 border-zinc-800 hover:border-zinc-600'}"
                >
                  {area.name}
                </button>
              {:else}
                <span class="text-[11px] text-zinc-600">No areas match.</span>
              {/each}
            </div>

            <!-- Add an area by name -->
            <div class="flex gap-2 pt-1">
              <input
                bind:value={newAreaName}
                onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addArea(); } }}
                placeholder="Add an area by name, e.g. Bedugul, Bali"
                class="flex-1 px-3 py-2 bg-black border border-zinc-800 rounded-xl text-xs text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition"
              />
              <button
                type="button"
                onclick={addArea}
                disabled={isAddingArea || !newAreaName.trim()}
                class="px-3 py-2 rounded-xl bg-zinc-900 border border-zinc-800 text-xs text-zinc-300 hover:text-white disabled:opacity-50 cursor-pointer"
              >
                {isAddingArea ? 'Finding…' : 'Add area'}
              </button>
            </div>
            <p class="text-[11px] text-zinc-500">
              Looks the place up on Google Maps once (1 API request) and saves it for everyone.
            </p>
          </div>

          {#if term.trim() && selected.size > 0}
            <p class="text-[11px] text-zinc-400 bg-zinc-900/60 border border-zinc-800 rounded-lg px-3 py-2">
              Creates {selected.size} keyword{selected.size > 1 ? 's' : ''}, e.g. <strong class="text-white">{term.trim()} · {areas.find((a) => selected.has(a.id))?.name}</strong>.
              Busy areas are split automatically to get past Google's 60-result limit.
            </p>
          {/if}
        {:else}
          <!-- Plain keywords (unchanged behaviour) -->
          <div class="space-y-1.5">
            <label for="auto-keywords-input" class="block text-xs text-zinc-400 font-medium">
              Search queries (one per line)
            </label>
            <textarea
              id="auto-keywords-input"
              rows="5"
              bind:value={keywordsText}
              placeholder="e.g. rooftop bar in Seminyak, Bali"
              class="w-full p-3 bg-black border border-zinc-800 rounded-xl text-sm text-white placeholder-zinc-600 focus:outline-hidden focus:border-zinc-500 transition leading-relaxed font-sans"
            ></textarea>
            <p class="text-[11px] text-zinc-500">
              Sent to Google as typed. Returns at most 60 places per query.
            </p>
          </div>
        {/if}

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
