<script lang="ts">
  let {
    title,
    value,
    subtitle = '',
    trend = '',
    iconBg = 'bg-indigo-500/10 text-indigo-400',
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

<div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm hover:border-slate-700 transition flex flex-col justify-between">
  <div class="flex items-center justify-between">
    <span class="text-xs font-semibold uppercase tracking-wider text-slate-400">{title}</span>
    {#if icon}
      <div class="w-9 h-9 rounded-lg flex items-center justify-center {iconBg}">
        {@render icon()}
      </div>
    {/if}
  </div>
  <div class="mt-3">
    <div class="text-2xl font-bold text-white tracking-tight">{value}</div>
    {#if subtitle || trend}
      <div class="mt-1 flex items-center gap-1.5 text-xs text-slate-400">
        {#if trend}
          <span class="text-emerald-400 font-medium">{trend}</span>
        {/if}
        <span>{subtitle}</span>
      </div>
    {/if}
  </div>
</div>
