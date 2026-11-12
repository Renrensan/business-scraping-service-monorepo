import type { LeadStatus } from '$lib/types';

export interface LeadStatusOption {
  value: LeadStatus;
  label: string;
  hint: string;
  /** Classes for the selected / badge state */
  active: string;
  /** Classes for the dot indicator */
  dot: string;
}

export const LEAD_STATUSES: LeadStatusOption[] = [
  {
    value: 'cold',
    label: 'Cold',
    hint: 'Contacted, no response yet',
    active: 'bg-sky-950/80 border-sky-700 text-sky-300',
    dot: 'bg-sky-400'
  },
  {
    value: 'warm',
    label: 'Warm',
    hint: 'They responded',
    active: 'bg-amber-950/80 border-amber-700 text-amber-300',
    dot: 'bg-amber-400'
  },
  {
    value: 'hot',
    label: 'Hot',
    hint: 'Looks interested in our product',
    active: 'bg-rose-950/80 border-rose-700 text-rose-300',
    dot: 'bg-rose-400'
  }
];

export function getLeadStatus(value?: LeadStatus | null): LeadStatusOption | undefined {
  return LEAD_STATUSES.find((s) => s.value === value);
}
