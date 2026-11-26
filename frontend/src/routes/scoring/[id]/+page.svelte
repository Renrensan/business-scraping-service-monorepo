<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { ProfileSummary, ScoringProfile } from '$lib/types';
  import ProspectsList from '$lib/components/ProspectsList.svelte';
  import ScoringProfileEditor, { type ScoringOptions } from '$lib/components/ScoringProfileEditor.svelte';
  import { Trophy, Plus, RefreshCw } from 'lucide-svelte';

  type Tab = 'prospects' | 'settings';

  const profileId = $derived(page.params.id ?? '');
  const tab = $derived<Tab>(page.url.searchParams.get('tab') === 'settings' ? 'settings' : 'prospects');

  let profiles = $state.raw<ScoringProfile[]>([]);
  let profile = $state.raw<ScoringProfile | null>(null);
  let summary = $state.raw<ProfileSummary | null>(null);
  let options = $state.raw<ScoringOptions | null>(null);
  let creating = $state(false);
  let notFound = $state(false);
  let rescoring = $state(false);

  async function loadProfiles() {
    profiles = (await api.get<ScoringProfile[]>('/api/v1/scoring/profiles')) || [];
  }

  async function loadProfile(id: string) {
    try {
      notFound = false;
      [profile, summary] = await Promise.all([
        api.get<ScoringProfile>(`/api/v1/scoring/profiles/${id}`),
        api.get<ProfileSummary>(`/api/v1/scoring/profiles/${id}/summary`)
      ]);
    } catch {
      notFound = true;
      profile = null;
    }
  }

  async function reloadSummary() {
    summary = (await api.get<ProfileSummary>(`/api/v1/scoring/profiles/${profileId}/summary`).catch(() => null)) ?? summary;
  }

  function setTab(t: Tab) {
    goto(t === 'settings' ? `?tab=settings` : `/scoring/${profileId}`, { replaceState: true, noScroll: true, keepFocus: true });
  }

  async function rescore() {
    try {
      rescoring = true;
      await api.post('/api/v1/scoring/rescore', {});
      toast.success('All leads rescored');
      await loadProfile(profileId);
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to rescore'));
    } finally {
      rescoring = false;
    }
  }

  async function onSaved(p: ScoringProfile) {
    creating = false;
    await loadProfiles();
    if (p.id === profileId) await loadProfile(p.id);
    else goto(`/scoring/${p.id}`);
  }

  async function onDeleted() {
    await loadProfiles();
    goto('/scoring');
  }

  $effect(() => {
    creating = false;
    loadProfile(profileId);
  });

  $effect(() => {
    loadProfiles().catch(() => {});
    api.get<ScoringOptions>('/api/v1/scoring/options').then((o) => (options = o)).catch(() => {});
  });
</script>

<div class="space-y-5">
  <!-- Header: pick the product, then see its prospects or how it's scored -->
  <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4 pb-2 border-b border-zinc-900">
    <div class="space-y-1">
      <div class="flex items-center gap-2">
        <Trophy class="w-5 h-5 text-zinc-400" />
        <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Scoring</h1>
        <select
          value={creating ? '' : profileId}
          onchange={(e) => goto(`/scoring/${e.currentTarget.value}${tab === 'settings' ? '?tab=settings' : ''}`)}
          class="ml-2 bg-zinc-900 border border-zinc-700 rounded-xl px-3 py-1.5 text-white text-sm font-medium cursor-pointer"
          aria-label="Scoring profile"
        >
          {#if creating}<option value="">New profile…</option>{/if}
          {#each profiles as p (p.id)}
            <option value={p.id}>{p.name}{p.is_active ? '' : ' (inactive)'}</option>
          {/each}
        </select>
      </div>
      {#if profile && !creating}
        <p class="text-xs text-zinc-500">
          {profile.description || 'Ranks leads into HOT, GOOD and LOW for this product.'}
          · {summary?.total.toLocaleString() ?? '…'} leads{profile.applies_to_types.length ? ` (${profile.applies_to_types.join(', ')})` : ''}
        </p>
      {/if}
    </div>
    <div class="flex gap-2 shrink-0 whitespace-nowrap">
      <button onclick={rescore} disabled={rescoring} class="px-3.5 py-2 rounded-xl border border-zinc-800 bg-zinc-950 text-zinc-300 hover:text-white text-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-50">
        <RefreshCw class="w-3.5 h-3.5 {rescoring ? 'animate-spin' : ''}" /> Rescore all
      </button>
      <button onclick={() => (creating = true)} disabled={!options || creating} class="px-4 py-2 rounded-xl bg-white text-black text-xs font-semibold flex items-center gap-1.5 cursor-pointer disabled:opacity-50">
        <Plus class="w-4 h-4" /> New profile
      </button>
    </div>
  </div>

  {#if creating && options}
    <ScoringProfileEditor profile={null} {options} {onSaved} onCancel={() => (creating = false)} />
  {:else if notFound}
    <p class="py-12 text-center text-sm text-zinc-500">This scoring profile doesn't exist. Pick another one above.</p>
  {:else if profile}
    <!-- Tabs -->
    <div class="flex gap-1 border-b border-zinc-900 text-xs">
      {#each [['prospects', 'Prospects'], ['settings', 'How it’s scored']] as [t, label] (t)}
        <button
          onclick={() => setTab(t as Tab)}
          class="px-4 py-2 -mb-px border-b-2 cursor-pointer transition {tab === t ? 'border-white text-white font-semibold' : 'border-transparent text-zinc-500 hover:text-zinc-300'}"
        >{label}</button>
      {/each}
    </div>

    {#if tab === 'prospects'}
      <ProspectsList {profile} {summary} onScoresChanged={reloadSummary} />
    {:else if options}
      <div class="max-w-4xl">
        <ScoringProfileEditor {profile} {options} {onSaved} {onDeleted} />
      </div>
    {/if}
  {/if}
</div>
