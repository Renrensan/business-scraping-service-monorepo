import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Prospects moved into the Scoring page (one page per profile, picked from a dropdown).
export const load: PageLoad = ({ params }) => {
	redirect(308, `/scoring/${params.id}`);
};
