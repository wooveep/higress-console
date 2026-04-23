<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons-vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import type { AiRoute } from '@/interfaces/ai-route';
import type { Domain } from '@/interfaces/domain';
import type { LlmProvider } from '@/interfaces/llm-provider';
import type { ModelAssetOptions } from '@/interfaces/model-asset';
import { buildProviderDisplayOptions } from '@/features/llm-provider/provider-display';
import { getGatewayDomains } from '@/services/domain';
import { getLlmProviders } from '@/services/llm-provider';
import { getModelAssetOptions } from '@/services/model-asset';
import { listOrgDepartmentsTree } from '@/services/organization';
import { showError } from '@/lib/feedback';
import RouteAuthSection from './RouteAuthSection.vue';
import {
  buildAiRoutePayload,
  buildPublishedBindingCatalog,
  createAiUpstreamFormItem,
  createKeyedPredicateFormItem,
  createModelPredicateFormItem,
  fallbackResponseCodeOptions,
  getAiRouteLegacyIssue,
  getDepartmentOptions,
  getTargetModelOptions,
  providerHasBoundModels,
  routeMatchTypeOptions,
  userLevelOptions,
  toAiRouteFormState,
  type AiRouteFormState,
  type AiUpstreamFormItem,
} from './route-form';

const props = defineProps<{
  open: boolean;
  route?: AiRoute | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: AiRoute, isEdit: boolean];
}>();

const { t } = useI18n();
const loadingOptions = ref(false);
const providers = ref<LlmProvider[]>([]);
const assetOptions = ref<ModelAssetOptions>({
  capabilities: {
    modelTypes: [],
    inputModalities: [],
    outputModalities: [],
    featureFlags: [],
    modalities: [],
    features: [],
    requestKinds: [],
  },
  providerModels: [],
  publishedBindings: [],
});
const domains = ref<Domain[]>([]);
const departmentTree = ref<any[]>([]);
const formState = reactive<AiRouteFormState>(toAiRouteFormState());
const legacyIssue = computed(() => getAiRouteLegacyIssue(props.route || undefined));

const providerOptions = computed(() => buildProviderDisplayOptions(
  providers.value,
  t,
  [
    ...formState.upstreams.map((item) => item.provider),
    formState.fallback.provider,
  ],
));
const providerModelCatalog = computed(() => buildPublishedBindingCatalog(assetOptions.value));
const departmentOptions = computed(() => getDepartmentOptions(departmentTree.value));
const domainOptions = computed(() => {
  const options = domains.value.map((item) => ({ label: item.name, value: item.name }));
  formState.domains.forEach((value) => {
    if (value && !options.some((item) => item.value === value)) {
      options.unshift({ label: `${value}（历史值）`, value });
    }
  });
  return options;
});

watch(() => [props.open, props.route], async ([open]) => {
  if (!open) {
    return;
  }
  loadingOptions.value = true;
  try {
    const [providerList, options, domainList, departments] = await Promise.all([
      getLlmProviders().catch(() => []),
      getModelAssetOptions().catch(() => ({
        capabilities: {
          modelTypes: [],
          inputModalities: [],
          outputModalities: [],
          featureFlags: [],
          modalities: [],
          features: [],
          requestKinds: [],
        },
        providerModels: [],
        publishedBindings: [],
      })),
      getGatewayDomains().catch(() => []),
      listOrgDepartmentsTree().catch(() => []),
    ]);
    providers.value = providerList || [];
    assetOptions.value = options;
    domains.value = domainList || [];
    departmentTree.value = departments || [];
  } finally {
    loadingOptions.value = false;
  }
  Object.assign(formState, toAiRouteFormState(props.route || undefined));
  if (formState.modelPredicates.length === 0) {
    formState.modelPredicates.push(createModelPredicateFormItem());
  }
  if (formState.upstreams.length === 0) {
    formState.upstreams.push(createAiUpstreamFormItem());
  }
}, { immediate: true });

function close() {
  emit('update:open', false);
}

function removeAt<T>(items: T[], index: number) {
  items.splice(index, 1);
}

function addHeaderPredicate() {
  formState.headerPredicates.push(createKeyedPredicateFormItem());
}

function addUrlParamPredicate() {
  formState.urlParamPredicates.push(createKeyedPredicateFormItem());
}

function addModelPredicate() {
  formState.modelPredicates.push(createModelPredicateFormItem());
}

function addUpstream() {
  formState.upstreams.push(createAiUpstreamFormItem());
}

function getMappingOptions(providerName: string, currentValue?: string) {
  return getTargetModelOptions(providerName, providerModelCatalog.value, currentValue);
}

function handleProviderChange(item: AiUpstreamFormItem, providerName: string) {
  item.provider = providerName;
}

function validateEditableRoute() {
  if (legacyIssue.value) {
    throw new Error(legacyIssue.value);
  }
}

async function submit() {
  try {
    validateEditableRoute();
    const payload = buildAiRoutePayload(formState, props.route || undefined);
    emit('submit', payload, Boolean(props.route));
  } catch (error: any) {
    showError(String(error?.message || error || '保存失败'));
  }
}
</script>

<template>
  <a-drawer
    :open="open"
    width="980"
    :title="route ? '编辑 AI 路由' : '创建 AI 路由'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-form layout="vertical">
      <div class="ai-route-drawer__grid">
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
          <div class="ai-route-drawer__helper">
            <a href="/domain" target="_blank">创建域名</a>
          </div>
        </a-form-item>
      </div>

      <div class="ai-route-drawer__grid ai-route-drawer__grid--path">
        <a-form-item label="路径匹配方式">
          <a-select v-model:value="formState.pathMatchType" :options="routeMatchTypeOptions.filter((item) => item.value === 'PRE') as any" />
        </a-form-item>
        <a-form-item label="路径匹配值" required>
          <a-input v-model:value="formState.pathMatchValue" />
        </a-form-item>
        <a-form-item label="忽略大小写" class="ai-route-drawer__path-check">
          <a-checkbox v-model:checked="formState.pathIgnoreCase">忽略大小写</a-checkbox>
        </a-form-item>
      </div>

      <a-card size="small" title="Header 匹配" class="ai-route-drawer__card">
        <div v-if="!formState.headerPredicates.length" class="ai-route-drawer__empty">未配置 Header 匹配条件。</div>
        <div v-for="(item, index) in formState.headerPredicates" :key="item.id" class="ai-route-drawer__row">
          <a-input v-model:value="item.key" placeholder="Header 名称" />
          <a-select v-model:value="item.matchType" :options="routeMatchTypeOptions.filter((option) => option.value !== 'REGULAR') as any" />
          <a-input v-model:value="item.matchValue" placeholder="匹配值" />
          <a-button danger @click="removeAt(formState.headerPredicates, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addHeaderPredicate">
          <template #icon><PlusOutlined /></template>
          新增 Header 条件
        </a-button>
      </a-card>

      <a-card size="small" title="Query 匹配" class="ai-route-drawer__card">
        <div v-if="!formState.urlParamPredicates.length" class="ai-route-drawer__empty">未配置 Query 匹配条件。</div>
        <div v-for="(item, index) in formState.urlParamPredicates" :key="item.id" class="ai-route-drawer__row">
          <a-input v-model:value="item.key" placeholder="Query 参数名" />
          <a-select v-model:value="item.matchType" :options="routeMatchTypeOptions.filter((option) => option.value !== 'REGULAR') as any" />
          <a-input v-model:value="item.matchValue" placeholder="匹配值" />
          <a-button danger @click="removeAt(formState.urlParamPredicates, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addUrlParamPredicate">
          <template #icon><PlusOutlined /></template>
          新增 Query 条件
        </a-button>
      </a-card>

      <a-card size="small" title="模型匹配规则" class="ai-route-drawer__card">
        <div v-if="!formState.modelPredicates.length" class="ai-route-drawer__empty">未配置模型匹配规则。</div>
        <div v-for="(item, index) in formState.modelPredicates" :key="item.id" class="ai-route-drawer__model-predicate-row">
          <div class="ai-route-drawer__model-key">model</div>
          <a-select v-model:value="item.matchType" :options="routeMatchTypeOptions.filter((option) => option.value !== 'REGULAR') as any" />
          <a-input v-model:value="item.matchValue" placeholder="模型名称或前缀" />
          <a-button danger @click="removeAt(formState.modelPredicates, index)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addModelPredicate">
          <template #icon><PlusOutlined /></template>
          添加模型匹配规则
        </a-button>
      </a-card>

      <a-alert
        v-if="legacyIssue"
        type="warning"
        show-icon
        style="margin-bottom: 16px"
        message="检测到当前路由包含旧格式 fallback 配置"
        :description="legacyIssue"
      />

      <a-card size="small" title="目标 AI 服务" class="ai-route-drawer__card">
        <div class="ai-route-drawer__helper">
          <a href="/ai/provider" target="_blank">创建AI服务提供者</a>
        </div>

        <div v-for="(upstream, upstreamIndex) in formState.upstreams" :key="upstream.id" class="ai-route-drawer__upstream">
          <div class="ai-route-drawer__grid">
            <a-form-item label="服务名称" required>
              <a-select
                :value="upstream.provider"
                show-search
                :options="providerOptions"
                :loading="loadingOptions"
                @update:value="(value) => handleProviderChange(upstream, String(value || ''))"
              />
            </a-form-item>
            <a-form-item label="请求比例" required>
              <a-input-number v-model:value="upstream.weight" :min="0" :max="100" style="width: 100%" />
            </a-form-item>
            <a-form-item label="目标模型" class="ai-route-drawer__grid-span-2">
              <a-auto-complete
                v-model:value="upstream.modelMapping"
                :options="getMappingOptions(upstream.provider, upstream.modelMapping)"
                :disabled="!upstream.provider"
                placeholder="直接填写默认目标模型，或使用 源模型=目标模型;源模型2=目标模型2"
              />
              <div class="ai-route-drawer__field-note">未填写时表示不做模型映射；支持单值默认映射和 `key=value;...` 高级写法。</div>
            </a-form-item>
          </div>

          <a-alert
            v-if="upstream.provider && !providerHasBoundModels(upstream.provider, providerModelCatalog)"
            type="info"
            show-icon
            style="margin-bottom: 12px"
            :message="`Provider ${upstream.provider} 当前没有已发布绑定，可手动填写目标模型或映射规则。`"
          />

          <div class="ai-route-drawer__upstream-actions">
            <a-button danger @click="removeAt(formState.upstreams, upstreamIndex)">删除上游</a-button>
          </div>
        </div>

        <a-button type="dashed" block @click="addUpstream">
          <template #icon><PlusOutlined /></template>
          新增目标 AI 服务
        </a-button>
      </a-card>

      <RouteAuthSection
        required
        :state="formState.auth"
        :department-options="departmentOptions"
        :level-options="userLevelOptions"
      />

      <a-card size="small" title="Fallback 配置" class="ai-route-drawer__card">
        <a-form-item label="启用 Fallback">
          <a-switch v-model:checked="formState.fallback.enabled" />
        </a-form-item>

        <template v-if="formState.fallback.enabled">
          <a-form-item label="触发状态码" required>
            <a-select
              v-model:value="formState.fallback.responseCodes"
              mode="multiple"
              :options="fallbackResponseCodeOptions"
            />
          </a-form-item>

          <div class="ai-route-drawer__grid">
            <a-form-item label="Fallback 服务名称" required>
              <a-select
                v-model:value="formState.fallback.provider"
                show-search
                :options="providerOptions"
                :loading="loadingOptions"
              />
            </a-form-item>
            <a-form-item label="Fallback 目标模型" required>
              <a-auto-complete
                v-model:value="formState.fallback.modelMapping"
                :options="getMappingOptions(formState.fallback.provider, formState.fallback.modelMapping)"
                :disabled="!formState.fallback.provider"
                placeholder="默认目标模型，或 源模型=目标模型"
              />
              <div class="ai-route-drawer__field-note">按 Higress 原方案，下发为单 fallback 服务顺序降级。</div>
            </a-form-item>
          </div>

          <a-alert
            v-if="formState.fallback.provider && !providerHasBoundModels(formState.fallback.provider, providerModelCatalog)"
            type="info"
            show-icon
            :message="`Provider ${formState.fallback.provider} 当前没有已发布绑定，可手动填写 fallback 目标模型。`"
          />
        </template>
      </a-card>
    </a-form>

    <DrawerFooter @cancel="close" @confirm="submit" />
  </a-drawer>
</template>

<style scoped>
.ai-route-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.ai-route-drawer__grid--path {
  align-items: end;
}

.ai-route-drawer__grid-span-2 {
  grid-column: span 2;
}

.ai-route-drawer__path-check {
  grid-column: span 2;
}

.ai-route-drawer__card {
  margin-bottom: 16px;
}

.ai-route-drawer__row,
.ai-route-drawer__model-predicate-row {
  display: grid;
  gap: 12px;
  margin-bottom: 12px;
}

.ai-route-drawer__row {
  grid-template-columns: minmax(0, 1fr) 160px minmax(0, 1fr) auto;
}

.ai-route-drawer__model-predicate-row {
  grid-template-columns: 120px 180px minmax(0, 1fr) auto;
  align-items: center;
}

.ai-route-drawer__model-key {
  color: rgba(0, 0, 0, 0.65);
}

.ai-route-drawer__upstream {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}

.ai-route-drawer__upstream-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.ai-route-drawer__empty {
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 12px;
}

.ai-route-drawer__helper {
  margin-bottom: 12px;
}

.ai-route-drawer__field-note {
  margin-top: 6px;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 960px) {
  .ai-route-drawer__grid,
  .ai-route-drawer__row,
  .ai-route-drawer__model-predicate-row {
    grid-template-columns: 1fr;
  }

  .ai-route-drawer__grid-span-2,
  .ai-route-drawer__path-check {
    grid-column: auto;
  }

  .ai-route-drawer__upstream-actions {
    flex-direction: column;
  }
}
</style>
