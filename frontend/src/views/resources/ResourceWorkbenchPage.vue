<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import { resourceRegistry } from '@/resources/registry';
import { showConfirm, showError, showSuccess } from '@/lib/feedback';

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const search = ref('');
const loading = ref(false);
const editorOpen = ref(false);
const detailOpen = ref(false);
const editingRecord = ref<any | null>(null);
const detailRecord = ref<any | null>(null);
const editorState = reactive({ json: '' });
const rows = ref<any[]>([]);

const resource = computed(() => {
  const key = route.meta.resourceKey as string | undefined;
  return key ? resourceRegistry[key] : undefined;
});

const filteredRows = computed(() => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword || !resource.value) {
    return rows.value;
  }
  const fields = resource.value.searchFields || resource.value.fields || [];
  return rows.value.filter((row) => fields.some((field) => String(row?.[field] ?? '').toLowerCase().includes(keyword)));
});

const columns = computed(() => {
  const fields = resource.value?.fields || [];
  return [
    ...fields.map((field) => ({
      title: field,
      dataIndex: field,
      key: field,
      ellipsis: true,
    })),
    {
      title: t('misc.actions'),
      key: '__actions',
      fixed: 'right' as const,
      width: 280,
    },
  ] as any[];
});

async function loadRows() {
  if (!resource.value) {
    rows.value = [];
    return;
  }

  loading.value = true;
  try {
    const payload = await resource.value.list();
    rows.value = resource.value.normalizeList ? resource.value.normalizeList(payload) : payload;
  } finally {
    loading.value = false;
  }
}

function formatCellValue(value: any) {
  if (Array.isArray(value)) {
    return value.join(', ');
  }
  if (value && typeof value === 'object') {
    return JSON.stringify(value);
  }
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  return String(value);
}

function getColumnValue(record: any, dataIndex: any) {
  if (Array.isArray(dataIndex)) {
    return dataIndex.reduce((current, key) => current?.[key], record);
  }
  return record?.[dataIndex];
}

function openCreate() {
  editingRecord.value = null;
  editorState.json = JSON.stringify(resource.value?.createTemplate?.() ?? {}, null, 2);
  editorOpen.value = true;
}

function openEdit(record: any) {
  editingRecord.value = record;
  editorState.json = JSON.stringify(record, null, 2);
  editorOpen.value = true;
}

async function submitEditor() {
  if (!resource.value) {
    return;
  }
  let payload: any;
  try {
    payload = JSON.parse(editorState.json);
  } catch (error) {
    showError(`JSON parse error: ${(error as Error).message}`);
    return;
  }

  if (editingRecord.value && resource.value.update) {
    await resource.value.update(editingRecord.value, payload);
  } else if (!editingRecord.value && resource.value.create) {
    await resource.value.create(payload);
  }

  editorOpen.value = false;
  await loadRows();
  showSuccess(t('misc.save'));
}

async function removeRecord(record: any) {
  if (!resource.value?.remove) {
    return;
  }
  showConfirm({
    title: t('misc.delete'),
    content: `${resource.value.keyField}: ${record?.[resource.value.keyField] ?? ''}`,
    async onOk() {
      await resource.value?.remove?.(record);
      await loadRows();
      showSuccess(t('misc.delete'));
    },
  });
}

async function openDetail(record: any) {
  if (route.meta.resourceKey === 'mcp-list') {
    router.push({ path: `/mcp/detail/${record.name}` });
    return;
  }

  detailRecord.value = resource.value?.detail ? await resource.value.detail(record) : record;
  detailOpen.value = true;
}

async function runAction(record: any, action: any) {
  await action.run(record);
  await loadRows();
  showSuccess(t('misc.save'));
}

watch(() => route.meta.resourceKey, () => {
  search.value = '';
  loadRows();
}, { immediate: true });
</script>

<template>
  <div class="resource-page">
    <PageSection :title="t(route.meta.titleKey || 'index.title')">
      <template #actions>
        <a-input-search
          v-model:value="search"
          :placeholder="t('misc.search')"
          allow-clear
          style="width: 240px"
        />
        <a-button @click="loadRows">
          {{ t('misc.refresh') }}
        </a-button>
        <a-button
          v-if="resource && !resource.readonly && resource.create"
          type="primary"
          @click="openCreate"
        >
          {{ t('misc.create') }}
        </a-button>
      </template>

      <a-table
        :columns="columns"
        :data-source="filteredRows"
        :loading="loading"
        :row-key="(record: any) => record[resource?.keyField || 'id']"
        :scroll="{ x: 980 }"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === '__actions'">
            <div class="resource-page__actions">
              <a-button size="small" type="link" @click="openDetail(record)">
                {{ t('misc.information') }}
              </a-button>
              <a-button
                v-if="resource?.update"
                size="small"
                type="link"
                @click="openEdit(record)"
              >
                {{ t('misc.edit') }}
              </a-button>
              <a-button
                v-if="resource?.remove"
                size="small"
                type="link"
                danger
                @click="removeRecord(record)"
              >
                {{ t('misc.delete') }}
              </a-button>
              <a-button
                v-for="action in resource?.actions?.filter((item) => !item.visible || item.visible(record))"
                :key="action.key"
                size="small"
                type="link"
                @click="runAction(record, action)"
              >
                {{ t(action.labelKey) }}
              </a-button>
            </div>
          </template>
          <template v-else>
            <span :title="formatCellValue(getColumnValue(record, column.dataIndex))">
              {{ formatCellValue(getColumnValue(record, column.dataIndex)) }}
            </span>
          </template>
        </template>
      </a-table>
    </PageSection>

    <a-drawer
      v-model:open="editorOpen"
      width="680"
      :title="editingRecord ? t('misc.edit') : t('misc.create')"
      destroy-on-close
    >
      <a-textarea
        v-model:value="editorState.json"
        :rows="24"
        spellcheck="false"
      />
      <div class="resource-page__drawer-actions">
        <a-button @click="editorOpen = false">{{ t('misc.cancel') }}</a-button>
        <a-button type="primary" @click="submitEditor">{{ t('misc.save') }}</a-button>
      </div>
    </a-drawer>

    <a-drawer
      v-model:open="detailOpen"
      width="680"
      :title="t('misc.information')"
      destroy-on-close
    >
      <pre class="portal-pre">{{ JSON.stringify(detailRecord, null, 2) }}</pre>
    </a-drawer>
  </div>
</template>

<style scoped>
.resource-page {
  min-width: 0;
}

.resource-page__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
}

.resource-page__drawer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 18px;
}
</style>
