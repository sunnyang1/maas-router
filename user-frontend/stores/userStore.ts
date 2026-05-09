import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { apiClient } from '@/lib/api/client';

export interface User {
  id: string;
  email: string;
  name: string;
  tier: 'free' | 'pro' | 'enterprise';
  status: 'active' | 'suspended';
  credBalance: number;
  createdAt?: string;
}

interface UserState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  // Actions
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  login: (email: string, password: string) => Promise<User>;
  register: (email: string, password: string, name: string) => Promise<User>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
  updateBalance: (balance: number) => void;
  checkAuth: () => Promise<boolean>;
}

export const useUserStore = create<UserState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,

      setUser: (user) => set({ 
        user, 
        isAuthenticated: !!user,
        isLoading: false 
      }),

      setLoading: (loading) => set({ isLoading: loading }),

      login: async (email: string, password: string) => {
        set({ isLoading: true });
        try {
          const response = await apiClient.login(email, password);
          const user: User = {
            id: response.user.id,
            email: response.user.email,
            name: response.user.name,
            tier: response.user.tier,
            status: response.user.status,
            credBalance: response.user.credBalance,
          };
          set({ 
            user, 
            isAuthenticated: true,
            isLoading: false 
          });
          return user;
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      register: async (email: string, password: string, name: string) => {
        set({ isLoading: true });
        try {
          const response = await apiClient.register(email, password, name);
          const user: User = {
            id: response.user.id,
            email: response.user.email,
            name: response.user.name,
            tier: response.user.tier,
            status: response.user.status,
            credBalance: response.user.credBalance,
          };
          set({ 
            user, 
            isAuthenticated: true,
            isLoading: false 
          });
          return user;
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: async () => {
        try {
          await apiClient.logout();
        } catch {
          // 即使 API 调用失败，也清除本地状态
        } finally {
          // Cookie 已由后端清除
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false
          });
        }
      },

      refreshUser: async () => {
        try {
          const profile = await apiClient.getProfile();
          const user: User = {
            id: profile.id,
            email: profile.email,
            name: profile.name,
            tier: profile.tier,
            status: profile.status,
            credBalance: profile.credBalance,
            createdAt: profile.createdAt,
          };
          set({ user });
        } catch (error) {
          // Token 可能已过期，清除状态
          set({ 
            user: null, 
            isAuthenticated: false 
          });
          throw error;
        }
      },

      updateBalance: (balance: number) =>
        set((state) => ({
          user: state.user ? { ...state.user, credBalance: balance } : null,
        })),

      checkAuth: async () => {
        // 服务端渲染时返回 false
        if (typeof window === 'undefined') {
          return false;
        }

        try {
          // 尝试获取用户信息 (Token 存储在 httpOnly cookie 中，会自动发送)
          const profile = await apiClient.getProfile();
          const user: User = {
            id: profile.id,
            email: profile.email,
            name: profile.name,
            tier: profile.tier,
            status: profile.status,
            credBalance: profile.credBalance,
            createdAt: profile.createdAt,
          };
          set({
            user,
            isAuthenticated: true,
            isLoading: false
          });
          return true;
        } catch (error) {
          // Token 无效或过期
          set({
            isAuthenticated: false,
            user: null,
            isLoading: false
          });
          return false;
        }
      },
    }),
    {
      name: 'user-storage',
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        // Do NOT persist user object with email, id, balance etc.
      }),
    }
  )
);

// Hook for checking authentication on app load
export const useAuthCheck = () => {
  const { checkAuth, isAuthenticated, isLoading } = useUserStore();
  
  return {
    checkAuth,
    isAuthenticated,
    isLoading,
  };
};
