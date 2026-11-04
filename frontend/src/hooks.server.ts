import type { Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const BACKEND_URL = env.INTERNAL_BACKEND_URL || env.BACKEND_URL || 'http://backend:8080';


export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname.startsWith('/api/') || event.url.pathname === '/health') {
		const targetUrl = new URL(event.url.pathname + event.url.search, BACKEND_URL);

		const headers = new Headers(event.request.headers);
		headers.delete('host');
		headers.delete('content-length');
		headers.set('X-Forwarded-Host', event.url.host);
		headers.set('X-Forwarded-Proto', event.url.protocol.replace(':', ''));

		try {
			const isBodyAllowed = event.request.method !== 'GET' && event.request.method !== 'HEAD';
			const bodyBuffer = isBodyAllowed ? await event.request.arrayBuffer() : undefined;

			console.log(`[Proxy] Forwarding ${event.request.method} ${event.url.pathname} -> ${targetUrl.toString()}`);

			const response = await fetch(targetUrl.toString(), {
				method: event.request.method,
				headers,
				body: bodyBuffer,
				redirect: 'manual'
			});

			return new Response(response.body, {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers
			});
		} catch (err) {
			console.error(`[Proxy Error] Failed to forward ${event.url.pathname} to ${targetUrl}:`, err);
			return new Response(JSON.stringify({ success: false, error: 'Backend gateway unavailable' }), {
				status: 502,
				headers: { 'Content-Type': 'application/json' }
			});
		}
	}

	return resolve(event);
};
