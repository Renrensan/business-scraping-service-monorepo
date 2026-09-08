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
  <div>
    <h1 class="text-xl font-bold text-white tracking-tight">System & Account Settings</h1>
    <p class="text-xs text-slate-400 mt-1">Configure extraction engines, rate limits, and administrator credentials</p>
  </div>

  <!-- System Health & Quota -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-slate-800">
      <Server class="w-4 h-4 text-indigo-400" />
      <h2 class="text-sm font-bold text-white">System Health & Storage Containers</h2>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
      <div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800">
        <div class="text-[11px] text-slate-400">PostgreSQL Status</div>
        <div class="text-emerald-400 font-mono font-medium mt-1">{health.database}</div>
      </div>
      <div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800">
        <div class="text-[11px] text-slate-400">Redis Cache & Sessions</div>
        <div class="text-emerald-400 font-mono font-medium mt-1">{health.redis}</div>
      </div>
      <div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800">
        <div class="text-[11px] text-slate-400">Daily Google API Guard</div>
        <div class="text-indigo-400 font-mono font-medium mt-1">160 calls/day (100% Free)</div>
      </div>
    </div>
  </div>

  <!-- Admin Account Security -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-4">
    <div class="flex items-center gap-2.5 pb-3 border-b border-slate-800">
      <Shield class="w-4 h-4 text-emerald-400" />
      <h2 class="text-sm font-bold text-white">Admin Account Security</h2>
    </div>

    <form onsubmit={(e) => { e.preventDefault(); handlePasswordChange(); }} class="space-y-4 text-xs max-w-md">
      <div>
        <label class="block font-medium text-slate-300 mb-1">Current Password</label>
        <input
          type="password"
          bind:value={currentPassword}
          required
          class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <div>
        <label class="block font-medium text-slate-300 mb-1">New Password</label>
        <input
          type="password"
          bind:value={newPassword}
          required
          placeholder="Min 6 characters"
          class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white focus:outline-hidden focus:border-indigo-500 text-xs"
        />
      </div>

      <button
        type="submit"
        disabled={isUpdatingPassword}
        class="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs disabled:opacity-50 transition cursor-pointer"
      >
        {isUpdatingPassword ? 'Saving...' : 'Update Admin Password'}
      </button>
    </form>
  </div>
</div>