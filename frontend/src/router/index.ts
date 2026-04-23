import { createRouter, createWebHistory } from 'vue-router';
import i18n from '@/i18n';
import { useAppStore } from '@/stores/app';
import { routes } from './routes';

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 };
  },
});

router.beforeEach(async (to) => {
  const appStore = useAppStore();
  await appStore.bootstrap();

  const title = i18n.global.t(to.meta.titleKey || 'index.title');
  document.title = `${title} · AIGateway Console`;

  if (!appStore.isInitialized && to.path !== '/init') {
    return `/init`;
  }

  if (appStore.isInitialized && to.path === '/init') {
    return '/login';
  }

  if (appStore.isInitialized && !appStore.isAuthenticated && to.meta.auth) {
    return `/login?redirect=${encodeURIComponent(to.fullPath)}`;
  }

  if (appStore.isAuthenticated && (to.path === '/login' || to.path === '/init')) {
    const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/dashboard';
    return redirect;
  }

  return true;
});

export default router;
