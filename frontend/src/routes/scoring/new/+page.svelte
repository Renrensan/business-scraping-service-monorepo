<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { errorMessage } from '$lib/utils/errors';
  import { scoringProfiles } from '$lib/stores/scoringProfiles.svelte';
  import ScoringProfileEditor, { type ScoringOptions } from '$lib/components/ScoringProfileEditor.svelte';
  import type { ScoringProfile } from '$lib/types';
  import { Trophy } from 'lucide-svelte';

  let options = $state.raw<ScoringOptions | null>(null);

  onMount(async () => {
    try {
      options = await api.get<ScoringOptions>('/api/v1/scoring/options');
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load scoring options'));
    }
  });

  async function onSaved(p: ScoringProfile) {
    await scoringProfiles.load().catch(() => {});
    goto(`/scoring/${p.id}`);
  }
</script>

<div class="max-w-4xl space-y-5">
  <div class="pb-2 border-b border-zinc-900">
    <p class="text-[11px] text-zinc-500 flex items-center gap-1.5"><Trophy class="w-3.5 h-3.5" /> Scoring</p>
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">New scoring profile</h1>
    <p class="text-xs text-zinc-500 mt-0.5">One profile per product you sell. Leads are ranked into HOT, GOOD and LOW for it.</p>
  </div>
  {#if options}
    <ScoringProfileEditor
      profile={null}
      {options}
      {onSaved}
      onCancel={() => (scoringProfiles.first ? goto(`/scoring/${scoringProfiles.first.id}`) : history.back())}
    />
  {/if}
</div>
