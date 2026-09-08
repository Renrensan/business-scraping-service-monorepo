<script lang="ts">
  let {
    title,
    value,
    subtitle = '',
    trend = '',
    iconBg = 'bg-zinc-900 text-white',
    icon
  }: {
    title: string;
    value: string | number;
    subtitle?: string;
    trend?: string;
    iconBg?: string;
    icon?: any;
  } = $props();
</script>

<div class="bg-zinc-950 border border-zinc-800/90 rounded-xl p-5 hover:border-zinc-700 transition flex flex-col justify-between">
  <div class="flex items-center justify-between">
    <span class="text-[11px] font-mono uppercase tracking-wider text-zinc-500">{title}</span>
    {#if icon}
      <div class="w-8 h-8 rounded-lg flex items-center justify-center border border-zinc-800 {iconBg}">
        {@render icon()}
      </div>
    {/if}
  </div>
  <div class="mt-4">
    <div class="text-3xl font-display font-light text-white tracking-tight">{value}</div>
    {#if subtitle || trend}
      <div class="mt-1.5 flex items-center gap-1.5 text-xs font-mono text-zinc-400">
        {#if trend}
          <span class="text-emerald-400 font-medium">{trend}</span>
          <span>•</span>
        {/if}
        <span>{subtitle}</span>
      </div>
    {/if}
  </div>
</div>

