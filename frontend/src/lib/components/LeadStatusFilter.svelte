<script lang="ts">
  import type { LeadStatus } from '$lib/types';
  import { LEAD_STATUSES } from '$lib/utils/leadStatus';

  let {
    selected = [],
    onChange
  }: {
    selected?: LeadStatus[];
    onChange: (statuses: LeadStatus[]) => void;
  } = $props();

  function toggle(value: LeadStatus) {
    onChange(selected.includes(value) ? selected.filter((s) => s !== value) : [...selected, value]);
  }
</script>

<div class="flex flex-wrap items-center gap-2">
  <span class="text-[11px] font-medium text-zinc-500 uppercase tracking-wider mr-1">Status:</span>
  <button
    type="button"
    onclick={() => onChange([])}
    class="px-3 py-1.5 rounded-xl border text-xs transition cursor-pointer {selected.length === 0
      ? 'bg-white/10 border-white/30 text-white font-medium'
      : 'bg-zinc-900/60 border-zinc-800/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}"
  >
    All
  </button>
  {#each LEAD_STATUSES as opt (opt.value)}
    {@const on = selected.includes(opt.value)}
    <button
      type="button"
      aria-pressed={on}
      title={opt.hint}
      onclick={() => toggle(opt.value)}
      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl border text-xs transition cursor-pointer {on
        ? opt.active + ' font-medium'
        : 'bg-zinc-900/60 border-zinc-800/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}"
    >
      <span class="w-1.5 h-1.5 rounded-full {opt.dot}"></span>
      {opt.label}
    </button>
  {/each}
</div>
