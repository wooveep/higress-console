<script setup lang="ts">
import { computed } from 'vue';
import type { RouteAuthFormState } from './route-form';

const props = defineProps<{
  state: RouteAuthFormState;
  required?: boolean;
  departmentOptions: Array<{ label: string; value: string }>;
  levelOptions: Array<{ label: string; value: string }>;
}>();

const authEnabled = computed({
  get: () => props.required ? true : props.state.enabled,
  set: (value: boolean) => {
    props.state.enabled = value;
  },
});
</script>

<template>
  <a-card size="small" title="请求认证">
    <a-alert
      type="info"
      show-icon
      style="margin-bottom: 16px"
      message="授权命中规则：部门和用户等级按并集放行；部门选择默认包含所有子部门。"
    />

    <a-form-item v-if="!required" label="启用请求认证">
      <a-switch v-model:checked="authEnabled" />
    </a-form-item>

    <a-form-item v-else label="启用请求认证">
      <a-switch :checked="true" disabled />
    </a-form-item>

    <template v-if="authEnabled">
      <a-form-item label="允许访问的部门">
        <a-select
          v-model:value="state.allowedDepartments"
          mode="multiple"
          show-search
          :options="departmentOptions"
          placeholder="请选择部门"
        />
      </a-form-item>
      <a-form-item label="允许访问的用户等级">
        <a-select
          v-model:value="state.allowedConsumerLevels"
          mode="multiple"
          show-search
          :options="levelOptions"
          placeholder="请选择用户等级"
        />
      </a-form-item>
    </template>
  </a-card>
</template>
