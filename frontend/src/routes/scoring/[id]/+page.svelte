<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import type { ProfileSummary, ScoringProfile } from '$lib/types';
  import ProspectsList from '$lib/components/ProspectsList.svelte';
  import ScoringProfileEditor, { type ScoringOptions } from '$lib/components/ScoringProfileEditor.svelte';
  import { scoringProfiles } from '$lib/stores/scoringProfiles.svelte';
  import { Trophy, RefreshCw } from 'lucide-svelte';

  type Tab = 'prospects' | 'settings';

  const profileId = $derived(page.params.id ?? '');
  const tab = $derived<Tab>(page.url.searchParams.get('tab') === 'settings' ? 'settings' : 'prospects');

  let profile = $state.raw<ScoringProfile | null>(null);
  let summary = $state.raw<ProfileSummary | null>(null);
  let options = $state.raw<ScoringOptions | null>(null);
  let notFound = $state(false);
  let rescoring = $state(false);

  const loadProfiles = () => scoringProfiles.load().catch(() => []);

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
    await loadProfiles();
    await loadProfile(p.id);
  }

  async function onDeleted() {
    await loadProfiles();
    goto('/scoring');
  }

  $effect(() => {
    loadProfile(profileId);
  });

  $effect(() => {
    api.get<ScoringOptions>('/api/v1/scoring/options').then((o) => (options = o)).catch(() => {});
  });
</script>

<div class="space-y-5">
  <!-- Header: pick the product, then see its prospects or how it's scored -->
  <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4 pb-2 border-b border-zinc-900">
    <div class="space-y-1">
      <p class="text-[11px] text-zinc-500 flex items-center gap-1.5"><Trophy class="w-3.5 h-3.5" /> Scoring</p>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">
        {profile?.name ?? '…'}
        {#if profile && !profile.is_active}<span class="ml-2 align-middle text-[10px] px-2 py-0.5 rounded-full bg-zinc-900 text-zinc-500 border border-zinc-800">inactive</span>{/if}
      </h1>
      {#if profile}
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
    </div>
  </div>

  {#if notFound}
    <p class="py-12 text-center text-sm text-zinc-500">This scoring profile doesn't exist. Pick another one under Scoring in the sidebar.</p>
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
