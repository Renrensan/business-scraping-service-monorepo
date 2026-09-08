<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { toast } from 'svelte-sonner';
  import { Layers, ShieldCheck, ArrowRight, Lock, User } from 'lucide-svelte';

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
        toast.success('Logged in successfully!');
        window.location.href = '/';
      }
    } catch (err: any) {
      toast.error(err.message || 'Login failed');
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="min-h-screen bg-slate-950 flex flex-col justify-center items-center p-4 relative overflow-hidden">
  <!-- Background Glows -->
  <div class="absolute -top-40 -left-40 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none"></div>
  <div class="absolute -bottom-40 -right-40 w-96 h-96 bg-violet-500/10 rounded-full blur-3xl pointer-events-none"></div>

  <div class="w-full max-w-md">
    <!-- Brand -->
    <div class="text-center mb-8">
      <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-gradient-to-tr from-indigo-500 to-violet-600 text-white shadow-xl shadow-indigo-500/20 mb-4">
        <Layers class="w-7 h-7" />
      </div>
      <h1 class="text-2xl font-bold text-white tracking-tight">BizScrape <span class="text-indigo-400">PRO</span></h1>
      <p class="text-xs text-slate-400 mt-1.5">High-Speed Google Maps Extraction & Lead Enrichment</p>
    </div>

    <!-- Login Card -->
    <div class="bg-slate-900 border border-slate-800/80 rounded-2xl p-7 shadow-2xl backdrop-blur-md">
      <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 text-xs">
        <div>
          <label class="block font-medium text-slate-300 mb-1.5">Username or Email</label>
          <div class="relative">
            <User class="w-4 h-4 absolute left-3.5 top-2.5 text-slate-400" />
            <input
              type="text"
              bind:value={usernameOrEmail}
              placeholder="admin or admin@bizscrape.local"
              required
              class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-10 pr-3.5 py-2.5 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs transition"
            />
          </div>
        </div>

        <div>
          <label class="block font-medium text-slate-300 mb-1.5">Password</label>
          <div class="relative">
            <Lock class="w-4 h-4 absolute left-3.5 top-2.5 text-slate-400" />
            <input
              type="password"
              bind:value={password}
              placeholder="••••••••••••"
              required
              class="w-full bg-slate-800/80 border border-slate-700 rounded-xl pl-10 pr-3.5 py-2.5 text-white placeholder:text-slate-400 focus:outline-hidden focus:border-indigo-500 text-xs transition"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={isLoading}
          class="w-full mt-2 py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/20 disabled:opacity-50 transition cursor-pointer"
        >
          <span>{isLoading ? 'Verifying Session...' : 'Sign In to Dashboard'}</span>
          <ArrowRight class="w-4 h-4" />
        </button>
      </form>

      <div class="mt-6 pt-5 border-t border-slate-800/80 flex items-center justify-between text-[11px] text-slate-400">
        <span class="flex items-center gap-1.5">
          <ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
          Redis Session Protected
        </span>
        <span class="font-mono text-slate-400">Port 8080 API</span>
      </div>
    </div>
  </div>
</div>
