<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { auth } from '$lib/stores/auth.svelte';
  import { errorMessage } from '$lib/utils/errors';
  import type { AdminUser } from '$lib/types';
  import { Users, UserPlus, KeyRound, Power } from 'lucide-svelte';

  let admins = $state.raw<AdminUser[]>([]);
  let adding = $state(false);
  let newEmail = $state('');
  let newUsername = $state('');
  let newPassword = $state('');
  let resetFor = $state<string | null>(null);
  let resetPassword = $state('');
  let busy = $state(false);

  const me = $derived(auth.user?.adminId);
  const fmt = (s?: string) => (s ? new Date(s).toLocaleString() : 'never');

  async function load() {
    try {
      admins = (await api.get<AdminUser[]>('/api/v1/admins')) || [];
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to load admins'));
    }
  }

  async function addAdmin() {
    try {
      busy = true;
      await api.post('/api/v1/admins', { email: newEmail.trim(), username: newUsername.trim(), password: newPassword });
      toast.success(`Admin ${newEmail.trim()} added`);
      newEmail = newUsername = newPassword = '';
      adding = false;
      await load();
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to add admin'));
    } finally {
      busy = false;
    }
  }

  async function setActive(a: AdminUser, active: boolean) {
    if (!active && !confirm(`Deactivate ${a.username}? They are signed out at once and can't log in until reactivated.`)) return;
    try {
      await api.patch(`/api/v1/admins/${a.id}`, { is_active: active });
      toast.success(active ? `${a.username} reactivated` : `${a.username} deactivated`);
      await load();
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to update admin'));
    }
  }

  async function doReset(a: AdminUser) {
    try {
      busy = true;
      await api.patch(`/api/v1/admins/${a.id}`, { password: resetPassword });
      toast.success(`Password for ${a.username} changed; they need to log in again`);
      resetFor = null;
      resetPassword = '';
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to reset password'));
    } finally {
      busy = false;
    }
  }

  onMount(load);
</script>

<div class="bg-zinc-950 border border-zinc-800 rounded-2xl p-6 shadow-sm space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-base font-medium text-white flex items-center gap-2"><Users class="w-4 h-4 text-zinc-400" /> Admins</h2>
      <p class="text-xs text-zinc-500 mt-0.5">Everyone here can use the whole app. Every change they make is recorded in the <a href="/audit" class="underline hover:text-white">audit log</a>.</p>
    </div>
    {#if !adding}
      <button onclick={() => (adding = true)} class="px-3 py-1.5 rounded-xl bg-white text-black text-xs font-semibold flex items-center gap-1.5 cursor-pointer">
        <UserPlus class="w-3.5 h-3.5" /> Add admin
      </button>
    {/if}
  </div>

  {#if adding}
    <form onsubmit={(e) => { e.preventDefault(); addAdmin(); }} class="grid sm:grid-cols-4 gap-2 text-xs">
      <input bind:value={newEmail} type="email" required placeholder="Email" class="px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white placeholder-zinc-600" />
      <input bind:value={newUsername} placeholder="Username (optional)" class="px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white placeholder-zinc-600" />
      <input bind:value={newPassword} type="password" required minlength="8" placeholder="Password (8+ characters)" autocomplete="new-password" class="px-3 py-2 bg-black border border-zinc-800 rounded-xl text-white placeholder-zinc-600" />
      <div class="flex gap-2">
        <button type="submit" disabled={busy} class="flex-1 px-3 py-2 rounded-xl bg-white text-black font-semibold disabled:opacity-50 cursor-pointer">{busy ? 'Adding…' : 'Add'}</button>
        <button type="button" onclick={() => (adding = false)} class="px-3 py-2 rounded-xl bg-zinc-900 text-zinc-400 cursor-pointer">Cancel</button>
      </div>
    </form>
  {/if}

  <div class="divide-y divide-zinc-900 text-xs">
    {#each admins as a (a.id)}
      <div class="py-2.5 flex flex-wrap items-center gap-3">
        <div class="flex-1 min-w-[180px]">
          <div class="text-white font-medium">
            {a.username}
            {#if a.id === me}<span class="ml-1 text-[10px] text-zinc-500">(you)</span>{/if}
            {#if !a.is_active}<span class="ml-1 text-[10px] px-1.5 py-0.5 rounded-full bg-zinc-900 text-zinc-500 border border-zinc-800">inactive</span>{/if}
          </div>
          <div class="text-zinc-500">{a.email} · last login {fmt(a.last_login_at)}</div>
        </div>
        {#if resetFor === a.id}
          <form onsubmit={(e) => { e.preventDefault(); doReset(a); }} class="flex gap-2">
            <input bind:value={resetPassword} type="password" required minlength="8" placeholder="New password" autocomplete="new-password" class="px-2 py-1.5 bg-black border border-zinc-800 rounded-lg text-white placeholder-zinc-600" />
            <button type="submit" disabled={busy} class="px-2.5 py-1.5 rounded-lg bg-white text-black font-semibold disabled:opacity-50 cursor-pointer">Set</button>
            <button type="button" onclick={() => (resetFor = null)} class="px-2.5 py-1.5 rounded-lg bg-zinc-900 text-zinc-400 cursor-pointer">Cancel</button>
          </form>
        {:else}
          <button onclick={() => { resetFor = a.id; resetPassword = ''; }} class="px-2.5 py-1.5 rounded-lg border border-zinc-800 text-zinc-300 hover:text-white flex items-center gap-1 cursor-pointer">
            <KeyRound class="w-3.5 h-3.5" /> Reset password
          </button>
          {#if a.id !== me}
            <button onclick={() => setActive(a, !a.is_active)} class="px-2.5 py-1.5 rounded-lg border border-zinc-800 flex items-center gap-1 cursor-pointer {a.is_active ? 'text-zinc-400 hover:text-red-300' : 'text-emerald-400 hover:text-emerald-300'}">
              <Power class="w-3.5 h-3.5" /> {a.is_active ? 'Deactivate' : 'Reactivate'}
            </button>
          {/if}
        {/if}
      </div>
    {/each}
  </div>
</div>
