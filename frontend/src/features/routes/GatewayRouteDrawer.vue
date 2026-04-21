<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons-vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import type { Domain } from '@/interfaces/domain';
import type { Route } from '@/interfaces/route';
import type { Service } from '@/interfaces/service';
import { getGatewayDomains } from '@/services/domain';
import { getGatewayServices } from '@/services/service';
import { listOrgDepartmentsTree } from '@/services/organization';
import { showError } from '@/lib/feedback';
import RouteAuthSection from './RouteAuthSection.vue';
import {
  buildGatewayRoutePayload,
  buildServiceOptions,
  createKeyedPredicateFormItem,
  createRouteServiceFormItem,
  getDepartmentOptions,
  routeMatchTypeOptions,
  routeMethodOptions,
  toGatewayRouteFormState,
  userLevelOptions,
  type GatewayRouteFormState,
} from './route-form';

const props = defineProps<{
  open: boolean;
  route?: Route | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: Route, isEdit: boolean];
}>();

const loadingOptions = ref(false);
const services = ref<Service[]>([]);
const domains = ref<Domain[]>([]);
const departmentTree = ref<any[]>([]);
const formState = reactive<GatewayRouteFormState>(toGatewayRouteFormState());

const servicesByKey = computed(() => new Map(services.value.map((item) => {
  const key = item.port != null ? `${item.name}:${item.port}` : item.name;
  return [key, item];
})));
const domainOptions = computed(() => {
  const options = domains.value.map((item) => ({ label: item.name, value: item.name }));
  formState.domains.forEach((value) => {
    if (value && !options.some((item) => item.value === value)) {
      options.unshift({ label: `${value}（历史值）`, value });
    }
  });
  return options;
});
const serviceOptions = computed(() => buildServiceOptions(services.value, formState.services));
const departmentOptions = computed(() => getDepartmentOptions(departmentTree.value));

watch(() => [props.open, props.route], async ([open]) => {
  if (!open) {
    return;
  }
  loadingOptions.value = true;
  try {
    const [serviceList, domainList, departments] = await Promise.all([
      getGatewayServices().catch(() => []),
      getGatewayDomains().catch(() => []),
      listOrgDepartmentsTree().catch(() => []),
    ]);
    services.value = serviceList || [];
    domains.value = domainList || [];
    departmentTree.value = departments || [];
  } finally {
    loadingOptions.value = false;
  }
  Object.assign(formState, toGatewayRouteFormState(props.route || undefined, servicesByKey.value));
  if (formState.services.length === 0) {
    formState.services.push(createRouteServiceFormItem(undefined, servicesByKey.value));
  }
}, { immediate: true });

function close() {
  emit('update:open', false);
}

function addHeader() {
  formState.headers.push(createKeyedPredicateFormItem());
}

function addUrlParam() {
  formState.urlParams.push(createKeyedPredicateFormItem());
}

function addService() {
  formState.services.push(createRouteServiceFormItem(undefined, servicesByKey.value));
}

function removeAt<T>(items: T[], index: number) {
  items.splice(index, 1);
}

async function submit() {
  try {
    const payload = buildGatewayRoutePayload(formState, props.route || undefined, servicesByKey.value);
    emit('submit', payload, Boolean(props.route));
  } catch (error: any) {
    showError(String(error?.message || error || '保存失败'));
  }
}
</script>

<template>
  <a-drawer
    :open="open"
    width="900"
    :title="route ? '编辑路由' : '新增路由'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-form layout="vertical">
      <div class="route-drawer__grid">
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" :disabled="Boolean(route)" />
        </a-form-item>
        <a-form-item label="域名">
        <a-select
          v-model:value="formState.domains"
          mode="multiple"
          show-search
          :options="domainOptions"
          :loading="loadingOptions"
          placeholder="可留空表示内部路由"
        />
        </a-form-item>
      </div>

      <div class="route-drawer__grid route-drawer__grid--path">
        <a-form-item label="路径匹配方式">
          <a-select v-model:value="formState.pathMatchType" :options="routeMatchTypeOptions as any" />
        </a-form-item>
        <a-form-item label="路径匹配值" required>
          <a-input v-model:value="formState.pathMatchValue" />
        </a-form-item>
      </div>

      <a-form-item label="请求方法">
        <a-select
          v-model:value="formState.methods"
          mode="multiple"
          show-search
          :options="routeMethodOptions"
          placeholder="请选择允许的方法"
        />
      </a-form-item>

      <a-card size="small" title="Header 匹配" class="route-drawer__card">
        <div v-if="!formState.headers.length" class="route-drawer__empty">未配置 Header 匹配条件。</div>
        <div v-for="(item, index) in formState.headers" :key="item.id" class="route-drawer__row">
          <a-input v-model:value="item.key" placeholder="Header 名称" />
          <a-select v-model:value="item.matchType" :options="routeMatchTypeOptions as any" />
          <a-input v-model:value="item.matchValue" placeholder="匹配值" />
          <a-button danger @click="removeAt(formState.headers, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addHeader">
          <template #icon><PlusOutlined /></template>
          新增 Header 条件
        </a-button>
      </a-card>

      <a-card size="small" title="Query 匹配" class="route-drawer__card">
        <div v-if="!formState.urlParams.length" class="route-drawer__empty">未配置 Query 匹配条件。</div>
        <div v-for="(item, index) in formState.urlParams" :key="item.id" class="route-drawer__row">
          <a-input v-model:value="item.key" placeholder="Query 参数名" />
          <a-select v-model:value="item.matchType" :options="routeMatchTypeOptions as any" />
          <a-input v-model:value="item.matchValue" placeholder="匹配值" />
          <a-button danger @click="removeAt(formState.urlParams, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addUrlParam">
          <template #icon><PlusOutlined /></template>
          新增 Query 条件
        </a-button>
      </a-card>

      <a-card size="small" title="目标服务" class="route-drawer__card">
        <div v-for="(item, index) in formState.services" :key="item.id" class="route-drawer__row">
          <a-select
            v-model:value="item.serviceKey"
            show-search
            :loading="loadingOptions"
            :options="serviceOptions"
            placeholder="请选择服务"
          />
          <a-input-number v-model:value="item.weight" :min="0" :max="100" style="width: 140px" placeholder="权重" />
          <a-button danger @click="removeAt(formState.services, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addService">
          <template #icon><PlusOutlined /></template>
          新增目标服务
        </a-button>
      </a-card>

      <RouteAuthSection
        :state="formState.auth"
        :department-options="departmentOptions"
        :level-options="userLevelOptions"
      />
    </a-form>

    <DrawerFooter @cancel="close" @confirm="submit" />
  </a-drawer>
</template>

<style scoped>
.route-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.route-drawer__grid--path {
  align-items: end;
}

.route-drawer__card {
  margin-bottom: 16px;
}

.route-drawer__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px minmax(0, 1fr) auto;
  gap: 12px;
  margin-bottom: 12px;
}

.route-drawer__empty {
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 12px;
}

@media (max-width: 900px) {
  .route-drawer__grid {
    grid-template-columns: 1fr;
  }

  .route-drawer__row {
    grid-template-columns: 1fr;
  }
}
</style>
