<script lang="ts">
  let {
    variant = 'default',
    children
  }: {
    variant?: 'default' | 'success' | 'warning' | 'error' | 'info' | 'purple';
    children: any;
  } = $props();

  const variantClasses = {
    default: 'bg-slate-800 text-slate-300 border-slate-700',
    success: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
    warning: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    error: 'bg-rose-500/10 text-rose-400 border-rose-500/20',
    info: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    purple: 'bg-purple-500/10 text-purple-400 border-purple-500/20'
  };
</script>

<span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border {variantClasses[variant]}">
  {@render children()}
</span>
