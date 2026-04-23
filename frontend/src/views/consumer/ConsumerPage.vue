<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import { showCopyValueModal, showError, showSuccess } from '@/lib/feedback';
import { formatDateTimeDisplay } from '@/utils/time';
import { useI18n } from 'vue-i18n';
import type { ConsumerDetail, InviteCodeRecord } from '@/interfaces/consumer';
import type { OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import {
  createOrgAccount,
  createOrgDepartment,
  listOrgAccounts,
  listOrgDepartmentsTree,
  updateOrgAccount,
  updateOrgAccountStatus,
} from '@/services/organization';
import {
  createInviteCode,
  disableInviteCode,
  enableInviteCode,
  getConsumerDetail,
  listInviteCodes,
  resetConsumerPassword,
} from '@/services/consumer';

const { t } = useI18n();
const { portalUnavailable } = usePortalAvailability();

const loading = ref(false);
const inviteLoading = ref(false);
const accountSaving = ref(false);
const departmentSaving = ref(false);

const departments = ref<OrgDepartmentNode[]>([]);
const accounts = ref<OrgAccountRecord[]>([]);
const inviteCodes = ref<InviteCodeRecord[]>([]);
const selectedDepartmentId = ref<string>();
const accountModalOpen = ref(false);
const departmentModalOpen = ref(false);
const detailOpen = ref(false);
const detailLoading = ref(false);
const editingAccount = ref<OrgAccountRecord | null>(null);
const consumerDetail = ref<ConsumerDetail | null>(null);

const accountFormRef = ref();
const departmentFormRef = ref();

const accountForm = reactive({
  consumerName: '',
  displayName: '',
  email: '',
  userLevel: 'normal',
  departmentId: '',
  password: '',
});

const departmentForm = reactive({
  name: '',
  parentDepartmentId: '',
});

const filteredAccounts = computed(() => {
  if (!selectedDepartmentId.value) {
    return accounts.value;
  }
  return accounts.value.filter((item) => item.departmentId === selectedDepartmentId.value);
});

const departmentTreeData = computed(() => toDepartmentTree(departments.value));
const departmentOptions = computed(() => flattenDepartmentOptions(departments.value));

function toDepartmentTree(nodes: OrgDepartmentNode[]): any[] {
  return (nodes || []).map((node) => ({
    key: node.departmentId,
    title: `${node.name} (${node.memberCount || 0})`,
    children: toDepartmentTree(node.children || []),
  }));
}

function flattenDepartmentOptions(nodes: OrgDepartmentNode[], level = 0): Array<{ label: string; value: string }> {
  return (nodes || []).flatMap((node) => {
    const prefix = level > 0 ? `${'  '.repeat(level)}- ` : '';
    return [
      { label: `${prefix}${node.name}`, value: node.departmentId },
      ...flattenDepartmentOptions(node.children || [], level + 1),
    ];
  });
}

function handleDepartmentSelect(keys: string[]) {
  selectedDepartmentId.value = keys[0];
}

function resolveAccountStatusText(status?: string) {
  const normalized = String(status || '').toLowerCase();
  if (normalized === 'active') {
    return t('misc.enabled');
  }
  if (normalized === 'disabled') {
    return t('misc.disabled');
  }
  if (normalized === 'pending') {
    return t('consumer.portalStatus.pending');
  }
  return status || '-';
}

function resolveInviteStatusText(status?: string) {
  const normalized = String(status || '').toLowerCase();
  if (normalized === 'active') {
    return t('misc.enabled');
  }
  if (normalized === 'disabled') {
    return t('misc.disabled');
  }
  if (normalized === 'used') {
    return t('consumer.inviteCode.status.used');
  }
  return status || '-';
}

function resolveUserLevelText(level?: string) {
  const normalized = String(level || '').toLowerCase();
  const key = `consumer.userLevel.${normalized}`;
  const translated = t(key);
  return translated === key ? level || '-' : translated;
}

async function load() {
  if (portalUnavailable.value) {
    departments.value = [];
    accounts.value = [];
    inviteCodes.value = [];
    loading.value = false;
    inviteLoading.value = false;
    return;
  }
  loading.value = true;
  inviteLoading.value = true;
  try {
    const [departmentTree, accountList, inviteList] = await Promise.all([
      listOrgDepartmentsTree().catch(() => []),
      listOrgAccounts().catch(() => []),
      listInviteCodes({ pageNum: 1, pageSize: 20 }).catch(() => [] as InviteCodeRecord[]),
    ]);
    departments.value = departmentTree || [];
    accounts.value = accountList || [];
    inviteCodes.value = Array.isArray(inviteList) ? inviteList : [];
  } finally {
    loading.value = false;
    inviteLoading.value = false;
  }
}

function openCreateAccount() {
  editingAccount.value = null;
  Object.assign(accountForm, {
    consumerName: '',
    displayName: '',
    email: '',
    userLevel: 'normal',
    departmentId: selectedDepartmentId.value || '',
    password: '',
  });
  accountModalOpen.value = true;
}

function openCreateDepartment() {
  Object.assign(departmentForm, {
    name: '',
    parentDepartmentId: selectedDepartmentId.value || '',
  });
  departmentModalOpen.value = true;
}

function openEditAccount(record: OrgAccountRecord) {
  editingAccount.value = record;
  Object.assign(accountForm, {
    consumerName: record.consumerName,
    displayName: record.displayName || '',
    email: record.email || '',
    userLevel: record.userLevel || 'normal',
    departmentId: record.departmentId || '',
    password: '',
  });
  accountModalOpen.value = true;
}

async function saveAccount() {
  await accountFormRef.value?.validate();

  accountSaving.value = true;
  try {
    const payload = {
      ...accountForm,
      consumerName: accountForm.consumerName.trim(),
      displayName: accountForm.displayName.trim(),
      email: accountForm.email.trim() || undefined,
      departmentId: accountForm.departmentId || undefined,
      password: accountForm.password || undefined,
    };
    if (editingAccount.value) {
      await updateOrgAccount(editingAccount.value.consumerName, payload);
    } else {
      await createOrgAccount(payload);
    }
    accountModalOpen.value = false;
    await load();
    showSuccess(t('misc.saveSuccess'));
  } finally {
    accountSaving.value = false;
  }
}

async function saveDepartment() {
  await departmentFormRef.value?.validate();

  departmentSaving.value = true;
  try {
    await createOrgDepartment({
      name: departmentForm.name.trim(),
      parentDepartmentId: departmentForm.parentDepartmentId || undefined,
    });
    departmentModalOpen.value = false;
    Object.assign(departmentForm, {
      name: '',
      parentDepartmentId: '',
    });
    await load();
    showSuccess(t('consumer.departmentCreateSuccess'));
  } finally {
    departmentSaving.value = false;
  }
}

async function handleResetPassword(record: OrgAccountRecord) {
  try {
    const result = await resetConsumerPassword(record.consumerName);
    showSuccess(t('consumer.resetPasswordSuccess'));
    showCopyValueModal({
      title: t('consumer.resetPasswordTitle'),
      message: t('consumer.resetPasswordHint', { name: record.consumerName }),
      value: result.tempPassword,
    });
  } catch {
    showError(t('consumer.resetPasswordFailed'));
  }
}

async function handleStatus(record: OrgAccountRecord, status: 'active' | 'disabled') {
  try {
    await updateOrgAccountStatus(record.consumerName, status);
    await load();
    showSuccess(status === 'active' ? t('consumer.enableSuccess') : t('consumer.disableSuccess'));
  } catch {
    showError(t('consumer.statusUpdateFailed'));
  }
}

async function openConsumerDetail(record: OrgAccountRecord) {
  detailOpen.value = true;
  detailLoading.value = true;
  consumerDetail.value = null;
  try {
    consumerDetail.value = await getConsumerDetail(record.consumerName);
  } catch {
    showError('加载 Consumer 详情失败');
  } finally {
    detailLoading.value = false;
  }
}

function resolveCredentialList(detail?: ConsumerDetail | null) {
  if (!detail?.credentials || !Array.isArray(detail.credentials)) {
    return [];
  }
  return detail.credentials
    .flatMap((item) => {
      if (!item) {
        return [];
      }
      if (typeof item === 'string') {
        return [item];
      }
      if (Array.isArray((item as any).values)) {
        return (item as any).values;
      }
      if ((item as any).value) {
        return [(item as any).value];
      }
      return [];
    })
    .filter((item) => String(item || '').trim() !== '');
}

async function handleInviteStatus(record: InviteCodeRecord, status: 'active' | 'disabled') {
  try {
    if (status === 'active') {
      await enableInviteCode(record.inviteCode);
      showSuccess(t('consumer.inviteCode.enableSuccess'));
    } else {
      await disableInviteCode(record.inviteCode);
      showSuccess(t('consumer.inviteCode.disableSuccess'));
    }
    await load();
  } catch {
    showError(status === 'active' ? t('consumer.inviteCode.enableFailed') : t('consumer.inviteCode.disableFailed'));
  }
}

async function createInvite() {
  try {
    const result = await createInviteCode(7);
    await load();
    showSuccess(t('consumer.inviteCode.createSuccess'));
    showCopyValueModal({
      title: t('consumer.inviteCode.codeGeneratedTitle'),
      message: t('consumer.inviteCode.codeGeneratedHint'),
      value: result.inviteCode,
    });
  } catch {
    showError(t('consumer.inviteCode.createFailed'));
  }
}

onMounted(load);
</script>

<template>
  <div class="consumer-page">
    <PageSection v-if="portalUnavailable" :title="t('menu.consumerManagement')">
      <PortalUnavailableState />
    </PageSection>

    <template v-else>
      <div class="consumer-page__layout">
        <PageSection :title="t('consumer.departmentTree')" subtle>
          <template #actions>
            <a-button size="small" @click="openCreateDepartment">{{ t('consumer.createDepartment') }}</a-button>
          </template>

          <a-tree
            :tree-data="departmentTreeData"
            block-node
            @select="handleDepartmentSelect"
          />
        </PageSection>

        <PageSection :title="t('menu.consumerManagement')">
          <template #actions>
            <a-button @click="load">{{ t('misc.refresh') }}</a-button>
            <a-button type="primary" @click="openCreateAccount">{{ t('consumer.create') }}</a-button>
          </template>

          <a-table
            :data-source="filteredAccounts"
            :loading="loading"
            row-key="consumerName"
            :scroll="{ x: 920 }"
            size="middle"
          >
            <a-table-column key="consumerName" data-index="consumerName" :title="t('consumer.columns.name')" />
            <a-table-column key="displayName" data-index="displayName" :title="t('consumer.columns.displayName')" />
            <a-table-column key="departmentPath" :title="t('consumer.columns.department')">
              <template #default="{ record }">
                {{ record.departmentPath || t('consumer.notAssigned') }}
              </template>
            </a-table-column>
            <a-table-column key="userLevel" :title="t('consumer.columns.userLevel')">
              <template #default="{ record }">
                {{ resolveUserLevelText(record.userLevel) }}
              </template>
            </a-table-column>
            <a-table-column key="status" :title="t('consumer.columns.portalStatus')">
              <template #default="{ record }">
                <StatusTag :value="record.status" :text="resolveAccountStatusText(record.status)" />
              </template>
            </a-table-column>
            <a-table-column key="actions" :title="t('misc.actions')" fixed="right" width="220">
              <template #default="{ record }">
                <a-button type="link" size="small" @click="openConsumerDetail(record)">详情</a-button>
                <a-button type="link" size="small" @click="openEditAccount(record)">{{ t('misc.edit') }}</a-button>
                <a-button type="link" size="small" @click="handleResetPassword(record)">{{ t('consumer.resetPassword') }}</a-button>
                <a-button
                  v-if="String(record.status || '').toLowerCase() === 'active'"
                  type="link"
                  size="small"
                  danger
                  @click="handleStatus(record, 'disabled')"
                >
                  {{ t('consumer.disable') }}
                </a-button>
                <a-button
                  v-else
                  type="link"
                  size="small"
                  @click="handleStatus(record, 'active')"
                >
                  {{ t('consumer.enable') }}
                </a-button>
              </template>
            </a-table-column>
          </a-table>
        </PageSection>
      </div>

      <PageSection :title="t('consumer.inviteCode.manage')">
        <template #actions>
          <a-button type="primary" @click="createInvite">{{ t('consumer.inviteCode.create') }}</a-button>
        </template>

        <a-table :data-source="inviteCodes" :loading="inviteLoading" row-key="inviteCode" size="small" :scroll="{ x: 980 }">
          <a-table-column key="inviteCode" data-index="inviteCode" :title="t('consumer.inviteCode.columns.code')" />
          <a-table-column key="status" :title="t('consumer.inviteCode.columns.status')">
            <template #default="{ record }">
              <StatusTag :value="record.status" :text="resolveInviteStatusText(record.status)" />
            </template>
          </a-table-column>
          <a-table-column key="expiresAt" :title="t('consumer.inviteCode.columns.expiresAt')">
            <template #default="{ record }">{{ formatDateTimeDisplay(record.expiresAt) }}</template>
          </a-table-column>
          <a-table-column key="usedByConsumer" :title="t('consumer.inviteCode.columns.usedBy')">
            <template #default="{ record }">{{ record.usedByConsumer || '-' }}</template>
          </a-table-column>
          <a-table-column key="usedAt" :title="t('consumer.inviteCode.columns.usedAt')">
            <template #default="{ record }">{{ formatDateTimeDisplay(record.usedAt) }}</template>
          </a-table-column>
          <a-table-column key="createdAt" :title="t('consumer.inviteCode.columns.createdAt')">
            <template #default="{ record }">{{ formatDateTimeDisplay(record.createdAt) }}</template>
          </a-table-column>
          <a-table-column key="actions" :title="t('misc.actions')" width="120">
            <template #default="{ record }">
              <a-button
                v-if="String(record.status || '').toLowerCase() === 'active'"
                type="link"
                size="small"
                danger
                @click="handleInviteStatus(record, 'disabled')"
              >
                {{ t('consumer.inviteCode.disable') }}
              </a-button>
              <a-button
                v-else-if="String(record.status || '').toLowerCase() === 'disabled'"
                type="link"
                size="small"
                @click="handleInviteStatus(record, 'active')"
              >
                {{ t('consumer.inviteCode.enable') }}
              </a-button>
              <span v-else>-</span>
            </template>
          </a-table-column>
        </a-table>
      </PageSection>
    </template>

    <a-modal
      v-model:open="accountModalOpen"
      :title="editingAccount ? t('consumer.edit') : t('consumer.create')"
      :confirm-loading="accountSaving"
      destroy-on-close
      @ok="saveAccount"
    >
      <a-form ref="accountFormRef" layout="vertical" :model="accountForm">
        <a-form-item
          :label="t('consumer.consumerForm.name')"
          name="consumerName"
          :rules="[{ required: true, message: t('consumer.consumerForm.nameRequired') }]"
        >
          <a-input v-model:value="accountForm.consumerName" :disabled="Boolean(editingAccount)" />
        </a-form-item>
        <a-form-item
          :label="t('consumer.consumerForm.displayName')"
          name="displayName"
          :rules="[{ required: true, message: t('consumer.consumerForm.displayNameRequired') }]"
        >
          <a-input v-model:value="accountForm.displayName" />
        </a-form-item>
        <a-form-item :label="t('consumer.consumerForm.email')" name="email">
          <a-input v-model:value="accountForm.email" />
        </a-form-item>
        <a-form-item
          :label="t('consumer.consumerForm.portalUserLevel')"
          name="userLevel"
          :rules="[{ required: true, message: t('consumer.consumerForm.portalUserLevelRequired') }]"
        >
          <a-select v-model:value="accountForm.userLevel">
            <a-select-option value="normal">{{ resolveUserLevelText('normal') }}</a-select-option>
            <a-select-option value="plus">{{ resolveUserLevelText('plus') }}</a-select-option>
            <a-select-option value="pro">{{ resolveUserLevelText('pro') }}</a-select-option>
            <a-select-option value="ultra">{{ resolveUserLevelText('ultra') }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :label="t('consumer.consumerForm.department')" name="departmentId">
          <a-select
            v-model:value="accountForm.departmentId"
            allow-clear
            show-search
            :options="departmentOptions"
            :placeholder="t('consumer.consumerForm.departmentPlaceholder')"
          />
        </a-form-item>
        <a-form-item :label="t('consumer.consumerForm.password')" name="password">
          <a-input-password v-model:value="accountForm.password" :placeholder="t('consumer.consumerForm.passwordPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="departmentModalOpen"
      :title="t('consumer.createDepartment')"
      :confirm-loading="departmentSaving"
      destroy-on-close
      @ok="saveDepartment"
    >
      <a-form ref="departmentFormRef" layout="vertical" :model="departmentForm">
        <a-form-item
          :label="t('consumer.departmentForm.name')"
          name="name"
          :rules="[{ required: true, message: t('consumer.departmentForm.nameRequired') }]"
        >
          <a-input v-model:value="departmentForm.name" :placeholder="t('consumer.departmentForm.namePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('consumer.departmentForm.parentDepartment')" name="parentDepartmentId">
          <a-select
            v-model:value="departmentForm.parentDepartmentId"
            allow-clear
            show-search
            :options="departmentOptions"
            :placeholder="t('consumer.departmentForm.parentDepartmentPlaceholder')"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="detailOpen" width="720" title="Consumer 详情" destroy-on-close>
      <a-skeleton :loading="detailLoading" active>
        <a-empty v-if="!consumerDetail" description="暂无可展示的 Consumer 详情" />
        <div v-else class="consumer-page__detail">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="Consumer Name">{{ consumerDetail.name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="状态">{{ resolveAccountStatusText(consumerDetail.portalStatus) }}</a-descriptions-item>
            <a-descriptions-item label="显示名">{{ consumerDetail.portalDisplayName || '-' }}</a-descriptions-item>
            <a-descriptions-item label="邮箱">{{ consumerDetail.portalEmail || '-' }}</a-descriptions-item>
            <a-descriptions-item label="用户等级">{{ resolveUserLevelText(consumerDetail.portalUserLevel) }}</a-descriptions-item>
            <a-descriptions-item label="用户来源">{{ consumerDetail.portalUserSource || '-' }}</a-descriptions-item>
            <a-descriptions-item label="部门">{{ consumerDetail.department || '-' }}</a-descriptions-item>
            <a-descriptions-item label="部门 ID">{{ consumerDetail.departmentId || '-' }}</a-descriptions-item>
            <a-descriptions-item label="部门路径" :span="2">{{ consumerDetail.departmentPath || '-' }}</a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ formatDateTimeDisplay(consumerDetail.createdAt) }}</a-descriptions-item>
            <a-descriptions-item label="更新时间">{{ formatDateTimeDisplay(consumerDetail.updatedAt) }}</a-descriptions-item>
            <a-descriptions-item label="最近登录" :span="2">{{ formatDateTimeDisplay(consumerDetail.lastLoginAt) }}</a-descriptions-item>
          </a-descriptions>

          <PageSection title="可见凭证概要" subtle>
            <a-empty v-if="!resolveCredentialList(consumerDetail).length" description="当前没有可展示的活跃凭证。" />
            <a-list
              v-else
              size="small"
              bordered
              :data-source="resolveCredentialList(consumerDetail)"
            >
              <template #renderItem="{ item }">
                <a-list-item>{{ item }}</a-list-item>
              </template>
            </a-list>
          </PageSection>
        </div>
      </a-skeleton>
    </a-drawer>
  </div>
</template>

<style scoped>
.consumer-page {
  display: grid;
  gap: 18px;
}

.consumer-page__layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 18px;
}

.consumer-page__detail {
  display: grid;
  gap: 18px;
}

@media (max-width: 1023px) {
  .consumer-page__layout {
    grid-template-columns: 1fr;
  }
}
</style>
