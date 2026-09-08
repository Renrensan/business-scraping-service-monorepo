<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { BusinessLead } from '$lib/types';
  import { CATEGORIES } from '$lib/constants';
  import Badge from '$lib/components/Badge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import BusinessDetailDrawer from '$lib/components/BusinessDetailDrawer.svelte';
  import ExportModal from '$lib/components/ExportModal.svelte';
  import NewJobModal from '$lib/components/NewJobModal.svelte';
  import { 
    Search, Filter, Download, Plus, Star, MapPin, 
    Phone, Globe, Mail, RefreshCw, Eye
  } from 'lucide-svelte';

  let leads = $state<BusinessLead[]>([]);
  let totalCount = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let pageSize = $state(20);
  let isLoading = $state(true);

  // Filter States
  let search = $state('');
  let selectedCategory = $state('');
  let city = $state('');
  let minRating = $state<string>('');
  let hasEmail = $state<boolean | undefined>(undefined);
  let hasPhone = $state<boolean | undefined>(undefined);

  // Drawers & Modals
  let selectedLead = $state<BusinessLead | null>(null);
  let showDrawer = $state(false);
  let showExportModal = $state(false);
  let showNewJobModal = $state(false);

  async function fetchBusinesses() {
    try {
      isLoading = true;
      const params: Record<string, any> = {
        page,
        page_size: pageSize,
        search: search.trim() || undefined,
        category: (selectedCategory && selectedCategory !== 'All Categories') ? selectedCategory : undefined,
        city: city.trim() || undefined,
        min_rating: minRating ? Number(minRating) : undefined,
        has_email: hasEmail !== undefined ? hasEmail : undefined,
        has_phone: hasPhone !== undefined ? hasPhone : undefined
      };

      const res = await api.get<{
        items: BusinessLead[];
        total_count: number;
        total_pages: number;
        page: number;
      }>('/api/v1/businesses', params);

      if (res) {
        leads = res.items || [];
        totalCount = res.total_count || 0;
        totalPages = res.total_pages || 1;
      }
    } catch (err) {
      console.error('Failed to query businesses', err);
    } finally {
      isLoading = false;
    }
  }

  function handleSearchInput() {
    page = 1;
    fetchBusinesses();
  }

  function openDetail(lead: BusinessLead) {
    selectedLead = lead;
    showDrawer = true;
  }

  onMount(() => {
    fetchBusinesses();
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-zinc-900">
    <div>
      <h1 class="text-2xl font-display font-light text-white tracking-[-0.04em]">Leads Database</h1>
      <p class="text-xs font-mono text-zinc-500 mt-0.5">High-speed verified business intelligence & enriched contacts ({totalCount.toLocaleString()} leads)</p>
    </div>
    <div class="flex items-center gap-2.5">
      <button
        onclick={() => (showExportModal = true)}
        class="px-3.5 py-2 rounded-xl border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-900 hover:text-white flex items-center gap-1.5 text-xs font-mono transition cursor-pointer"
      >
        <Download class="w-3.5 h-3.5" />
        <span>Export CSV</span>
      </button>
      <button
        onclick={() => (showNewJobModal = true)}
        class="px-4 py-2 rounded-xl bg-white hover:bg-zinc-200 text-black flex items-center gap-1.5 text-xs font-mono font-medium shadow-sm transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>New Extraction</span>
      </button>
    </div>
  </div>

  <!-- Filter Bar -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl p-4 shadow-sm space-y-3">
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
      <!-- Search Input -->
      <div class="relative">
        <Search class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
        <input
          type="text"
          bind:value={search}
          oninput={handleSearchInput}
          placeholder="Search name, phone, domain..."
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 font-mono text-xs"
        />
      </div>

      <!-- Category Filter -->
      <select
        bind:value={selectedCategory}
        onchange={handleSearchInput}
        class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 focus:outline-hidden focus:border-zinc-400 font-mono text-xs cursor-pointer"
      >
        {#each CATEGORIES as cat}
          <option value={cat}>{cat}</option>
        {/each}
      </select>

      <!-- City Input -->
      <div class="relative">
        <MapPin class="w-4 h-4 absolute left-3 top-2.5 text-zinc-500" />
        <input
          type="text"
          bind:value={city}
          oninput={handleSearchInput}
          placeholder="Filter by city..."
          class="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-white placeholder:text-zinc-600 focus:outline-hidden focus:border-zinc-400 font-mono text-xs"
        />
      </div>

      <!-- Rating Filter -->
      <select
        bind:value={minRating}
        onchange={handleSearchInput}
        class="w-full bg-zinc-900 border border-zinc-800 rounded-xl px-3 py-2 text-zinc-200 focus:outline-hidden focus:border-zinc-400 font-mono text-xs cursor-pointer"
      >
        <option value="">All Ratings</option>
        <option value="4.5">★ 4.5 and above</option>
        <option value="4.0">★ 4.0 and above</option>
        <option value="3.5">★ 3.5 and above</option>
      </select>
    </div>

    <!-- Contact Checkbox Filters -->
    <div class="flex flex-wrap items-center gap-4 pt-2 border-t border-zinc-900 text-xs font-mono text-zinc-400">
      <label class="flex items-center gap-1.5 cursor-pointer">
        <input
          type="checkbox"
          checked={hasEmail === true}
          onchange={(e) => {
            hasEmail = e.currentTarget.checked ? true : undefined;
            handleSearchInput();
          }}
          class="rounded accent-white bg-zinc-900"
        />
        <span>Has Verified Email</span>
      </label>

      <label class="flex items-center gap-1.5 cursor-pointer">
        <input
          type="checkbox"
          checked={hasPhone === true}
          onchange={(e) => {
            hasPhone = e.currentTarget.checked ? true : undefined;
            handleSearchInput();
          }}
          class="rounded accent-white bg-zinc-900"
        />
        <span>Has Primary Phone</span>
      </label>

      <div class="ml-auto text-[11px] font-mono text-zinc-500">
        Showing {leads.length} of {totalCount} records
      </div>
    </div>
  </div>

  <!-- Data Table -->
  <div class="bg-zinc-950 border border-zinc-800/90 rounded-2xl overflow-hidden shadow-sm">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs font-mono">
        <thead class="bg-black border-b border-zinc-800 text-[10px] uppercase tracking-wider text-zinc-500 font-medium">
          <tr>
            <th class="px-5 py-3.5">Business Name</th>
            <th class="px-4 py-3.5">Category</th>
            <th class="px-4 py-3.5">Location</th>
            <th class="px-4 py-3.5">Rating</th>
            <th class="px-4 py-3.5">Discovered Contacts</th>
            <th class="px-4 py-3.5 text-right">Details</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-900 text-zinc-300">
          {#if isLoading}
            <tr>
              <td colspan="6" class="px-5 py-16 text-center text-zinc-500">
                <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-zinc-400" />
                <span>Loading leads from PostgreSQL...</span>
              </td>
            </tr>
          {:else if leads.length === 0}
            <tr>
              <td colspan="6" class="px-5 py-16 text-center text-zinc-500">
                No business listings match your query. Try adjusting your search filters.
              </td>
            </tr>
          {:else}
            {#each leads as lead}
              <tr class="hover:bg-zinc-900/40 transition">
                <td class="px-5 py-3.5 font-medium text-white max-w-xs truncate">
                  <button 
                    onclick={() => openDetail(lead)}
                    class="hover:text-zinc-300 text-left truncate font-semibold block transition cursor-pointer"
                  >
                    {lead.name}
                  </button>
                  <div class="text-[11px] text-zinc-500 font-normal truncate">
                    {lead.street || lead.address}
                  </div>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <Badge variant="default">{lead.category}</Badge>
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap text-zinc-400">
                  {lead.city || 'N/A'}{lead.state ? `, ${lead.state}` : ''}
                </td>

                <td class="px-4 py-3.5 whitespace-nowrap">
                  <div class="flex items-center gap-1 text-white font-medium">
                    <Star class="w-3.5 h-3.5 fill-white text-white" />
                    <span>{lead.rating.toFixed(1)}</span>
                    <span class="text-[10px] text-zinc-500">({lead.review_count})</span>
                  </div>
                </td>

                <td class="px-4 py-3.5">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    {#if lead.emails && lead.emails.length > 0}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 text-emerald-400 border border-zinc-800 text-[10px] font-mono">
                        <Mail class="w-2.5 h-2.5" />
                        {lead.emails.length}
                      </span>
                    {/if}
                    {#if lead.primary_phone}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 text-zinc-300 border border-zinc-800 text-[10px] font-mono">
                        <Phone class="w-2.5 h-2.5" />
                      </span>
                    {/if}
                    {#if lead.website}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 text-zinc-400 border border-zinc-800 text-[10px]">
                        <Globe class="w-2.5 h-2.5" />
                      </span>
                    {/if}
                  </div>
                </td>

                <td class="px-4 py-3.5 text-right whitespace-nowrap">
                  <button
                    onclick={() => openDetail(lead)}
                    class="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 hover:text-white transition cursor-pointer"
                    title="View Profile Details"
                  >
                    <Eye class="w-3.5 h-3.5" />
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <Pagination
      page={page}
      totalPages={totalPages}
      onPageChange={(newPage) => {
        page = newPage;
        fetchBusinesses();
      }}
    />
  </div>
</div>

<BusinessDetailDrawer
  lead={selectedLead}
  isOpen={showDrawer}
  onClose={() => (showDrawer = false)}
/>

<ExportModal
  isOpen={showExportModal}
  onClose={() => (showExportModal = false)}
  filterParams={{
    search,
    category: selectedCategory,
    city,
    has_email: hasEmail,
    min_rating: minRating
  }}
/>

<NewJobModal
  isOpen={showNewJobModal}
  onClose={() => (showNewJobModal = false)}
  onJobCreated={fetchBusinesses}
/>