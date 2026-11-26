<script lang="ts">
  import { api } from '$lib/api/client';
  import type { AuditEntry, AuditListResponse } from '$lib/types';
  import AuditTable from '$lib/components/AuditTable.svelte';
  import { History } from 'lucide-svelte';

  let { leadId, refreshKey = 0 }: { leadId: string; refreshKey?: number } = $props();
  let entries = $state.raw<AuditEntry[]>([]);

  async function load(id: string) {
    try {
      const res = await api.get<AuditListResponse>('/api/v1/audit', { entity_type: 'lead', entity_id: id, page_size: 20 });
      if (id === leadId) entries = res?.items || [];
    } catch {
      /* history is optional */
    }
  }

  $effect(() => {
    void refreshKey; // reload after changes made in the drawer
    load(leadId);
  });
</script>

<div class="mt-4 p-3 rounded-xl bg-black border border-zinc-800">
  <div class="flex items-center gap-2 mb-1">
    <History class="w-4 h-4 text-zinc-400" />
    <span class="text-[10px] uppercase tracking-wider text-zinc-500 font-semibold">History</span>
  </div>
  <AuditTable {entries} showEntity={false} />
</div>
