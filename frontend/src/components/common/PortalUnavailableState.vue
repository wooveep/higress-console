<script setup lang="ts">
import { PORTAL_UNAVAILABLE_MESSAGE } from '@/composables/usePortalAvailability';
</script>

<template>
  <div class="portal-unavailable-state">
    <a-alert
      type="warning"
      show-icon
      :message="PORTAL_UNAVAILABLE_MESSAGE"
    />
    <a-empty :description="PORTAL_UNAVAILABLE_MESSAGE" />
  </div>
</template>

<style scoped>
.portal-unavailable-state {
  display: grid;
  gap: 16px;
}
</style>
