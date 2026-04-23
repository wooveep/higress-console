<script setup lang="ts">
import { reactive, watch } from 'vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';

const props = defineProps<{
  open: boolean;
  title: string;
  loading?: boolean;
  saving?: boolean;
  consumerOptions: Array<{ label: string; value: string }>;
  departmentOptions: Array<{ label: string; value: string }>;
  userLevelOptions: Array<{ label: string; value: string }>;
  values: {
    consumers: string[];
    departments: string[];
    userLevels: string[];
  };
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: { consumers: string[]; departments: string[]; userLevels: string[] }];
}>();

const state = reactive({
  consumers: [] as string[],
  departments: [] as string[],
  userLevels: [] as string[],
});

watch(() => [props.open, props.values], () => {
  state.consumers = [...(props.values.consumers || [])];
  state.departments = [...(props.values.departments || [])];
  state.userLevels = [...(props.values.userLevels || [])];
}, { immediate: true, deep: true });
</script>

<template>
  <a-drawer
    :open="open"
    width="680"
    :title="title"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-alert
      type="info"
      show-icon
      style="margin-bottom: 16px"
      message="未配置授权记录时，已发布模型默认公开可见。配置任意授权项后，仅命中的 consumer、department 或 user level 可见。"
    />
    <a-form layout="vertical">
      <a-form-item label="允许访问的 Consumer">
        <a-select
          v-model:value="state.consumers"
          mode="multiple"
          :loading="loading"
          :options="consumerOptions"
          show-search
        />
      </a-form-item>
      <a-form-item label="允许访问的 Department">
        <a-select
          v-model:value="state.departments"
          mode="multiple"
          :loading="loading"
          :options="departmentOptions"
          show-search
        />
      </a-form-item>
      <a-form-item label="允许访问的用户等级">
        <a-select
          v-model:value="state.userLevels"
          mode="multiple"
          :loading="loading"
          :options="userLevelOptions"
        />
      </a-form-item>
    </a-form>
    <DrawerFooter :loading="saving" @cancel="emit('update:open', false)" @confirm="emit('submit', { ...state })" />
  </a-drawer>
</template>
