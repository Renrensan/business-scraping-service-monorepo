<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Key } from 'lucide-svelte';

  let {
    isOpen = false,
    onClose
  }: {
    isOpen: boolean;
    onClose: () => void;
  } = $props();

  let currentPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let isSubmitting = $state(false);

  async function handleSubmit() {
    if (newPassword !== confirmPassword) {
      toast.error('New passwords do not match');
      return;
    }
    if (newPassword.length < 6) {
      toast.error('Password must be at least 6 characters');
      return;
    }

    try {
      isSubmitting = true;
      await api.post('/api/v1/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword
      });
      toast.success('Password updated successfully');
      onClose();
      currentPassword = '';
      newPassword = '';
      confirmPassword = '';
    } catch (err: any) {
      toast.error(err.message || 'Failed to update password');
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-xs">
    <div class="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-sm overflow-hidden shadow-2xl">
      <div class="flex items-center justify-between px-6 py-4 border-b border-slate-800">
        <div class="flex items-center gap-2">
          <Key class="w-4 h-4 text-indigo-400" />
          <h2 class="text-sm font-bold text-white">Change Admin Password</h2>
        </div>
        <button onclick={onClose} class="text-slate-400 hover:text-white">
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4 text-xs">
        <div>
          <label class="block font-medium text-slate-300 mb-1">Current Password</label>
          <input
            type="password"
            bind:value={currentPassword}
            required
            class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white text-xs focus:outline-hidden focus:border-indigo-500"
          />
        </div>

        <div>
          <label class="block font-medium text-slate-300 mb-1">New Password</label>
          <input
            type="password"
            bind:value={newPassword}
            required
            class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white text-xs focus:outline-hidden focus:border-indigo-500"
          />
        </div>

        <div>
          <label class="block font-medium text-slate-300 mb-1">Confirm New Password</label>
          <input
            type="password"
            bind:value={confirmPassword}
            required
            class="w-full bg-slate-800 border border-slate-700 rounded-xl px-3.5 py-2 text-white text-xs focus:outline-hidden focus:border-indigo-500"
          />
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-slate-800">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl border border-slate-700 bg-slate-800 text-slate-300 hover:bg-slate-700"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium"
          >
            {isSubmitting ? 'Updating...' : 'Update Password'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
