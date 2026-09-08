<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { toast } from 'svelte-sonner';
  import { ArrowRight, Lock, User, ShieldCheck } from 'lucide-svelte';

  let usernameOrEmail = $state('');
  let password = $state('');
  let isLoading = $state(false);

  async function handleLogin() {
    if (!usernameOrEmail.trim() || !password) {
      toast.error('Please enter username/email and password');
      return;
    }

    try {
      isLoading = true;
      const ok = await auth.login(usernameOrEmail.trim(), password);
      if (ok) {
        toast.success('Authenticated session established');
        window.location.href = '/';
      }
    } catch (err: any) {
      toast.error(err.message || 'Authentication failed');
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="min-h-screen bg-black flex flex-col justify-center items-center p-6 relative selection:bg-white selection:text-black">
  <div class="w-full max-w-sm">
    <!-- Brand Lockup -->
    <div class="text-left mb-8 space-y-1">
      <div class="flex items-center gap-3">
        <h1 class="text-4xl font-display font-light text-white tracking-[-0.05em]">
          MŌMOS
        </h1>
        <span class="text-[10px] font-mono uppercase px-2 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-zinc-300">
          PRO
        </span>
      </div>
      <p class="text-xs font-mono text-zinc-500 uppercase tracking-wider">
        Lead Intelligence & Data Extraction Engine
      </p>
    </div>

    <!-- Login Container -->
    <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-7 shadow-2xl">
      <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 text-xs font-mono">
        <div>
          <label class="block text-zinc-400 mb-1.5 uppercase text-[11px] tracking-wider">Username or Email</label>
          <div class="relative">
            <User class="w-4 h-4 absolute left-3.5 top-2.5 text-zinc-500" />
            <input
              type="text"
              bind:value={usernameOrEmail}
              placeholder="admin"
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-10 pr-3.5 py-2.5 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs transition"
            />
          </div>
        </div>

        <div>
          <label class="block text-zinc-400 mb-1.5 uppercase text-[11px] tracking-wider">Password</label>
          <div class="relative">
            <Lock class="w-4 h-4 absolute left-3.5 top-2.5 text-zinc-500" />
            <input
              type="password"
              bind:value={password}
              placeholder="••••••••••••"
              required
              class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-10 pr-3.5 py-2.5 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 text-xs transition"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={isLoading}
          class="w-full mt-3 py-2.5 px-4 rounded-xl bg-white hover:bg-zinc-200 text-black font-semibold flex items-center justify-center gap-2 shadow-sm disabled:opacity-50 transition cursor-pointer"
        >
          <span>{isLoading ? 'Verifying...' : 'Sign In'}</span>
          <ArrowRight class="w-4 h-4" />
        </button>
      </form>

      <div class="mt-6 pt-5 border-t border-zinc-800/80 flex items-center justify-between text-[10px] font-mono text-zinc-500">
        <span class="flex items-center gap-1.5">
          <ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
          Redis Session Active
        </span>
        <span>Port 8080 API</span>
      </div>
    </div>
  </div>
</div>

