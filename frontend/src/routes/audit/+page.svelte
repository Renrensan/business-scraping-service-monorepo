<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { AdminUser, AuditEntry, AuditListResponse } from '$lib/types';
  import AuditTable from '$lib/components/AuditTable.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import { Search } from 'lucide-svelte';

  let entries = $state.raw<AuditEntry[]>([]);
  let admins = $state.raw<AdminUser[]>([]);
  let total = $state(0);
  let page = $state(1);
  let totalPages = $state(1);
  let adminId = $state('');
  let search = $state('');
  let failedOnly = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function load() {
    try {
      const res = await api.get<AuditListResponse>('/api/v1/audit', {
        page,
        page_size: 50,
        admin_id: adminId || undefined,
        search: search.trim() || undefined,
        failed: failedOnly ? 'true' : undefined
      });
      entries = res?.items || [];
      total = res?.total_count || 0;
      totalPages = res?.total_pages || 1;
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load the audit log'));
    }
  }

  function refilter() {
    page = 1;
    load();
  }

  onMount(async () => {
    load();
    admins = (await api.get<AdminUser[]>('/api/v1/admins').catch(() => [])) || [];
  });
</script>

<div class="max-w-5xl mx-auto space-y-5">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Audit log</h1>
    <p class="text-xs text-zinc-500 mt-0.5">{total.toLocaleString()} recorded actions: every change, export and login, with who made it</p>
  </div>

  <div class="flex flex-wrap gap-2 items-center">
    <select bind:value={adminId} onchange={refilter} class="bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 text-xs cursor-pointer">
      <option value="">All admins</option>
      {#each admins as a (a.id)}<option value={a.id}>{a.username}</option>{/each}
    </select>
    <div class="relative flex-1 min-w-[200px]">
      <Search class="w-4 h-4 absolute left-3 top-2 text-zinc-500" />
      <input
        bind:value={search}
        oninput={() => { clearTimeout(timer); timer = setTimeout(refilter, 300); }}
        placeholder="Search actions, admins, details…"
        class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-zinc-600 text-xs"
      />
    </div>
    <label class="inline-flex items-center gap-2 text-xs text-zinc-400 cursor-pointer">
      <input type="checkbox" bind:checked={failedOnly} onchange={refilter} class="accent-white" /> Failed only
    </label>
  </div>

  <div class="bg-zinc-950 border border-zinc-800 rounded-2xl px-4">
    <AuditTable {entries} />
    <Pagination {page} {totalPages} onPageChange={(p) => { page = p; load(); }} />
  </div>
</div>
