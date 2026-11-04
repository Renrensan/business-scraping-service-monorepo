import { toast } from 'svelte-sonner';

export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
  request_id?: string;
}

export interface ApiError {
  message: string;
  status: number;
  requestId?: string;
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

function generateRequestId(): string {
  return 'req_' + Math.random().toString(36).substring(2, 10);
}

function sanitizeErrorMessage(rawMsg: string, status: number): string {
  if (!rawMsg) return status >= 500 ? 'Internal server error. Please try again.' : 'Request could not be completed.';
  const lower = rawMsg.toLowerCase();
  if (lower.includes('duplicate key') || lower.includes('already exists')) {
    return 'An item with this name or identifier already exists.';
  }
  if (lower.includes('no rows in result set') || lower.includes('not found')) {
    return 'The requested record could not be found.';
  }
  if (lower.includes('invalid credentials')) {
    return 'Invalid username or password.';
  }
  if (status === 401) {
    return 'Session expired. Please log in again.';
  }
  return rawMsg;
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const reqId = generateRequestId();
  const url = endpoint.startsWith('http') ? endpoint : `${BASE_URL}${endpoint}`;

  const headers = new Headers(options.headers || {});
  headers.set('X-Request-ID', reqId);
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include', // Send and receive HttpOnly session cookie
      signal: options.signal || AbortSignal.timeout(15000)
    });

    const responseReqId = response.headers.get('X-Request-ID') || reqId;

    if (response.status === 401) {
      if (
        typeof window !== 'undefined' &&
        !window.location.pathname.includes('/login') &&
        !endpoint.includes('/auth/me') &&
        !endpoint.includes('/auth/login')
      ) {
        window.location.replace('/login');
      }
      throw {
        message: 'Unauthorized',
        status: 401,
        requestId: responseReqId
      } as ApiError;
    }

    const json: ApiResponse<T> = await response.json().catch(() => ({
      success: response.ok,
      error: response.statusText
    }));

    if (!response.ok || json.success === false) {
      const rawErrorMsg = json.error || json.message || `Request failed (${response.status})`;
      const friendlyMsg = sanitizeErrorMessage(rawErrorMsg, response.status);

      if (!endpoint.includes('/auth/me')) {
        toast.error(friendlyMsg);
      }

      throw {
        message: friendlyMsg,
        status: response.status,
        requestId: responseReqId
      } as ApiError;
    }

    return (json.data !== undefined ? json.data : json) as T;
  } catch (err: any) {
    if (err?.status !== 401 && err?.message !== 'Unauthorized') {
      console.error(`API Error on ${url}:`, err);
    }
    throw err;
  }
}

export const api = {
  get: <T>(endpoint: string, params?: Record<string, any>, options?: RequestInit) => {
    let url = endpoint;
    if (params) {
      const sp = new URLSearchParams();
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') {
          sp.append(k, String(v));
        }
      });
      const qs = sp.toString();
      if (qs) url += (url.includes('?') ? '&' : '?') + qs;
    }
    return request<T>(url, { method: 'GET', ...options });
  },

  post: <T>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'POST',
      body: body instanceof FormData ? body : JSON.stringify(body)
    }),

  put: <T>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'PUT',
      body: body instanceof FormData ? body : JSON.stringify(body)
    }),

  patch: <T>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'PATCH',
      body: body instanceof FormData ? body : JSON.stringify(body)
    }),

  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: 'DELETE' }),

  downloadCSV: async (endpoint: string, filename: string, params?: Record<string, any>) => {
    let url = endpoint.startsWith('http') ? endpoint : `${BASE_URL}${endpoint}`;
    if (params) {
      const sp = new URLSearchParams();
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') {
          sp.append(k, String(v));
        }
      });
      const qs = sp.toString();
      if (qs) url += (url.includes('?') ? '&' : '?') + qs;
    }

    const response = await fetch(url, { credentials: 'include' });
    if (!response.ok) {
      toast.error('Failed to download CSV export');
      return;
    }

    const blob = await response.blob();
    const blobUrl = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(blobUrl);
    toast.success('CSV Download started');
  }
};
