<script lang="ts">
  import { errorMessage } from '$lib/utils/errors';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { BusinessLead, LeadScore, GroupType, Tier } from '$lib/types';
  import { GROUP_TYPE_LABELS } from '$lib/types';
  import TierBadge from '$lib/components/TierBadge.svelte';
  import { Trophy, AlertTriangle } from 'lucide-svelte';

  let { leadId, onChanged }: { leadId: string; onChanged?: () => void } = $props();

  let lead = $state<BusinessLead | null>(null);
  let scores = $state.raw<LeadScore[]>([]);
  let groupType = $state<string>('');
  let groupName = $state('');
  let notes = $state<Record<string, string>>({});
  let saving = $state(false);

  async function load(id: string) {
    try {
      const res = await api.get<BusinessLead>(`/api/v1/businesses/${id}`);
      if (!res || res.id !== leadId) return;
      lead = res;
      scores = res.scores || [];
      groupType = res.group_type ?? '';
      groupName = res.group_name ?? '';
      notes = Object.fromEntries(scores.map((s) => [s.profile_id, s.note]));
    } catch {
      /* the drawer already shows the lead */
    }
  }

  $effect(() => {
    if (leadId) load(leadId);
  });

  const suggestedLabel = $derived(
    lead?.suggested_group_type ? GROUP_TYPE_LABELS[lead.suggested_group_type] + (lead.suggested_group_name ? ` (${lead.suggested_group_name})` : '') : ''
  );

  async function saveGrouping() {
    try {
      saving = true;
      scores = (await api.patch<LeadScore[]>(`/api/v1/businesses/${leadId}/grouping`, { group_type: groupType, group_name: groupName })) || [];
      toast.success('Group saved, score updated');
      onChanged?.();
    } catch (e) {
      toast.error(errorMessage(e, 'Failed to save group'));
    } finally {
      saving = false;
    }
  }

  async function saveOverride(s: LeadScore, tier: string | null) {
    try {
      scores = (await api.patch<LeadScore[]>(`/api/v1/businesses/${leadId}/scores/${s.profile_id}`, {
        tier_override: tier ?? '',
        note: notes[s.profile_id] ?? ''
      })) || [];
      toast.success('Saved');
      onChanged?.();
    } catch (e) {
      toast.error(errorMessage(e, 'Failed to save'));
    }
  }

  const pts = (n: number) => (n >= 0 ? '+' : '') + (Math.round(n * 10) / 10).toString();
</script>

{#if lead && scores.length > 0}
  <div class="mt-4 space-y-3">
    {#each scores as s (s.profile_id)}
      <div class="p-3 rounded-xl bg-black border border-zinc-800 space-y-2.5 text-xs">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Trophy class="w-4 h-4 text-zinc-400" />
            <span class="text-white font-medium">{s.profile_name}</span>
            <TierBadge tier={s.tier} score={s.score} overridden={!!s.tier_override} />
          </div>
          <span class="text-[10px] text-zinc-500">{s.area_name || 'no area'} · HOT ≥{s.breakdown.cutoff.hot} · GOOD ≥{s.breakdown.cutoff.good}</span>
        </div>

        <!-- Breakdown, like the sheet's columns -->
        <div class="grid grid-cols-4 gap-1.5 text-center">
          <div class="bg-zinc-900 rounded-lg py-1.5"><div class="text-zinc-500 text-[10px]">Reviews</div><div class="font-mono text-white">{pts(s.breakdown.review_points)}</div></div>
          <div class="bg-zinc-900 rounded-lg py-1.5" title={s.breakdown.no_rating ? 'No rating: counted as 0' : ''}><div class="text-zinc-500 text-[10px]">Rating</div><div class="font-mono text-white">{s.breakdown.no_rating ? '—' : pts(s.breakdown.rating_points)}</div></div>
          <div class="bg-zinc-900 rounded-lg py-1.5" title={(s.breakdown.channels || []).join(', ')}><div class="text-zinc-500 text-[10px]">Channels</div><div class="font-mono text-white">{pts(s.breakdown.channel_points)}</div></div>
          <div class="bg-zinc-900 rounded-lg py-1.5"><div class="text-zinc-500 text-[10px]">Group</div><div class="font-mono text-white">{pts(s.breakdown.group_bonus)}</div></div>
        </div>
        <div class="text-[11px] text-zinc-500">
          Counted: {(s.breakdown.channels || []).join(', ') || 'none'}
          {#if s.breakdown.tier_reason}<div class="text-zinc-400 mt-0.5">{s.breakdown.tier_reason}</div>{/if}
        </div>
        {#if s.breakdown.ignored?.length}
          <div class="text-[11px] text-amber-300/90 space-y-0.5">
            <div class="flex items-center gap-1"><AlertTriangle class="w-3 h-3" /> Data check: not counted</div>
            {#each s.breakdown.ignored as ig}
              <div class="text-zinc-500 truncate" title={ig.value}>{ig.channel}: {ig.value} <span class="text-zinc-600">({ig.reason})</span></div>
            {/each}
          </div>
        {/if}

        <!-- Manual tier + why -->
        <div class="flex items-center gap-2">
          <span class="text-zinc-500 text-[11px] w-12">Tier</span>
          <select
            value={s.tier_override ?? ''}
            onchange={(e) => saveOverride(s, e.currentTarget.value || null)}
            class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2 py-1 text-xs text-zinc-200 cursor-pointer"
          >
            <option value="">Automatic ({s.computed_tier.toUpperCase()})</option>
            {#each ['hot', 'good', 'low', 'skip'] as t (t)}
              <option value={t}>{t.toUpperCase()} (by hand)</option>
            {/each}
          </select>
        </div>
        <div class="flex items-start gap-2">
          <span class="text-zinc-500 text-[11px] w-12 pt-1">Why</span>
          <textarea
            rows="2"
            bind:value={notes[s.profile_id]}
            onblur={() => notes[s.profile_id] !== s.note && saveOverride(s, s.tier_override)}
            placeholder="e.g. Pitch Paradiso once for both"
            class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2 py-1 text-xs text-zinc-200 placeholder-zinc-600"
          ></textarea>
        </div>
      </div>
    {/each}

    <!-- Group (shared by all profiles) -->
    <div class="p-3 rounded-xl bg-black border border-zinc-800 space-y-2 text-xs">
      <div class="text-[10px] uppercase tracking-wider text-zinc-500 font-semibold">Group</div>
      <div class="flex gap-2">
        <select bind:value={groupType} class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2 py-1.5 text-xs text-zinc-200 cursor-pointer">
          <option value="">Suggested: {suggestedLabel || 'Independent'}</option>
          {#each Object.entries(GROUP_TYPE_LABELS) as [value, label]}
            <option {value}>{label}</option>
          {/each}
        </select>
        <input
          bind:value={groupName}
          placeholder={lead.suggested_group_name || 'Group name (links sister properties)'}
          class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2 py-1.5 text-xs text-zinc-200 placeholder-zinc-600"
        />
      </div>
      <div class="flex justify-between items-center">
        <span class="text-[10px] text-zinc-500">Properties with the same group name are listed together.</span>
        <button onclick={saveGrouping} disabled={saving} class="px-3 py-1 rounded-lg bg-white text-black text-[11px] font-semibold disabled:opacity-50 cursor-pointer">
          {saving ? 'Saving…' : 'Save group'}
        </button>
      </div>
    </div>
  </div>
{/if}
