<script lang="ts">
  import type { Tier } from '$lib/types';

  let { tier, score, overridden = false }: { tier: Tier; score?: number; overridden?: boolean } = $props();

  const styles: Record<Tier, { label: string; cls: string }> = {
    hot: { label: '🔥 HOT', cls: 'bg-red-950 text-red-300 border-red-800' },
    good: { label: '⭐ GOOD', cls: 'bg-amber-950 text-amber-300 border-amber-800' },
    low: { label: '⚪ LOW', cls: 'bg-zinc-900 text-zinc-400 border-zinc-800' },
    skip: { label: '⛔ SKIP', cls: 'bg-zinc-950 text-zinc-500 border-zinc-800 line-through' }
  };
</script>

<span
  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full border text-[10px] font-semibold whitespace-nowrap {styles[tier].cls}"
  title={overridden ? 'Tier set by hand' : 'Tier from score and area cut-offs'}
>
  {styles[tier].label}{#if score !== undefined}<span class="font-mono font-normal opacity-80">{score}</span>{/if}{#if overridden}<span class="opacity-70">✎</span>{/if}
</span>
