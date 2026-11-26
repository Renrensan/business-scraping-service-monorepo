<script lang="ts">
  import type { AuditEntry } from '$lib/types';

  let { entries, showEntity = true }: { entries: AuditEntry[]; showEntity?: boolean } = $props();

  let open = $state<number | null>(null);
  const fmt = (s: string) => new Date(s).toLocaleString();
</script>

<div class="divide-y divide-zinc-900 text-xs">
  {#each entries as e (e.id)}
    <div class="py-2">
      <button onclick={() => (open = open === e.id ? null : e.id)} class="w-full text-left flex flex-wrap items-baseline gap-x-2 gap-y-0.5 cursor-pointer">
        <span class="text-zinc-500 font-mono text-[11px] shrink-0">{fmt(e.created_at)}</span>
        <span class="text-white font-medium">{e.admin_name || 'unknown'}</span>
        <span class="text-zinc-300">{e.summary}</span>
        {#if showEntity && (e.entity_name || e.entity_id)}
          <span class="text-zinc-500">· {e.entity_name || `${e.entity_type} ${e.entity_id?.slice(0, 8)}`}</span>
        {/if}
        {#if e.status >= 400}
          <span class="text-[10px] px-1.5 py-0.5 rounded-full bg-red-950 text-red-300 border border-red-900">failed {e.status}</span>
        {/if}
      </button>
      {#if open === e.id}
        <div class="mt-1.5 p-2 rounded-lg bg-black border border-zinc-900 text-[11px] text-zinc-400 space-y-1">
          {#if e.details?.error}<div class="text-red-300">Error: {e.details.error}</div>{/if}
          <div>{e.method} {e.route} · IP {e.ip || '—'} · ref {e.request_id || '—'}</div>
          {#if e.details?.query}<div class="font-mono break-all">?{e.details.query}</div>{/if}
          {#if e.details?.request}<pre class="font-mono whitespace-pre-wrap break-all">{JSON.stringify(e.details.request, null, 2)}</pre>{/if}
        </div>
      {/if}
    </div>
  {:else}
    <p class="py-6 text-center text-zinc-600">No recorded actions.</p>
  {/each}
</div>
