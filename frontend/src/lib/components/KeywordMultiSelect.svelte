<script lang="ts">
  import { ChevronDown, Search, X, Check } from 'lucide-svelte';

  let {
    options = [],
    selected = [],
    placeholder = 'All Scrape Keywords',
    onChange
  }: {
    options?: string[];
    selected?: string[];
    placeholder?: string;
    onChange: (keywords: string[]) => void;
  } = $props();

  let open = $state(false);
  let query = $state('');
  let root: HTMLDivElement | undefined = $state();

  const filtered = $derived(
    query.trim() ? options.filter((o) => o.toLowerCase().includes(query.trim().toLowerCase())) : options
  );

  const summary = $derived(
    selected.length === 0 ? placeholder : selected.length === 1 ? selected[0] : `${selected.length} keywords selected`
  );

  function toggle(kw: string) {
    onChange(selected.includes(kw) ? selected.filter((k) => k !== kw) : [...selected, kw]);
  }

  function handleWindowClick(e: MouseEvent) {
    if (open && root && !root.contains(e.target as Node)) open = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
  }
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<div class="relative" bind:this={root}>
  <button
    type="button"
    onclick={() => (open = !open)}
    aria-haspopup="listbox"
    aria-expanded={open}
    class="w-full flex items-center justify-between gap-2 bg-zinc-900 border rounded-xl px-3 py-2 text-xs cursor-pointer transition {open
      ? 'border-zinc-500'
      : 'border-zinc-800'} {selected.length ? 'text-white' : 'text-zinc-200'}"
  >
    <span class="truncate">{summary}</span>
    <span class="flex items-center gap-1 shrink-0">
      {#if selected.length}
        <span class="px-1.5 rounded bg-white/10 text-[10px] font-mono">{selected.length}</span>
      {/if}
      <ChevronDown class="w-3.5 h-3.5 text-zinc-500 transition {open ? 'rotate-180' : ''}" />
    </span>
  </button>

  {#if open}
    <div class="absolute z-30 mt-1.5 w-full min-w-[240px] bg-zinc-950 border border-zinc-800 rounded-xl shadow-2xl overflow-hidden">
      <div class="p-2 border-b border-zinc-900">
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-2.5 top-2 text-zinc-500" />
          <input
            type="text"
            bind:value={query}
            placeholder="Find keyword..."
            class="w-full bg-zinc-900 border border-zinc-800 rounded-lg pl-8 pr-2 py-1.5 text-xs text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-600"
          />
        </div>
      </div>

      <ul class="max-h-64 overflow-y-auto py-1" role="listbox" aria-multiselectable="true">
        {#each filtered as kw (kw)}
          {@const on = selected.includes(kw)}
          <li>
            <button
              type="button"
              role="option"
              aria-selected={on}
              onclick={() => toggle(kw)}
              class="w-full flex items-center gap-2 px-3 py-1.5 text-left text-xs transition cursor-pointer hover:bg-zinc-900 {on ? 'text-white' : 'text-zinc-300'}"
            >
              <span class="w-3.5 h-3.5 rounded border flex items-center justify-center shrink-0 {on ? 'bg-white border-white' : 'border-zinc-700'}">
                {#if on}<Check class="w-2.5 h-2.5 text-black" />{/if}
              </span>
              <span class="truncate">{kw}</span>
            </button>
          </li>
        {:else}
          <li class="px-3 py-3 text-xs text-zinc-600 text-center">No keywords found</li>
        {/each}
      </ul>

      {#if selected.length}
        <div class="flex items-center justify-between px-3 py-2 border-t border-zinc-900 text-[11px]">
          <span class="text-zinc-500">{selected.length} selected · matches any</span>
          <button type="button" onclick={() => onChange([])} class="text-zinc-300 hover:text-white cursor-pointer">Clear</button>
        </div>
      {/if}
    </div>
  {/if}

  {#if selected.length > 1}
    <div class="flex flex-wrap gap-1 mt-1.5">
      {#each selected as kw (kw)}
        <span class="inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-md bg-zinc-900 border border-zinc-800 text-[10px] text-zinc-300 max-w-full">
          <span class="truncate">{kw}</span>
          <button type="button" onclick={() => toggle(kw)} class="text-zinc-500 hover:text-white cursor-pointer" aria-label="Remove {kw}">
            <X class="w-3 h-3" />
          </button>
        </span>
      {/each}
    </div>
  {/if}
</div>
