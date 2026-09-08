import { api } from '$lib/api/client';
import type { UserSession } from '$lib/types';

class AuthStore {
  user = $state<UserSession | null>(null);
  isLoading = $state<boolean>(true);
  isAuthenticated = $derived(this.user !== null);

  async checkAuth(): Promise<boolean> {
    try {
      this.isLoading = true;
      const data = await api.get<UserSession>('/api/v1/auth/me');
      this.user = data;
      return true;
    } catch {
      this.user = null;
      return false;
    } finally {
      this.isLoading = false;
    }
  }

  async login(usernameOrEmail: string, password: string): Promise<boolean> {
    try {
      this.isLoading = true;
      const res = await api.post<{ user: any }>('/api/v1/auth/login', {
        usernameOrEmail,
        password
      });
      if (res && res.user) {
        this.user = {
          adminId: res.user.id,
          username: res.user.username,
          email: res.user.email,
          createdAt: res.user.createdAt,
          lastActiveAt: new Date().toISOString()
        };
        return true;
      }
      return false;
    } finally {
      this.isLoading = false;
    }
  }

  async logout(): Promise<void> {
    try {
      await api.post('/api/v1/auth/logout');
    } catch (e) {
      console.warn('Logout failed:', e);
    } finally {
      this.user = null;
      if (typeof window !== 'undefined') {
        window.location.href = '/login';
      }
    }
  }
}

export const auth = new AuthStore();
