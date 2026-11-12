<script lang="ts">
  import type { LeadStatus } from '$lib/types';
  import { getLeadStatus } from '$lib/utils/leadStatus';

  let { status }: { status?: LeadStatus | null } = $props();
  const opt = $derived(getLeadStatus(status));
</script>

{#if opt}
  <span
    class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[9px] font-medium border shrink-0 {opt.active}"
    title={opt.hint}
  >
    <span class="w-1.5 h-1.5 rounded-full {opt.dot}"></span>
    {opt.label}
  </span>
{/if}
