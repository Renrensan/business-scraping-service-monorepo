<script lang="ts">
  import { errorMessage } from '$lib/utils/errors';
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import { X, Key, Eye, EyeOff } from 'lucide-svelte';

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
  let showCurrentPassword = $state(false);
  let showNewPassword = $state(false);
  let showConfirmPassword = $state(false);
  let isSubmitting = $state(false);

  async function handleSubmit() {
    if (newPassword !== confirmPassword) {
      toast.error('New passwords do not match');
      return;
    }
    if (newPassword.length < 8) {
      toast.error('Password must be at least 8 characters');
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
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to update password'));
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl w-full max-w-sm overflow-hidden shadow-2xl font-mono">
      <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <Key class="w-4 h-4 text-zinc-400" />
          <h2 class="text-base font-display font-light text-white">Change Password</h2>
        </div>
        <button onclick={onClose} class="text-zinc-500 hover:text-white transition cursor-pointer">
          <X class="w-4 h-4" />
        </button>
      </div>

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-4 text-xs">
        <div>
          <label for="modal-current-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">Current Password</label>
          <div class="relative">
            <input
              id="modal-current-password"
              type={showCurrentPassword ? 'text' : 'password'}
              bind:value={currentPassword}
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-3.5 pr-10 py-2 text-white text-xs focus:outline-hidden focus:border-zinc-400"
            />
            <button
              type="button"
              onclick={() => (showCurrentPassword = !showCurrentPassword)}
              class="absolute right-3 top-2 text-zinc-500 hover:text-zinc-300 transition cursor-pointer"
              aria-label={showCurrentPassword ? 'Hide current password' : 'Show current password'}
              tabindex="-1"
            >
              {#if showCurrentPassword}
                <EyeOff class="w-4 h-4" />
              {:else}
                <Eye class="w-4 h-4" />
              {/if}
            </button>
          </div>
        </div>

        <div>
          <label for="modal-new-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">New Password</label>
          <div class="relative">
            <input
              id="modal-new-password"
              type={showNewPassword ? 'text' : 'password'}
              bind:value={newPassword}
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-3.5 pr-10 py-2 text-white text-xs focus:outline-hidden focus:border-zinc-400"
            />
            <button
              type="button"
              onclick={() => (showNewPassword = !showNewPassword)}
              class="absolute right-3 top-2 text-zinc-500 hover:text-zinc-300 transition cursor-pointer"
              aria-label={showNewPassword ? 'Hide new password' : 'Show new password'}
              tabindex="-1"
            >
              {#if showNewPassword}
                <EyeOff class="w-4 h-4" />
              {:else}
                <Eye class="w-4 h-4" />
              {/if}
            </button>
          </div>
        </div>

        <div>
          <label for="modal-confirm-password" class="block text-zinc-400 mb-1 uppercase text-[10px] tracking-wider">Confirm New Password</label>
          <div class="relative">
            <input
              id="modal-confirm-password"
              type={showConfirmPassword ? 'text' : 'password'}
              bind:value={confirmPassword}
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-3.5 pr-10 py-2 text-white text-xs focus:outline-hidden focus:border-zinc-400"
            />
            <button
              type="button"
              onclick={() => (showConfirmPassword = !showConfirmPassword)}
              class="absolute right-3 top-2 text-zinc-500 hover:text-zinc-300 transition cursor-pointer"
              aria-label={showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'}
              tabindex="-1"
            >
              {#if showConfirmPassword}
                <EyeOff class="w-4 h-4" />
              {:else}
                <Eye class="w-4 h-4" />
              {/if}
            </button>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-zinc-800">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 hover:text-white transition cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="px-5 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold transition cursor-pointer"
          >
            {isSubmitting ? 'Updating...' : 'Update Password'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

