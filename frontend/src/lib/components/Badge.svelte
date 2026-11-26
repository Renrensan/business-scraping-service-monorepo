<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    variant = 'default',
    children
  }: {
    variant?: 'default' | 'success' | 'warning' | 'error' | 'info' | 'purple' | 'mono';
    children: Snippet;
  } = $props();

  const variantClasses = {
    default: 'bg-zinc-900 text-zinc-300 border-zinc-800',
    mono: 'bg-zinc-900 text-white border-zinc-700 font-semibold',
    success: 'bg-zinc-900 text-emerald-400 border-zinc-800',
    warning: 'bg-zinc-900 text-amber-400 border-zinc-800',
    error: 'bg-zinc-900 text-rose-400 border-zinc-800',
    info: 'bg-zinc-900 text-zinc-300 border-zinc-800',
    purple: 'bg-zinc-900 text-zinc-200 border-zinc-800'
  };
</script>

<span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md text-[11px] font-mono border {variantClasses[variant]}">
  {@render children()}
</span>

