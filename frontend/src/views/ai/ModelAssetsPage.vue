<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import ListToolbar from '@/components/common/ListToolbar.vue';
import { showConfirm, showSuccess, showWarning } from '@/lib/feedback';
import {
  createModelAsset,
  createModelBinding,
  deleteModelAsset,
  deleteModelBinding,
  getModelAssets,
  getModelAssetOptions,
  getModelBindingPriceVersions,
  publishModelBinding,
  restoreModelBindingPriceVersion,
  unpublishModelBinding,
  updateModelAsset,
  updateModelBinding,
} from '@/services/model-asset';
import { getLlmProviders } from '@/services/llm-provider';
import { listAssetGrants, listOrgAccounts, listOrgDepartmentsTree, replaceAssetGrants } from '@/services/organization';
import { USER_LEVELS } from '@/utils/consumer-level';
import { formatProviderDisplayName } from '@/features/llm-provider/provider-display';
import {
  buildGrantAssignments,
  describeCapabilities,
  describePricing,
  flattenDepartmentOptions,
  MODEL_TYPE_LABELS,
  splitGrantAssignments,
  statusColorMap,
} from '@/features/model-assets/model-asset-form';

const ModelAssetDrawer = defineAsyncComponent(() => import('@/features/model-assets/ModelAssetDrawer.vue'));
const ModelBindingDrawer = defineAsyncComponent(() => import('@/features/model-assets/ModelBindingDrawer.vue'));
const ModelBindingHistoryDrawer = defineAsyncComponent(() => import('@/features/model-assets/ModelBindingHistoryDrawer.vue'));
const ModelBindingGrantDrawer = defineAsyncComponent(() => import('@/features/model-assets/ModelBindingGrantDrawer.vue'));
const { portalUnavailable } = usePortalAvailability();
const { t } = useI18n();

const loading = ref(false);
const search = ref('');
const assets = ref<any[]>([]);
const providers = ref<any[]>([]);
const assetOptions = ref<any>({
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
const accounts = ref<any[]>([]);
const departments = ref<any[]>([]);
const selectedAssetId = ref('');

const assetDrawerOpen = ref(false);
const bindingDrawerOpen = ref(false);
const historyDrawerOpen = ref(false);
const grantDrawerOpen = ref(false);

const editingAsset = ref<any>(null);
const editingBinding = ref<any>(null);
const historyBinding = ref<any>(null);
const grantBinding = ref<any>(null);

const priceVersions = ref<any[]>([]);
const priceVersionsLoading = ref(false);
const grantLoading = ref(false);
const grantSaving = ref(false);
const grantValues = ref({
  consumers: [] as string[],
  departments: [] as string[],
  userLevels: [] as string[],
});

const filteredAssets = computed(() => assets.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.assetId, item.canonicalName, item.displayName]
    .some((value) => String(value || '').toLowerCase().includes(keyword));
}));

const selectedAsset = computed(() =>
  assets.value.find((item) => item.assetId === selectedAssetId.value) || null);

const selectedBindings = computed(() => selectedAsset.value?.bindings || []);
const activePriceVersion = computed(() =>
  priceVersions.value.find((item) => item.active || item.status === 'active') || null);

const consumerOptions = computed(() =>
  accounts.value.map((item) => ({
    label: item.displayName ? `${item.displayName} (${item.consumerName})` : item.consumerName,
    value: item.consumerName,
  })));

const departmentOptions = computed(() => flattenDepartmentOptions(departments.value));
const userLevelOptions = computed(() => USER_LEVELS.map((item) => ({ label: `等级 ${item}`, value: item })));

async function loadAssets() {
  if (portalUnavailable.value) {
    assets.value = [];
    selectedAssetId.value = '';
    loading.value = false;
    return;
  }
  loading.value = true;
  try {
    const result = await getModelAssets().catch(() => []);
    assets.value = result || [];
    if (selectedAssetId.value && assets.value.some((item) => item.assetId === selectedAssetId.value)) {
      return;
    }
    selectedAssetId.value = assets.value[0]?.assetId || '';
  } finally {
    loading.value = false;
  }
}

async function loadSupportData() {
  if (portalUnavailable.value) {
    providers.value = [];
    assetOptions.value = {
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
    };
    accounts.value = [];
    departments.value = [];
    return;
  }
  const [nextProviders, nextAssetOptions, nextAccounts, nextDepartments] = await Promise.all([
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
    listOrgAccounts().catch(() => []),
    listOrgDepartmentsTree().catch(() => []),
  ]);
  providers.value = nextProviders;
  assetOptions.value = nextAssetOptions;
  accounts.value = nextAccounts;
  departments.value = nextDepartments;
}

async function loadPriceVersions(assetId?: string, bindingId?: string) {
  if (!assetId || !bindingId) {
    priceVersions.value = [];
    return;
  }
  priceVersionsLoading.value = true;
  try {
    priceVersions.value = await getModelBindingPriceVersions(assetId, bindingId).catch(() => []);
  } finally {
    priceVersionsLoading.value = false;
  }
}

function openAssetDrawer(record?: any) {
  editingAsset.value = record || null;
  assetDrawerOpen.value = true;
}

async function saveAsset(payload: any, isEdit: boolean) {
  if (isEdit && editingAsset.value) {
    await updateModelAsset(editingAsset.value.assetId, payload);
  } else {
    await createModelAsset(payload);
  }
  assetDrawerOpen.value = false;
  await loadAssets();
  showSuccess(isEdit ? '模型资产已更新' : '模型资产已创建');
}

async function openBindingDrawer(record?: any) {
  if (!selectedAsset.value) {
    showWarning('请先选择一个模型资产');
    return;
  }
  editingBinding.value = record || null;
  bindingDrawerOpen.value = true;
  if (record) {
    await loadPriceVersions(record.assetId || selectedAsset.value.assetId, record.bindingId);
  } else {
    priceVersions.value = [];
  }
}

function selectAsset(assetId: string) {
  selectedAssetId.value = assetId;
}

async function saveBinding(payload: any, isEdit: boolean) {
  const targetAssetId = payload.assetId || selectedAsset.value?.assetId;
  if (!targetAssetId) {
    return;
  }
  if (isEdit && editingBinding.value) {
    await updateModelBinding(targetAssetId, editingBinding.value.bindingId, payload);
  } else {
    await createModelBinding(targetAssetId, payload);
  }
  bindingDrawerOpen.value = false;
  selectedAssetId.value = targetAssetId;
  await Promise.all([loadAssets(), loadSupportData()]);
  showSuccess(isEdit ? '发布绑定已更新' : '发布绑定已创建');
}

async function toggleBinding(record: any) {
  const assetId = record.assetId || selectedAsset.value?.assetId;
  if (!assetId) {
    return;
  }
  if (record.status === 'published') {
    await unpublishModelBinding(assetId, record.bindingId);
    showSuccess('绑定已下架');
  } else {
    await publishModelBinding(assetId, record.bindingId);
    showSuccess('绑定已发布');
  }
  await loadAssets();
  if (editingBinding.value?.bindingId === record.bindingId) {
    await loadPriceVersions(assetId, record.bindingId);
  }
}

async function openHistoryDrawer(record: any) {
  historyBinding.value = record;
  historyDrawerOpen.value = true;
  await loadPriceVersions(record.assetId || selectedAsset.value?.assetId, record.bindingId);
}

async function restorePrice(versionId: number) {
  if (!historyBinding.value) {
    return;
  }
  showConfirm({
    title: `恢复价格版本 #${versionId}`,
    content: '该操作只会把历史价格回填到当前绑定草稿，不会自动发布。',
    okText: '恢复到草稿',
    cancelText: '取消',
    async onOk() {
      const assetId = historyBinding.value.assetId || selectedAsset.value?.assetId;
      if (!assetId) {
        return;
      }
      await restoreModelBindingPriceVersion(assetId, historyBinding.value.bindingId, versionId);
      await Promise.all([
        loadAssets(),
        loadPriceVersions(assetId, historyBinding.value.bindingId),
      ]);
      if (editingBinding.value?.bindingId === historyBinding.value.bindingId) {
        const nextAsset = assets.value.find((item) => item.assetId === assetId);
        const nextBinding = nextAsset?.bindings?.find((item: any) => item.bindingId === historyBinding.value.bindingId);
        editingBinding.value = nextBinding || editingBinding.value;
      }
      showSuccess('历史版本已恢复到草稿');
    },
  });
}

async function openGrantDrawer(record: any) {
  grantBinding.value = record;
  grantDrawerOpen.value = true;
  grantLoading.value = true;
  try {
    const grants = await listAssetGrants('model_binding', record.bindingId).catch(() => []);
    grantValues.value = splitGrantAssignments(grants);
  } finally {
    grantLoading.value = false;
  }
}

async function saveGrantAssignments(payload: { consumers: string[]; departments: string[]; userLevels: string[] }) {
  if (!grantBinding.value) {
    return;
  }
  grantSaving.value = true;
  try {
    const grants = buildGrantAssignments(
      grantBinding.value.bindingId,
      payload.consumers,
      payload.departments,
      payload.userLevels,
    );
    await replaceAssetGrants('model_binding', grantBinding.value.bindingId, grants);
    grantDrawerOpen.value = false;
    showSuccess('绑定授权已更新');
  } finally {
    grantSaving.value = false;
  }
}

function removeAsset(record: any) {
  showConfirm({
    title: `删除模型资产 ${record.assetId}`,
    content: '仅空资产可删除；若仍存在绑定，后端会阻止删除。',
    okText: '删除资产',
    okButtonProps: { danger: true },
    cancelText: '取消',
    async onOk() {
      await deleteModelAsset(record.assetId);
      if (selectedAssetId.value === record.assetId) {
        selectedAssetId.value = '';
      }
      await loadAssets();
      showSuccess('模型资产已删除');
    },
  });
}

function removeBinding(record: any) {
  const assetId = record.assetId || selectedAsset.value?.assetId;
  if (!assetId) {
    return;
  }
  showConfirm({
    title: `删除绑定 ${record.bindingId}`,
    content: '若绑定仍被发布、授权或 AI 路由引用，后端会阻止删除；价格历史会随绑定一起清理。',
    okText: '删除绑定',
    okButtonProps: { danger: true },
    cancelText: '取消',
    async onOk() {
      await deleteModelBinding(assetId, record.bindingId);
      await Promise.all([loadAssets(), loadSupportData()]);
      showSuccess('绑定已删除');
    },
  });
}

function providerDisplayName(providerName?: string) {
  return formatProviderDisplayName(String(providerName || ''), providers.value, t);
}

onMounted(async () => {
  await Promise.all([loadAssets(), loadSupportData()]);
});
</script>

<template>
  <div class="model-assets-page">
    <PageSection v-if="portalUnavailable" title="模型资产管理">
      <PortalUnavailableState />
    </PageSection>

    <template v-else>
      <PageSection title="模型资产管理">
        <ListToolbar
          v-model:search="search"
          search-placeholder="搜索资产 ID、标准名、显示名"
          create-text="创建模型资产"
          @refresh="loadAssets"
          @create="openAssetDrawer()"
        />
        <a-table
          :data-source="filteredAssets"
          :loading="loading"
          row-key="assetId"
          :scroll="{ x: 920 }"
          :row-class-name="(record) => record.assetId === selectedAssetId ? 'model-assets-page__selected-row' : ''"
          :custom-row="(record) => ({ onClick: () => selectAsset(record.assetId) })"
        >
          <a-table-column key="displayName" title="展示名">
            <template #default="{ record }">{{ record.displayName || record.canonicalName || record.assetId }}</template>
          </a-table-column>
          <a-table-column key="canonicalName" data-index="canonicalName" title="规范名" />
          <a-table-column key="modelType" title="模型类型" width="140">
            <template #default="{ record }">
              {{ MODEL_TYPE_LABELS[record.modelType] || record.modelType || '-' }}
            </template>
          </a-table-column>
          <a-table-column key="capabilities" title="能力摘要">
            <template #default="{ record }">{{ describeCapabilities(record) }}</template>
          </a-table-column>
          <a-table-column key="tags" title="标签">
            <template #default="{ record }">
              <a-space wrap size="small">
                <a-tag v-for="tag in record.tags || []" :key="tag">{{ tag }}</a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column key="bindings" title="绑定数" width="100">
            <template #default="{ record }">{{ record.bindings?.length || 0 }}</template>
          </a-table-column>
          <a-table-column key="actions" title="操作" width="180">
            <template #default="{ record }">
              <a-button type="link" size="small" @click.stop="openAssetDrawer(record)">编辑</a-button>
              <a-button type="link" size="small" danger @click.stop="removeAsset(record)">删除</a-button>
            </template>
          </a-table-column>
        </a-table>
      </PageSection>

      <PageSection :title="selectedAsset ? `发布绑定 · ${selectedAsset.displayName || selectedAsset.assetId}` : '发布绑定'">
        <template #actions>
          <a-space>
            <a-button :disabled="!selectedAsset" @click="selectedAsset && openAssetDrawer(selectedAsset)">编辑资产</a-button>
            <a-button type="primary" :disabled="!selectedAsset" @click="openBindingDrawer()">新建绑定</a-button>
          </a-space>
        </template>

        <a-empty v-if="!selectedAsset" description="还没有模型资产，先创建一个资产。" />

        <template v-else>
          <div class="model-assets-page__summary">
            <article class="model-assets-page__summary-card">
              <span>简介</span>
              <strong>{{ selectedAsset.intro || '-' }}</strong>
            </article>
            <article class="model-assets-page__summary-card">
              <span>模型类型</span>
              <strong>{{ MODEL_TYPE_LABELS[selectedAsset.modelType] || selectedAsset.modelType || '-' }}</strong>
            </article>
            <article class="model-assets-page__summary-card">
              <span>能力</span>
              <strong>{{ describeCapabilities(selectedAsset) }}</strong>
            </article>
          </div>

          <a-table :data-source="selectedBindings" row-key="bindingId" :scroll="{ x: 1160 }">
            <a-table-column key="modelId" data-index="modelId" title="模型 ID" />
            <a-table-column key="providerName" title="Provider">
              <template #default="{ record }">{{ providerDisplayName(record.providerName) }}</template>
            </a-table-column>
            <a-table-column key="targetModel" data-index="targetModel" title="目标模型" />
            <a-table-column key="status" title="状态" width="120">
              <template #default="{ record }">
                <a-tag :color="statusColorMap[record.status] || 'default'">{{ record.status || 'draft' }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column key="pricing" title="当前草稿价格">
              <template #default="{ record }">{{ describePricing(record.pricing, selectedAsset?.modelType) }}</template>
            </a-table-column>
            <a-table-column key="actions" title="操作" width="380" fixed="right">
              <template #default="{ record }">
                <a-space wrap size="small">
                  <a-button type="link" size="small" @click="openBindingDrawer(record)">编辑</a-button>
                  <a-button type="link" size="small" @click="toggleBinding(record)">{{ record.status === 'published' ? '下架' : '发布' }}</a-button>
                  <a-button type="link" size="small" @click="openGrantDrawer(record)">授权</a-button>
                  <a-button type="link" size="small" @click="openHistoryDrawer(record)">价格历史</a-button>
                  <a-button type="link" size="small" danger @click="removeBinding(record)">删除</a-button>
                </a-space>
              </template>
            </a-table-column>
          </a-table>
        </template>
      </PageSection>

      <PageSection title="授权上下文">
        <div class="model-assets-page__stats">
          <article class="model-assets-page__stat"><span>账号数</span><strong>{{ accounts.length }}</strong></article>
          <article class="model-assets-page__stat"><span>部门树节点</span><strong>{{ departments.length }}</strong></article>
        </div>
      </PageSection>
    </template>

    <ModelAssetDrawer
      v-model:open="assetDrawerOpen"
      :asset="editingAsset"
      :asset-options="assetOptions"
      @submit="saveAsset"
    />

    <ModelBindingDrawer
      v-model:open="bindingDrawerOpen"
      :binding="editingBinding"
      :assets="assets"
      :selected-asset-id="selectedAssetId"
      :providers="providers"
      :asset-options="assetOptions"
      :active-price-version="activePriceVersion"
      @submit="saveBinding"
      @open-history="editingBinding && openHistoryDrawer(editingBinding)"
    />

    <ModelBindingHistoryDrawer
      v-model:open="historyDrawerOpen"
      :title="historyBinding ? `价格历史 · ${historyBinding.modelId || historyBinding.bindingId}` : '价格历史'"
      :loading="priceVersionsLoading"
      :versions="priceVersions"
      @restore="restorePrice"
    />

    <ModelBindingGrantDrawer
      v-model:open="grantDrawerOpen"
      :title="grantBinding ? `模型可见性授权 · ${grantBinding.modelId || grantBinding.bindingId}` : '模型可见性授权'"
      :loading="grantLoading"
      :saving="grantSaving"
      :consumer-options="consumerOptions"
      :department-options="departmentOptions"
      :user-level-options="userLevelOptions"
      :values="grantValues"
      @submit="saveGrantAssignments"
    />
  </div>
</template>

<style scoped>
.model-assets-page {
  display: grid;
  gap: 18px;
}

.model-assets-page__summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}

.model-assets-page__summary-card,
.model-assets-page__stat {
  padding: 14px 16px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface-soft);
}

.model-assets-page__summary-card span,
.model-assets-page__stat span {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--portal-text-soft);
}

.model-assets-page__summary-card strong {
  display: block;
  color: var(--portal-text);
  line-height: 1.6;
}

.model-assets-page__stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

@media (max-width: 960px) {
  .model-assets-page__summary,
  .model-assets-page__stats {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
