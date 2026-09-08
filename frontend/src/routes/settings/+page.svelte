<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { toast } from 'svelte-sonner';
  import { Settings, Shield, Zap, Server, Key, Check } from 'lucide-svelte';

  let health = $state<{ status: string; database: string; redis: string }>({
    status: 'checking...',
    database: 'checking...',
    redis: 'checking...'
  });

  let currentPassword = $state('');
  let newPassword = $state('');
  let isUpdatingPassword = $state(false);

  async function loadSystemStatus() {
    try {
      const res = await api.get<{ status: string; database: string; redis: string }>('/api/v1/system/health');
      if (res) health = res;
    } catch (e) {
      console.error('System health check error', e);
    }
  }

  async function handlePasswordChange() {
    if (!newPassword || newPassword.length < 6) {
      toast.error('New password must be at least 6 characters');
      return;
    }
    try {
      isUpdatingPassword = true;
      await api.post('/api/v1/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword
      });
      toast.success('Password updated successfully');
      currentPassword = '';
      newPassword = '';
    } catch (err: any) {
      toast.error(err.message || 'Failed to update password');
    } finally {
      isUpdatingPassword = false;
    }
  }

  onMount(() => {
    loadSystemStatus();
  });
</script>

<div class="max-w-4xl mx-auto space-y-6">
  <div class="pb-2 border-b border-zinc-900">
    <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">System & Security</h1>
    <p class="text-xs font-mono text-zinc-500 mt-0.5">Configure extraction engines, storage health, and administrator credentials</p>
  </div>

  <!-- System Health & Quota -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-zinc-800/80">
      <Server class="w-4 h-4 text-zinc-400" />
      <h2 class="text-base font-display font-light text-white">System Health & Storage Infrastructure</h2>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
      <div class="p-3.5 rounded-xl bg-black border border-zinc-800">
        <div class="text-[10px] uppercase text-zinc-500 tracking-wider">PostgreSQL Status</div>
        <div class="text-emerald-400 font-mono font-medium mt-1">{health.database}</div>
      </div>
      <div class="p-3.5 rounded-xl bg-black border border-zinc-800">
        <div class="text-[10px] uppercase text-zinc-500 tracking-wider">Redis Session Store</div>
        <div class="text-emerald-400 font-mono font-medium mt-1">{health.redis}</div>
      </div>
      <div class="p-3.5 rounded-xl bg-black border border-zinc-800">
        <div class="text-[10px] uppercase text-zinc-500 tracking-wider">Daily Rate Budget</div>
        <div class="text-white font-mono font-medium mt-1">160 calls/day (Free Tier)</div>
      </div>
    </div>
  </div>

  <!-- Admin Account Security -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-zinc-800/80">
      <Shield class="w-4 h-4 text-emerald-400" />
      <h2 class="text-base font-display font-light text-white">Admin Credentials & Authentication</h2>
    </div>

    <form onsubmit={(e) => { e.preventDefault(); handlePasswordChange(); }} class="space-y-4 text-xs font-mono max-w-md">
      <div>
        <label class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">Current Password</label>
        <input
          type="password"
          bind:value={currentPassword}
          required
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <div>
        <label class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">New Password</label>
        <input
          type="password"
          bind:value={newPassword}
          required
          placeholder="Min 6 characters"
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-zinc-400 text-xs"
        />
      </div>

      <button
        type="submit"
        disabled={isUpdatingPassword}
        class="px-5 py-2.5 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold text-xs disabled:opacity-50 transition cursor-pointer"
      >
        {isUpdatingPassword ? 'Saving...' : 'Update Password'}
      </button>
    </form>
  </div>
</div>