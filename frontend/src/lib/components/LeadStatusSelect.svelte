<script lang="ts">
  import type { LeadStatus } from '$lib/types';
  import { LEAD_STATUSES } from '$lib/utils/leadStatus';

  let {
    value = null,
    disabled = false,
    size = 'sm',
    onChange
  }: {
    value?: LeadStatus | null;
    disabled?: boolean;
    size?: 'sm' | 'md';
    onChange: (status: LeadStatus) => void;
  } = $props();

  const pad = $derived(size === 'md' ? 'px-3 py-1.5 text-xs' : 'px-2 py-1 text-[10px]');
</script>

<div class="inline-flex rounded-lg border border-zinc-800 bg-zinc-900/60 p-0.5 gap-0.5" role="radiogroup" aria-label="Lead status">
  {#each LEAD_STATUSES as opt}
    {@const selected = value === opt.value}
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      {disabled}
      title={opt.hint}
      onclick={() => !selected && onChange(opt.value)}
      class="inline-flex items-center gap-1 rounded-md border font-medium transition cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed {pad} {selected
        ? opt.active
        : 'border-transparent text-zinc-500 hover:text-zinc-200 hover:bg-zinc-800/80'}"
    >
      <span class="w-1.5 h-1.5 rounded-full {selected ? opt.dot : 'bg-zinc-600'}"></span>
      {opt.label}
    </button>
  {/each}
</div>
