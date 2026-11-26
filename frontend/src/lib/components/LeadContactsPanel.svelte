<script lang="ts">
  import { api } from '$lib/api/client';
  import { toast } from 'svelte-sonner';
  import type { LeadContact, LeadContactInput } from '$lib/types';
  import { getWhatsAppUrl } from '$lib/utils/format';
  import { errorMessage } from '$lib/utils/errors';
  import { Users, Plus, Pencil, Trash2, MessageSquare, Mail, Phone } from 'lucide-svelte';

  let { leadId, onChanged }: { leadId: string; onChanged?: () => void } = $props();

  const ROLE_SUGGESTIONS = ['Owner', 'General Manager', 'Reservations', 'Front Office', 'Marketing', 'Sales'];
  const empty = (): LeadContactInput => ({ name: '', role: '', phone: '', email: '', note: '' });

  let contacts = $state.raw<LeadContact[]>([]);
  let editingId = $state<string | null>(null); // 'new' while adding
  let form = $state<LeadContactInput>(empty());
  let saving = $state(false);

  async function load(id: string) {
    try {
      const res = await api.get<LeadContact[]>(`/api/v1/businesses/${id}/contacts`);
      if (id === leadId) contacts = res || [];
    } catch {
      /* the drawer still shows the lead */
    }
  }

  $effect(() => {
    editingId = null;
    load(leadId);
  });

  function startAdd() {
    form = empty();
    editingId = 'new';
  }

  function startEdit(c: LeadContact) {
    form = { name: c.name, role: c.role, phone: c.phone, email: c.email, note: c.note };
    editingId = c.id;
  }

  async function save() {
    if (!form.name.trim()) {
      toast.error('Name is required');
      return;
    }
    try {
      saving = true;
      if (editingId === 'new') {
        await api.post(`/api/v1/businesses/${leadId}/contacts`, form);
        toast.success('Contact added');
      } else {
        await api.put(`/api/v1/businesses/${leadId}/contacts/${editingId}`, form);
        toast.success('Contact updated');
      }
      editingId = null;
      await load(leadId);
      onChanged?.();
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to save contact'));
    } finally {
      saving = false;
    }
  }

  async function remove(c: LeadContact) {
    if (!confirm(`Remove ${c.name} from this lead's contacts?`)) return;
    try {
      await api.delete(`/api/v1/businesses/${leadId}/contacts/${c.id}`);
      toast.success('Contact removed');
      await load(leadId);
      onChanged?.();
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to remove contact'));
    }
  }
</script>

<div class="mt-4 p-3 rounded-xl bg-black border border-zinc-800 space-y-2.5 text-xs">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Users class="w-4 h-4 text-zinc-400" />
      <span class="text-[10px] uppercase tracking-wider text-zinc-500 font-semibold">Contacts</span>
      {#if contacts.length}<span class="text-zinc-600">{contacts.length}</span>{/if}
    </div>
    {#if editingId === null}
      <button onclick={startAdd} class="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-zinc-300 hover:text-white cursor-pointer">
        <Plus class="w-3.5 h-3.5" /> Add
      </button>
    {/if}
  </div>

  {#each contacts as c, i (c.id)}
    {#if editingId === c.id}
      {@render editor()}
    {:else}
      <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-900 space-y-1">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <span class="text-white font-medium">{c.name}</span>
            {#if c.role}<span class="text-zinc-500"> · {c.role}</span>{/if}
            {#if i === 0 && contacts.length > 1}<span class="ml-1 text-[10px] text-zinc-600">main</span>{/if}
          </div>
          <div class="flex gap-1 shrink-0">
            <button onclick={() => startEdit(c)} disabled={editingId !== null} class="p-1 text-zinc-500 hover:text-white disabled:opacity-30 cursor-pointer" aria-label="Edit contact"><Pencil class="w-3.5 h-3.5" /></button>
            <button onclick={() => remove(c)} disabled={editingId !== null} class="p-1 text-zinc-500 hover:text-red-300 disabled:opacity-30 cursor-pointer" aria-label="Remove contact"><Trash2 class="w-3.5 h-3.5" /></button>
          </div>
        </div>
        {#if c.phone || c.email}
          <div class="flex flex-wrap gap-x-3 gap-y-1">
            {#if c.phone}
              {@const wa = getWhatsAppUrl(c.phone)}
              <span class="inline-flex items-center gap-1 text-zinc-300 font-mono"><Phone class="w-3 h-3 text-zinc-500" />{c.phone}</span>
              {#if wa}
                <a href={wa} target="_blank" rel="noreferrer" class="inline-flex items-center gap-1 text-emerald-400 hover:text-emerald-300"><MessageSquare class="w-3 h-3" />WhatsApp</a>
              {/if}
            {/if}
            {#if c.email}
              <a href="mailto:{c.email}" class="inline-flex items-center gap-1 text-zinc-300 hover:text-white"><Mail class="w-3 h-3 text-zinc-500" />{c.email}</a>
            {/if}
          </div>
        {/if}
        {#if c.note}<p class="text-zinc-400 whitespace-pre-line">{c.note}</p>{/if}
      </div>
    {/if}
  {:else}
    {#if editingId !== 'new'}
      <p class="text-zinc-600">No contact people yet. Add the owner, GM or whoever you spoke to.</p>
    {/if}
  {/each}

  {#if editingId === 'new'}
    {@render editor()}
  {/if}
</div>

{#snippet editor()}
  <form onsubmit={(e) => { e.preventDefault(); save(); }} class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-700 space-y-2">
    <div class="grid grid-cols-2 gap-2">
      <input bind:value={form.name} placeholder="Name *" required class="px-2 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-white placeholder-zinc-600" />
      <input bind:value={form.role} placeholder="Role (e.g. GM)" list="contact-roles" class="px-2 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-white placeholder-zinc-600" />
      <input bind:value={form.phone} placeholder="Phone / WhatsApp" inputmode="tel" class="px-2 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-white placeholder-zinc-600 font-mono" />
      <input bind:value={form.email} placeholder="Email" type="email" class="px-2 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-white placeholder-zinc-600" />
    </div>
    <datalist id="contact-roles">
      {#each ROLE_SUGGESTIONS as r (r)}<option value={r}></option>{/each}
    </datalist>
    <textarea bind:value={form.note} rows="2" placeholder="Note (e.g. prefers WhatsApp, decides for both properties)" class="w-full px-2 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-white placeholder-zinc-600"></textarea>
    <div class="flex justify-end gap-2">
      <button type="button" onclick={() => (editingId = null)} class="px-3 py-1 rounded-lg bg-zinc-900 text-zinc-400 hover:text-white cursor-pointer">Cancel</button>
      <button type="submit" disabled={saving} class="px-3 py-1 rounded-lg bg-white text-black font-semibold disabled:opacity-50 cursor-pointer">{saving ? 'Saving…' : 'Save'}</button>
    </div>
  </form>
{/snippet}
