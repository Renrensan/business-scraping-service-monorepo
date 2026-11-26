<script lang="ts">
  import { untrack } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { BusinessLead, LeadStatus, ProfileSummary, ScoringProfile, Tier } from '$lib/types';
  import TierBadge from '$lib/components/TierBadge.svelte';
  import LeadStatusBadge from '$lib/components/LeadStatusBadge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import BusinessDetailDrawer from '$lib/components/BusinessDetailDrawer.svelte';
  import ExportModal from '$lib/components/ExportModal.svelte';
  import { Search, Download, Star, Eye, RefreshCw } from 'lucide-svelte';

  /** The ranked list of one scoring profile's leads (its "prospects"). */
  let {
    profile,
    summary,
    onScoresChanged
  }: {
    profile: ScoringProfile;
    summary: ProfileSummary | null;
    onScoresChanged?: () => void;
  } = $props();

  const TIERS: Tier[] = ['hot', 'good', 'low', 'skip'];
  const OTHER_AREAS = '-'; // leads outside every named area (default cut-offs)

  const profileId = $derived(profile.id);
  let leads = $state<BusinessLead[]>([]); // items are edited in place (contacted)
  let totalCount = $state(0);
  let totalPages = $state(1);
  let pageNo = $state(1);
  let isLoading = $state(true);

  let selectedTiers = $state<Tier[]>([]);
  let area = $state('');
  let search = $state('');
  let contacted = $state<'' | 'true' | 'false'>('');
  let searchTimeout: ReturnType<typeof setTimeout> | undefined;

  let selectedLead = $state<BusinessLead | null>(null);
  let showDrawer = $state(false);
  let showExport = $state(false);

  const filterParams = $derived({
    score_profile: profileId,
    scored_only: 'true',
    sort_by: 'score',
    tiers: selectedTiers.length ? selectedTiers.join(',') : undefined,
    score_area: area || undefined,
    search: search.trim() || undefined,
    contacted: contacted || undefined
  });

  // Counts for the chips: the selected area's, or the whole profile's.
  const tierCounts = $derived.by(() => {
    if (!summary) return {} as Partial<Record<Tier, number>>;
    if (!area) return summary.tiers;
    return summary.areas.find((a) => (area === OTHER_AREAS ? a.name === '' : a.name === area))?.tiers ?? {};
  });
  const cutoffFor = (name: string) => {
    const c = profile.config.area_cutoffs?.[name] ?? profile.config.default_cutoff;
    return c ? `HOT ≥${c.hot} · GOOD ≥${c.good}` : '';
  };

  async function fetchLeads() {
    try {
      isLoading = true;
      const res = await api.get<{ items: BusinessLead[]; total_count: number; total_pages: number }>('/api/v1/businesses', {
        ...filterParams,
        page: pageNo,
        page_size: 25
      });
      leads = res?.items || [];
      totalCount = res?.total_count || 0;
      totalPages = res?.total_pages || 1;
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load prospects'));
    } finally {
      isLoading = false;
    }
  }

  function refilter() {
    pageNo = 1;
    fetchLeads();
  }

  function toggleTier(t: Tier) {
    selectedTiers = selectedTiers.includes(t) ? selectedTiers.filter((x) => x !== t) : [...selectedTiers, t];
    refilter();
  }

  async function toggleContacted(lead: BusinessLead, checked: boolean) {
    const prev = { contacted: lead.contacted, lead_status: lead.lead_status };
    lead.contacted = checked;
    lead.lead_status = checked ? (lead.lead_status ?? 'cold') : null;
    try {
      await api.patch(`/api/v1/businesses/${lead.id}/contacted`, { contacted: checked });
      toast.success(checked ? `Marked "${lead.name}" as contacted` : `Unmarked "${lead.name}"`);
    } catch (err) {
      lead.contacted = prev.contacted;
      lead.lead_status = prev.lead_status;
      toast.error(errorMessage(err, 'Failed to update contacted status'));
    }
  }

  async function setStatus(lead: BusinessLead, status: LeadStatus) {
    const prev = { contacted: lead.contacted, lead_status: lead.lead_status };
    lead.lead_status = status;
    lead.contacted = true;
    try {
      await api.patch(`/api/v1/businesses/${lead.id}/status`, { status });
    } catch (err) {
      lead.contacted = prev.contacted;
      lead.lead_status = prev.lead_status;
      toast.error(errorMessage(err, 'Failed to set status'));
    }
  }

  // Reset and load when the profile changes (switching in the dropdown). Only profileId is a
  // dependency: the filters are read inside untrack, or changing a filter would re-run this
  // and wipe it.
  $effect(() => {
    void profileId;
    untrack(() => {
      selectedTiers = [];
      area = '';
      search = '';
      contacted = '';
      pageNo = 1;
      fetchLeads();
    });
  });
</script>

<div class="space-y-5">
    <!-- Filters -->
    <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-4 space-y-3">
      <div class="flex flex-wrap items-center gap-2">
        {#each TIERS as t (t)}
          <button
            onclick={() => toggleTier(t)}
            class="inline-flex items-center gap-1 rounded-full cursor-pointer transition {selectedTiers.length && !selectedTiers.includes(t) ? 'opacity-40 hover:opacity-80' : ''}"
            title="Show only these tiers"
          >
            <TierBadge tier={t} />
            <span class="text-[11px] text-zinc-400 font-mono">{tierCounts[t] ?? 0}</span>
          </button>
        {/each}
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div class="relative">
          <Search class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
          <input
            bind:value={search}
            oninput={() => { clearTimeout(searchTimeout); searchTimeout = setTimeout(refilter, 300); }}
            placeholder="Search name, phone, contact…"
            class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-zinc-600 text-xs"
          />
        </div>
        <select bind:value={area} onchange={refilter} class="bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 text-xs cursor-pointer" title="Area used for the HOT/GOOD cut-offs">
          <option value="">All areas</option>
          {#each summary?.areas ?? [] as a (a.name)}
            <option value={a.name === '' ? OTHER_AREAS : a.name}>
              {a.name || 'Other areas'} ({a.total}) · {cutoffFor(a.name)}
            </option>
          {/each}
        </select>
        <select bind:value={contacted} onchange={refilter} class="bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 text-xs cursor-pointer">
          <option value="">Contacted or not</option>
          <option value="false">Not contacted yet</option>
          <option value="true">Contacted</option>
        </select>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-black border-b border-zinc-800 text-[11px] uppercase tracking-wider text-zinc-400 font-semibold">
            <tr>
              <th class="px-4 py-3.5 w-12 text-center" title="Mark if already contacted">Done</th>
              <th class="px-4 py-3.5">Tier</th>
              <th class="px-4 py-3.5">Business</th>
              <th class="px-4 py-3.5">Area · group</th>
              <th class="px-4 py-3.5">Rating</th>
              <th class="px-4 py-3.5">Channels</th>
              <th class="px-4 py-3.5 text-right">Details</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-zinc-900 text-zinc-300">
            {#if isLoading}
              <tr><td colspan="7" class="px-5 py-16 text-center text-zinc-500"><RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-zinc-400" />Loading prospects…</td></tr>
            {:else if leads.length === 0}
              <tr><td colspan="7" class="px-5 py-16 text-center text-zinc-500">No prospects match these filters.</td></tr>
            {:else}
              {#each leads as lead (lead.id)}
                <tr class="hover:bg-zinc-900/40 transition {lead.contacted ? 'bg-emerald-950/10' : ''}">
                  <td class="px-4 py-3.5 text-center">
                    <input
                      type="checkbox"
                      checked={lead.contacted ?? false}
                      onchange={(e) => toggleContacted(lead, e.currentTarget.checked)}
                      class="w-4 h-4 rounded border-zinc-700 bg-zinc-900 accent-emerald-500 cursor-pointer"
                      title={lead.contacted ? 'Contacted - click to unmark' : 'Click to mark as contacted'}
                    />
                  </td>
                  <td class="px-4 py-3.5 whitespace-nowrap">
                    {#if lead.score}
                      <TierBadge tier={lead.score.tier} score={lead.score.score} overridden={!!lead.score.tier_override} />
                    {/if}
                  </td>
                  <td class="px-4 py-3.5 max-w-xs">
                    <div class="flex items-center gap-1.5">
                      <button onclick={() => { selectedLead = lead; showDrawer = true; }} class="text-white font-semibold truncate hover:text-zinc-300 text-left cursor-pointer">{lead.name}</button>
                      {#if lead.contacted}<LeadStatusBadge status={lead.lead_status ?? 'cold'} />{/if}
                    </div>
                    <div class="text-[11px] text-zinc-500 truncate">{lead.street || lead.address}</div>
                    {#if lead.score?.note}<div class="text-[11px] text-zinc-400 truncate" title={lead.score.note}>Why: {lead.score.note}</div>{/if}
                  </td>
                  <td class="px-4 py-3.5 whitespace-nowrap text-zinc-400">
                    {lead.score?.area_name || 'Other area'}
                    {#if lead.group_name || lead.suggested_group_name}
                      <div class="text-[11px] text-zinc-500">{lead.group_name || lead.suggested_group_name}</div>
                    {/if}
                  </td>
                  <td class="px-4 py-3.5 whitespace-nowrap">
                    <span class="inline-flex items-center gap-1 text-white"><Star class="w-3.5 h-3.5 fill-white" />{lead.rating.toFixed(1)}</span>
                    <span class="text-[10px] text-zinc-500">({lead.review_count.toLocaleString()})</span>
                  </td>
                  <td class="px-4 py-3.5 text-[11px] text-zinc-400">
                    {(lead.score?.breakdown.channels ?? []).join(', ') || '—'}
                  </td>
                  <td class="px-4 py-3.5 text-right">
                    <button onclick={() => { selectedLead = lead; showDrawer = true; }} class="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-white cursor-pointer" title="View details">
                      <Eye class="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              {/each}
            {/if}
          </tbody>
        </table>
      </div>
      <Pagination page={pageNo} {totalPages} onPageChange={(p) => { pageNo = p; fetchLeads(); }} />
    </div>
    <div class="flex items-center justify-between text-[11px] text-zinc-600">
      <span>{totalCount.toLocaleString()} shown with these filters. Sister properties (same group) are listed together.</span>
      <button onclick={() => (showExport = true)} class="px-3 py-1.5 rounded-lg border border-zinc-800 text-zinc-300 hover:text-white flex items-center gap-1.5 cursor-pointer">
        <Download class="w-3.5 h-3.5" /> Export these
      </button>
    </div>
</div>

<BusinessDetailDrawer
  lead={selectedLead}
  isOpen={showDrawer}
  onClose={() => (showDrawer = false)}
  onToggleContacted={(l, c) => toggleContacted(l, c)}
  onStatusChange={(l, s) => setStatus(l, s)}
  onScoreChanged={() => { fetchLeads(); onScoresChanged?.(); }}
/>

<ExportModal isOpen={showExport} onClose={() => (showExport = false)} {filterParams} />
