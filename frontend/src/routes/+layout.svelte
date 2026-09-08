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
    const isAuthed = await auth.checkAuth();
    if (!isAuthed && !isLoginPage && typeof window !== 'undefined') {
      window.location.replace('/login');
    } else if (isAuthed && isLoginPage && typeof window !== 'undefined') {
      window.location.replace('/');
    }
  });
</script>

<Toaster position="top-right" richColors theme="dark" />

{#if isLoginPage}
  {@render children()}
{:else if auth.isLoading}
  <!-- Dark minimal loader preventing any flash of protected admin UI -->
  <div class="min-h-screen bg-black flex items-center justify-center">
    <div class="w-5 h-5 border-2 border-zinc-800 border-t-white rounded-full animate-spin"></div>
  </div>
{:else if auth.isAuthenticated}
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

