<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { ScoringProfile, ScoringConfig } from '$lib/types';
  import { GROUP_TYPE_LABELS } from '$lib/types';
  import { Trash2, Save } from 'lucide-svelte';

  export type ScoringOptions = { channels: string[]; group_types: string[]; tiers: string[]; default_config: ScoringConfig };

  type Draft = {
    id?: string;
    name: string;
    description: string;
    appliesTo: string;
    is_active: boolean;
    config: ScoringConfig;
    areas: { name: string; hot: number; good: number }[];
  };

  /** Edits a scoring profile, or creates one when profile is null. */
  let {
    profile,
    options,
    onSaved,
    onDeleted,
    onCancel
  }: {
    profile: ScoringProfile | null;
    options: ScoringOptions;
    onSaved: (p: ScoringProfile) => void;
    onDeleted?: () => void;
    onCancel?: () => void;
  } = $props();

  const CHANNEL_LABELS: Record<string, string> = {
    email: 'Email', website: 'Website', instagram: 'Instagram', facebook: 'Facebook', tiktok: 'TikTok',
    linkedin: 'LinkedIn', twitter: 'Twitter / X', youtube: 'YouTube', phone: 'Phone'
  };

  function toDraft(p: ScoringProfile | null): Draft {
    const config: ScoringConfig = structuredClone(p ? p.config : options.default_config);
    const areas = Object.entries(config.area_cutoffs || {})
      .map(([name, c]) => ({ name, hot: c.hot, good: c.good }))
      .sort((a, b) => a.name.localeCompare(b.name));
    if (!p) return { name: '', description: '', appliesTo: '', is_active: true, config, areas: [] };
    return { id: p.id, name: p.name, description: p.description, appliesTo: p.applies_to_types.join(', '), is_active: p.is_active, config, areas };
  }

  // A fresh, deeply reactive copy of the profile to edit; recreated when a different profile is
  // passed in. ($state inside $derived.by: a plain $derived object would not react to field edits.)
  let draft = $derived.by(() => {
    const d = $state(toDraft(profile));
    return d;
  });
  let saving = $state(false);

  function toggleChannel(ch: string) {
    const c = draft.config.channels;
    draft.config.channels = c.includes(ch) ? c.filter((x) => x !== ch) : [...c, ch];
  }

  async function save() {
    const config: ScoringConfig = {
      ...draft.config,
      area_cutoffs: Object.fromEntries(
        draft.areas.filter((a) => a.name.trim()).map((a) => [a.name.trim(), { hot: Number(a.hot), good: Number(a.good) }])
      )
    };
    const body = {
      name: draft.name,
      description: draft.description,
      applies_to_types: draft.appliesTo.split(',').map((s) => s.trim()).filter(Boolean),
      is_active: draft.is_active,
      config
    };
    try {
      saving = true;
      const saved = draft.id
        ? await api.put<ScoringProfile>(`/api/v1/scoring/profiles/${draft.id}`, body)
        : await api.post<ScoringProfile>('/api/v1/scoring/profiles', body);
      toast.success('Saved. All leads were rescored.');
      if (saved) onSaved(saved);
    } catch (e) {
      toast.error(errorMessage(e, 'Failed to save'));
    } finally {
      saving = false;
    }
  }

  async function remove() {
    if (!draft.id || !confirm(`Delete the "${draft.name}" scoring profile? Its scores, tier overrides and notes are removed.`)) return;
    try {
      await api.delete(`/api/v1/scoring/profiles/${draft.id}`);
      toast.success('Deleted');
      onDeleted?.();
    } catch (e) {
      toast.error(errorMessage(e, 'Failed to delete'));
    }
  }
</script>

  <!-- Editor -->
  <form onsubmit={(e) => { e.preventDefault(); save(); }} class="bg-zinc-950 border border-zinc-800 rounded-2xl p-6 space-y-6 text-xs">
    <div class="grid sm:grid-cols-2 gap-4">
      <label class="space-y-1 block">
        <span class="text-zinc-400">Name</span>
        <input bind:value={draft.name} required placeholder="e.g. GuestChatPro" class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" />
      </label>
      <label class="space-y-1 block">
        <span class="text-zinc-400">Applies to place types <span class="text-zinc-600">(comma-separated, empty = all)</span></span>
        <input bind:value={draft.appliesTo} placeholder="e.g. Lodging" class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" />
      </label>
      <label class="space-y-1 block sm:col-span-2">
        <span class="text-zinc-400">Description</span>
        <input bind:value={draft.description} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" />
      </label>
      <label class="flex items-center gap-2 text-zinc-300">
        <input type="checkbox" bind:checked={draft.is_active} class="accent-white" /> Active (inactive profiles are not scored)
      </label>
    </div>

    <section class="space-y-2">
      <h3 class="text-sm text-white font-medium">Score = reviews + rating + channels + group</h3>
      <div class="grid sm:grid-cols-3 gap-3">
        <label class="space-y-1 block"><span class="text-zinc-400">Reviews: points per ×10 reviews</span>
          <input type="number" step="any" bind:value={draft.config.reviews_weight} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
        <label class="space-y-1 block"><span class="text-zinc-400">Rating: neutral rating</span>
          <input type="number" step="any" bind:value={draft.config.rating_baseline} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
        <label class="space-y-1 block"><span class="text-zinc-400">Rating: points per star above it</span>
          <input type="number" step="any" bind:value={draft.config.rating_weight} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
        <label class="space-y-1 block"><span class="text-zinc-400">Points per contact channel</span>
          <input type="number" step="any" bind:value={draft.config.channel_points} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
        <label class="space-y-1 block"><span class="text-zinc-400">Channel points cap (0 = none)</span>
          <input type="number" step="any" bind:value={draft.config.channel_cap} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
      </div>
      <div class="flex flex-wrap gap-2 pt-1">
        <span class="text-zinc-400 mr-1">Channels that count:</span>
        {#each options?.channels || [] as ch (ch)}
          <label class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border cursor-pointer {draft.config.channels.includes(ch) ? 'bg-white/10 border-white/30 text-white' : 'border-zinc-800 text-zinc-500'}">
            <input type="checkbox" checked={draft.config.channels.includes(ch)} onchange={() => toggleChannel(ch)} class="accent-white" />
            {CHANNEL_LABELS[ch] || ch}
          </label>
        {/each}
      </div>
      <label class="flex items-center gap-2 text-zinc-300">
        <input type="checkbox" bind:checked={draft.config.ignore_junk_contacts} class="accent-white" />
        Don't count junk contacts (template/system emails, booking & link platforms, broken links)
      </label>
    </section>

    <section class="space-y-2">
      <h3 class="text-sm text-white font-medium">Group bonus</h3>
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {#each options?.group_types || [] as g (g)}
          <label class="space-y-1 block"><span class="text-zinc-400">{GROUP_TYPE_LABELS[g as keyof typeof GROUP_TYPE_LABELS] || g}</span>
            <input type="number" step="any" bind:value={draft.config.group_bonuses[g]} class="w-full px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white" /></label>
        {/each}
      </div>
      <label class="flex items-center gap-2 text-zinc-300">
        <input type="checkbox" bind:checked={draft.config.chain_forces_low} class="accent-white" /> Chains are always LOW, whatever their score
      </label>
    </section>

    <section class="space-y-2">
      <h3 class="text-sm text-white font-medium">Tier cut-offs</h3>
      <p class="text-zinc-500">A lead is HOT from the HOT score, GOOD from the GOOD score, LOW below. Areas without their own row use the default.</p>
      <div class="grid grid-cols-[1fr_90px_90px_32px] gap-2 items-center">
        <span class="text-zinc-500">Area</span><span class="text-zinc-500">🔥 HOT from</span><span class="text-zinc-500">⭐ GOOD from</span><span></span>
        <span class="text-zinc-300 font-medium">Default (other areas)</span>
        <input type="number" bind:value={draft.config.default_cutoff.hot} class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white" />
        <input type="number" bind:value={draft.config.default_cutoff.good} class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white" />
        <span></span>
        {#each draft.areas as a, i}
          <input bind:value={a.name} placeholder="Area name (as in Areas)" class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white" />
          <input type="number" bind:value={a.hot} class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white" />
          <input type="number" bind:value={a.good} class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white" />
          <button type="button" onclick={() => draft && (draft.areas = draft.areas.filter((_, j) => j !== i))} class="text-zinc-500 hover:text-red-300 cursor-pointer" aria-label="Remove area"><Trash2 class="w-3.5 h-3.5" /></button>
        {/each}
      </div>
      <button type="button" onclick={() => draft && (draft.areas = [...draft.areas, { name: '', hot: draft.config.default_cutoff.hot, good: draft.config.default_cutoff.good }])} class="text-zinc-400 hover:text-white cursor-pointer">+ Add area</button>
    </section>

    <div class="flex justify-end gap-3 pt-3 border-t border-zinc-900">
      {#if draft.id}
        <button type="button" onclick={remove} class="mr-auto px-3 py-2 rounded-xl border border-zinc-800 text-zinc-500 hover:text-red-300 flex items-center gap-1.5 cursor-pointer">
          <Trash2 class="w-3.5 h-3.5" /> Delete profile
        </button>
      {/if}
      {#if onCancel}<button type="button" onclick={onCancel} class="px-4 py-2 rounded-xl bg-zinc-900 text-zinc-400 hover:text-white cursor-pointer">Cancel</button>{/if}
      <button type="submit" disabled={saving} class="px-5 py-2.5 rounded-xl bg-white text-black font-semibold flex items-center gap-1.5 disabled:opacity-50 cursor-pointer">
        <Save class="w-4 h-4" /> {saving ? 'Saving & rescoring…' : 'Save'}
      </button>
    </div>
  </form>
