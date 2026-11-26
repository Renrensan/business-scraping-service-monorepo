import { api } from '$lib/api/client';
import type { ScoringProfile } from '$lib/types';

/** The scoring profiles, shared by the sidebar and the Scoring pages so both stay in step. */
class ScoringProfilesStore {
  list = $state.raw<ScoringProfile[]>([]);
  loaded = $state(false);

  async load(): Promise<ScoringProfile[]> {
    this.list = (await api.get<ScoringProfile[]>('/api/v1/scoring/profiles')) || [];
    this.loaded = true;
    return this.list;
  }

  /** The profile Scoring opens by default: the first active one. */
  get first(): ScoringProfile | undefined {
    return this.list.find((p) => p.is_active) ?? this.list[0];
  }
}

export const scoringProfiles = new ScoringProfilesStore();
