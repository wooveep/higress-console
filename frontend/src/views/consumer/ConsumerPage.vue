<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import { showConfirm, showCopyValueModal, showError, showSuccess } from '@/lib/feedback';
import { formatDateTimeDisplay } from '@/utils/time';
import { useI18n } from 'vue-i18n';
import type { ConsumerDetail, InviteCodeRecord } from '@/interfaces/consumer';
import type { OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import {
  createOrgAccount,
  createOrgDepartment,
  deleteOrgAccount,
  listOrgAccounts,
  listOrgDepartmentsTree,
  rebindOrgAccountSSOIdentity,
  updateOrgAccount,
  updateOrgDepartment,
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
const ssoRebindSaving = ref(false);

const departments = ref<OrgDepartmentNode[]>([]);
const accounts = ref<OrgAccountRecord[]>([]);
const inviteCodes = ref<InviteCodeRecord[]>([]);
const selectedDepartmentId = ref<string>();
const accountModalOpen = ref(false);
const departmentModalOpen = ref(false);
const departmentModalMode = ref<'create' | 'edit'>('create');
const detailOpen = ref(false);
const detailLoading = ref(false);
const editingAccount = ref<OrgAccountRecord | null>(null);
const consumerDetail = ref<ConsumerDetail | null>(null);
const ssoRebindModalOpen = ref(false);
const ssoRebindSourceAccount = ref<OrgAccountRecord | null>(null);

const accountFormRef = ref();
const departmentFormRef = ref();
const ssoRebindFormRef = ref();

const accountForm = reactive({
  consumerName: '',
  displayName: '',
  email: '',
  userLevel: 'normal',
  status: 'active',
  departmentId: '',
  password: '',
});

const departmentForm = reactive({
  name: '',
  parentDepartmentId: '',
  adminMode: 'existing',
  adminConsumerName: '',
  adminDisplayName: '',
  adminEmail: '',
  adminUserLevel: 'normal',
  adminPassword: '',
});

const ssoRebindForm = reactive({
  targetConsumerName: '',
});

const filteredAccounts = computed(() => {
  if (!selectedDepartmentId.value) {
    return accounts.value;
  }
  return accounts.value.filter((item) => item.departmentId === selectedDepartmentId.value);
});

const selectedDepartment = computed(() => findDepartmentNode(departments.value, selectedDepartmentId.value));
const departmentTreeData = computed(() => toDepartmentTree(departments.value));
const departmentOptions = computed(() => flattenDepartmentOptions(departments.value));
const departmentAdminOptions = computed(() => {
  const departmentId = selectedDepartment.value?.departmentId;
  if (!departmentId) {
    return [];
  }
  return accounts.value
    .filter((item) => item.departmentId === departmentId && String(item.status || '').toLowerCase() === 'active')
    .map((item) => ({
      label: `${item.displayName || item.consumerName} (${item.consumerName})`,
      value: item.consumerName,
    }));
});
const departmentExistingAdminOptions = computed(() => accounts.value
  .filter((item) => String(item.status || '').toLowerCase() === 'active')
  .map((item) => ({
    label: [
      `${item.displayName || item.consumerName} (${item.consumerName})`,
      item.email || '',
      item.departmentPath || t('consumer.notAssigned'),
    ].filter(Boolean).join(' · '),
    value: item.consumerName,
  })));
const ssoRebindTargetOptions = computed(() => {
  const sourceConsumerName = ssoRebindSourceAccount.value?.consumerName;
  return accounts.value
    .filter((item) => item.consumerName !== sourceConsumerName)
    .map((item) => ({
      label: [
        `${item.displayName || item.consumerName} (${item.consumerName})`,
        item.email || '',
        resolveAccountStatusText(item.status),
      ].filter(Boolean).join(' · '),
      value: item.consumerName,
    }));
});

function isRebindableSSOAccount(record: OrgAccountRecord) {
  const source = String(record.source || '').toLowerCase();
  const status = String(record.status || '').toLowerCase();
  return source === 'sso' && ['pending', 'disabled'].includes(status);
}

function toDepartmentTree(nodes: OrgDepartmentNode[]): any[] {
  return (nodes || []).map((node) => ({
    key: node.departmentId,
    title: `${node.name} (${node.memberCount || 0}) / ${node.adminConsumerName || '-'}`,
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

function findDepartmentNode(nodes: OrgDepartmentNode[], departmentId?: string): OrgDepartmentNode | undefined {
  if (!departmentId) {
    return undefined;
  }
  for (const node of nodes || []) {
    if (node.departmentId === departmentId) {
      return node;
    }
    const child = findDepartmentNode(node.children || [], departmentId);
    if (child) {
      return child;
    }
  }
  return undefined;
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
    status: 'active',
    departmentId: selectedDepartmentId.value || '',
    password: '',
  });
  accountModalOpen.value = true;
}

function openCreateDepartment() {
  departmentModalMode.value = 'create';
  Object.assign(departmentForm, {
    name: '',
    parentDepartmentId: selectedDepartmentId.value || '',
    adminMode: 'existing',
    adminConsumerName: '',
    adminDisplayName: '',
    adminEmail: '',
    adminUserLevel: 'normal',
    adminPassword: '',
  });
  departmentModalOpen.value = true;
}

function openEditDepartment() {
  if (!selectedDepartment.value) {
    return;
  }
  departmentModalMode.value = 'edit';
  Object.assign(departmentForm, {
    name: selectedDepartment.value.name || '',
    parentDepartmentId: selectedDepartment.value.parentDepartmentId || '',
    adminMode: 'existing',
    adminConsumerName: selectedDepartment.value.adminConsumerName || '',
    adminDisplayName: '',
    adminEmail: '',
    adminUserLevel: 'normal',
    adminPassword: '',
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
    status: record.status || 'active',
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
      status: accountForm.status || 'active',
      departmentId: accountForm.departmentId || undefined,
      password: accountForm.password || undefined,
    };
    if (editingAccount.value) {
      await updateOrgAccount(editingAccount.value.consumerName, payload);
    } else {
      const created = await createOrgAccount(payload);
      if (created.tempPassword) {
        showCopyValueModal({
          title: t('consumer.resetPasswordTitle'),
          message: t('consumer.resetPasswordHint', { name: created.consumerName }),
          value: created.tempPassword,
        });
      }
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
    if (departmentModalMode.value === 'create') {
      const payload = {
        name: departmentForm.name.trim(),
        parentDepartmentId: departmentForm.parentDepartmentId || undefined,
        adminMode: departmentForm.adminMode,
        adminConsumerName: departmentForm.adminConsumerName.trim(),
      };
      if (departmentForm.adminMode === 'create') {
        Object.assign(payload, {
          adminDisplayName: departmentForm.adminDisplayName.trim(),
          adminEmail: departmentForm.adminEmail.trim() || undefined,
          adminUserLevel: departmentForm.adminUserLevel || 'normal',
          adminPassword: departmentForm.adminPassword || undefined,
        });
      }
      const created = await createOrgDepartment(payload);
      if (created.createdAdminTempPassword) {
        showCopyValueModal({
          title: t('consumer.departmentAdminTempPasswordTitle'),
          message: t('consumer.departmentAdminTempPasswordHint', { name: created.adminConsumerName || departmentForm.adminConsumerName.trim() }),
          value: created.createdAdminTempPassword,
        });
      }
      showSuccess(t('consumer.departmentCreateSuccess'));
    } else if (selectedDepartment.value) {
      await updateOrgDepartment(selectedDepartment.value.departmentId, {
        name: departmentForm.name.trim(),
        adminConsumerName: departmentForm.adminConsumerName || undefined,
      });
      showSuccess(t('consumer.departmentUpdateSuccess'));
    }
    departmentModalOpen.value = false;
    Object.assign(departmentForm, {
      name: '',
      parentDepartmentId: '',
      adminMode: 'existing',
      adminConsumerName: '',
      adminDisplayName: '',
      adminEmail: '',
      adminUserLevel: 'normal',
      adminPassword: '',
    });
    await load();
  } finally {
    departmentSaving.value = false;
  }
}

function openSSORebind(record: OrgAccountRecord) {
  ssoRebindSourceAccount.value = record;
  ssoRebindForm.targetConsumerName = '';
  ssoRebindModalOpen.value = true;
}

async function saveSSORebind() {
  if (!ssoRebindSourceAccount.value) {
    return;
  }
  await ssoRebindFormRef.value?.validate();

  ssoRebindSaving.value = true;
  try {
    await rebindOrgAccountSSOIdentity(ssoRebindSourceAccount.value.consumerName, {
      targetConsumerName: ssoRebindForm.targetConsumerName,
    });
    ssoRebindModalOpen.value = false;
    ssoRebindSourceAccount.value = null;
    ssoRebindForm.targetConsumerName = '';
    await load();
    showSuccess(t('consumer.ssoRebindSuccess'));
  } catch {
    showError(t('consumer.ssoRebindFailed'));
  } finally {
    ssoRebindSaving.value = false;
  }
}

function handleDeleteAccount(record: OrgAccountRecord) {
  showConfirm({
    title: t('consumer.deleteTitle'),
    content: `${t('consumer.deleteSoftHint')} ${record.consumerName}`,
    okText: t('misc.ok'),
    cancelText: t('misc.cancel'),
    okButtonProps: {
      danger: true,
    },
    async onOk() {
      try {
        await deleteOrgAccount(record.consumerName);
        await load();
        showSuccess(t('consumer.deleteSuccess'));
      } catch {
        showError(t('consumer.deleteFailed'));
      }
    },
  });
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
            <a-button size="small" :disabled="!selectedDepartment" @click="openEditDepartment">{{ t('consumer.editDepartment') }}</a-button>
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
            <a-table-column key="actions" :title="t('misc.actions')" fixed="right" width="320">
              <template #default="{ record }">
                <a-button type="link" size="small" @click="openConsumerDetail(record)">详情</a-button>
                <a-button type="link" size="small" @click="openEditAccount(record)">{{ t('misc.edit') }}</a-button>
                <a-button type="link" size="small" @click="handleResetPassword(record)">{{ t('consumer.resetPassword') }}</a-button>
                <a-button
                  v-if="isRebindableSSOAccount(record)"
                  type="link"
                  size="small"
                  @click="openSSORebind(record)"
                >
                  {{ t('consumer.ssoRebind') }}
                </a-button>
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
                <a-button type="link" size="small" danger @click="handleDeleteAccount(record)">
                  {{ t('misc.delete') }}
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
      :title="departmentModalMode === 'create' ? t('consumer.createDepartment') : t('consumer.editDepartment')"
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
            option-filter-prop="label"
            :options="departmentOptions"
            :placeholder="t('consumer.departmentForm.parentDepartmentPlaceholder')"
            :disabled="departmentModalMode === 'edit'"
          />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create'"
          :label="t('consumer.departmentForm.adminMode')"
          name="adminMode"
        >
          <a-radio-group v-model:value="departmentForm.adminMode">
            <a-radio-button value="existing">{{ t('consumer.departmentForm.adminModeExisting') }}</a-radio-button>
            <a-radio-button value="create">{{ t('consumer.departmentForm.adminModeCreate') }}</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-alert
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'existing'"
          type="info"
          show-icon
          :message="t('consumer.departmentForm.adminModeExistingHint')"
          class="consumer-page__inline-alert"
        />
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'create'"
          :label="t('consumer.departmentForm.adminConsumerName')"
          name="adminConsumerName"
          :rules="[{ required: true, message: t('consumer.departmentForm.adminConsumerNameRequired') }]"
        >
          <a-input v-model:value="departmentForm.adminConsumerName" :placeholder="t('consumer.departmentForm.adminConsumerNamePlaceholder')" />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'create'"
          :label="t('consumer.departmentForm.adminDisplayName')"
          name="adminDisplayName"
          :rules="[{ required: true, message: t('consumer.departmentForm.adminDisplayNameRequired') }]"
        >
          <a-input v-model:value="departmentForm.adminDisplayName" :placeholder="t('consumer.departmentForm.adminDisplayNamePlaceholder')" />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'existing'"
          :label="t('consumer.departmentForm.adminConsumerName')"
          name="adminConsumerName"
          :rules="[{ required: true, message: t('consumer.departmentForm.adminConsumerNameRequired') }]"
        >
          <a-select
            v-model:value="departmentForm.adminConsumerName"
            show-search
            option-filter-prop="label"
            :options="departmentExistingAdminOptions"
            :placeholder="t('consumer.departmentForm.adminConsumerNameExistingPlaceholder')"
          />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'create'"
          :label="t('consumer.departmentForm.adminEmail')"
          name="adminEmail"
        >
          <a-input v-model:value="departmentForm.adminEmail" :placeholder="t('consumer.departmentForm.adminEmailPlaceholder')" />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'create'"
          :label="t('consumer.departmentForm.adminUserLevel')"
          name="adminUserLevel"
          :rules="[{ required: true, message: t('consumer.departmentForm.adminUserLevelRequired') }]"
        >
          <a-select v-model:value="departmentForm.adminUserLevel">
            <a-select-option value="normal">{{ resolveUserLevelText('normal') }}</a-select-option>
            <a-select-option value="plus">{{ resolveUserLevelText('plus') }}</a-select-option>
            <a-select-option value="pro">{{ resolveUserLevelText('pro') }}</a-select-option>
            <a-select-option value="ultra">{{ resolveUserLevelText('ultra') }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'create' && departmentForm.adminMode === 'create'"
          :label="t('consumer.departmentForm.adminPassword')"
          name="adminPassword"
        >
          <a-input-password v-model:value="departmentForm.adminPassword" :placeholder="t('consumer.departmentForm.adminPasswordPlaceholder')" />
        </a-form-item>
        <a-form-item
          v-if="departmentModalMode === 'edit'"
          :label="t('consumer.departmentForm.adminConsumerName')"
          name="adminConsumerName"
          :rules="[{ required: true, message: t('consumer.departmentForm.adminConsumerNameRequired') }]"
        >
          <a-select
            v-model:value="departmentForm.adminConsumerName"
            show-search
            option-filter-prop="label"
            :options="departmentAdminOptions"
            :placeholder="t('consumer.departmentForm.adminConsumerNamePlaceholder')"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="ssoRebindModalOpen"
      :title="t('consumer.ssoRebindTitle')"
      :confirm-loading="ssoRebindSaving"
      destroy-on-close
      @ok="saveSSORebind"
    >
      <a-alert
        type="info"
        show-icon
        :message="t('consumer.ssoRebindHint', { name: ssoRebindSourceAccount?.consumerName || '-' })"
        class="consumer-page__inline-alert"
      />
      <a-form ref="ssoRebindFormRef" layout="vertical" :model="ssoRebindForm">
        <a-form-item
          :label="t('consumer.ssoRebindTarget')"
          name="targetConsumerName"
          :rules="[{ required: true, message: t('consumer.ssoRebindTargetRequired') }]"
        >
          <a-select
            v-model:value="ssoRebindForm.targetConsumerName"
            show-search
            option-filter-prop="label"
            :options="ssoRebindTargetOptions"
            :placeholder="t('consumer.ssoRebindTargetPlaceholder')"
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

.consumer-page__inline-alert {
  margin-bottom: 16px;
}

@media (max-width: 1023px) {
  .consumer-page__layout {
    grid-template-columns: 1fr;
  }
}
</style>
