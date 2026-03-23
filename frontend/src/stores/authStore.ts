import { create } from 'zustand';
import { getCurrentUser, login } from '../api/authApi';
import type { User } from '../types/user';

interface AuthState {
  token: string | null;
  user: User | null;
  loading: boolean;
  loginWithPassword: (username: string, password: string) => Promise<boolean>;
  loadCurrentUser: () => Promise<boolean>;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: window.localStorage.getItem('auth_token'),
  user: null,
  loading: false,
  loginWithPassword: async (username: string, password: string) => {
    set({ loading: true });
    try {
      const response = await login({ username, password });
      window.localStorage.setItem('auth_token', response.token);
      set({ token: response.token, user: response.user, loading: false });
      return true;
    } catch (_error) {
      set({ loading: false });
      return false;
    }
  },
  loadCurrentUser: async () => {
    const token = window.localStorage.getItem('auth_token');
    if (!token) {
      set({ token: null, user: null });
      return false;
    }
    set({ loading: true, token });
    try {
      const user = await getCurrentUser();
      set({ user, loading: false });
      return true;
    } catch (_error) {
      window.localStorage.removeItem('auth_token');
      set({ token: null, user: null, loading: false });
      return false;
    }
  },
  logout: () => {
    window.localStorage.removeItem('auth_token');
    set({ token: null, user: null });
  },
}));
