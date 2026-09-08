<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { Toaster } from 'svelte-sonner';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  import ChangePasswordModal from '$lib/components/ChangePasswordModal.svelte';

  let { children } = $props();
  let showPasswordModal = $state(false);

  const isLoginPage = $derived(page.url.pathname === '/login');

  onMount(async () => {
    if (!isLoginPage) {
      const isAuthed = await auth.checkAuth();
      if (!isAuthed && typeof window !== 'undefined') {
        window.location.href = '/login';
      }
    }
  });
</script>

<Toaster position="top-right" richColors theme="dark" />

{#if isLoginPage}
  {@render children()}
{:else}
  <div class="flex min-h-screen bg-black text-white font-mono">
    <Sidebar />
    <div class="flex-1 flex flex-col min-w-0 bg-black">
      <Navbar onOpenPasswordModal={() => (showPasswordModal = true)} />
      <main class="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl w-full mx-auto">
        {@render children()}
      </main>
    </div>
  </div>

  <ChangePasswordModal
    isOpen={showPasswordModal}
    onClose={() => (showPasswordModal = false)}
  />
{/if}

