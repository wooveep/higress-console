import { computed } from 'vue';
import { useAppStore } from '@/stores/app';

export const PORTAL_UNAVAILABLE_MESSAGE = 'Portal database is unavailable';

export function usePortalAvailability() {
  const appStore = useAppStore();

  const portalEnabled = computed(() => appStore.portalEnabled);
  const portalHealthy = computed(() => appStore.portalHealthy);
  const portalUnavailable = computed(() => !portalHealthy.value);

  return {
    portalEnabled,
    portalHealthy,
    portalUnavailable,
    portalUnavailableMessage: PORTAL_UNAVAILABLE_MESSAGE,
  };
}
