import { defineStore } from 'pinia';
import { PORTAL_ENABLED, PORTAL_HEALTHY, SYSTEM_INITIALIZED } from '@/interfaces/config';
import type { UserInfo } from '@/interfaces/user';
import { getConfigs, getSystemInfo } from '@/services/system';
import { fetchUserInfo, logout } from '@/services/user';

interface AppState {
  ready: boolean;
  bootstrapping: boolean;
  currentUser: UserInfo | null;
  config: Record<string, any>;
  systemInfo: Record<string, any>;
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    ready: false,
    bootstrapping: false,
    currentUser: null,
    config: {},
    systemInfo: {},
  }),
  getters: {
    isInitialized: (state) => Boolean(state.config?.[SYSTEM_INITIALIZED]),
    isAuthenticated: (state) => Boolean(state.currentUser?.username),
    displayName: (state) => state.currentUser?.displayName || state.currentUser?.username || 'Admin',
    portalEnabled: (state) => Boolean(state.config?.[PORTAL_ENABLED]),
    portalHealthy: (state) => Boolean(state.config?.[PORTAL_HEALTHY]),
  },
  actions: {
    async bootstrap(force = false) {
      if (this.bootstrapping) {
        return;
      }
      if (this.ready && !force) {
        return;
      }

      this.bootstrapping = true;
      try {
        const [config, systemInfo, currentUser] = await Promise.all([
          getConfigs().catch(() => ({})),
          getSystemInfo().catch(() => ({})),
          fetchUserInfo().catch(() => null),
        ]);

        this.config = config || {};
        this.systemInfo = systemInfo || {};
        this.currentUser = currentUser && currentUser.username ? currentUser : null;
        this.ready = true;
      } finally {
        this.bootstrapping = false;
      }
    },
    setUser(user: UserInfo | null) {
      this.currentUser = user;
    },
    async signOut() {
      try {
        await logout();
      } finally {
        this.currentUser = null;
      }
    },
  },
});
