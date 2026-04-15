<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import { showSuccess } from '@/lib/feedback';
import {
  deleteAiSensitiveDetectRule,
  deleteAiSensitiveReplaceRule,
  getAiSensitiveAudits,
  getAiSensitiveDetectRules,
  getAiSensitiveReplaceRules,
  getAiSensitiveStatus,
  getAiSensitiveSystemConfig,
  saveAiSensitiveDetectRule,
  saveAiSensitiveReplaceRule,
  updateAiSensitiveSystemConfig,
} from '@/services/ai-sensitive';

const { t } = useI18n();
const { portalUnavailable } = usePortalAvailability();
const status = ref<any>({});
const detectRules = ref<any[]>([]);
const replaceRules = ref<any[]>([]);
const audits = ref<any[]>([]);
const systemConfig = reactive<any>({
  systemDenyEnabled: false,
  dictionaryText: '',
});

const editorOpen = ref(false);
const editorKind = ref<'detect' | 'replace'>('detect');
const editorJson = ref('{}');

const auditQuery = reactive({
  consumerName: '',
  routeName: '',
});

async function load() {
  if (portalUnavailable.value) {
    status.value = {};
    detectRules.value = [];
    replaceRules.value = [];
    Object.assign(systemConfig, {
      systemDenyEnabled: false,
      dictionaryText: '',
    });
    return;
  }
  const [nextStatus, nextDetect, nextReplace, nextSystem] = await Promise.all([
    getAiSensitiveStatus().catch(() => ({})),
    getAiSensitiveDetectRules().catch(() => []),
    getAiSensitiveReplaceRules().catch(() => []),
    getAiSensitiveSystemConfig().catch(() => ({ systemDenyEnabled: false, dictionaryText: '' })),
  ]);
  status.value = nextStatus;
  detectRules.value = nextDetect;
  replaceRules.value = nextReplace;
  Object.assign(systemConfig, nextSystem);
}

async function queryAudits() {
  if (portalUnavailable.value) {
    audits.value = [];
    return;
  }
  audits.value = await getAiSensitiveAudits({
    consumerName: auditQuery.consumerName || undefined,
    routeName: auditQuery.routeName || undefined,
  }).catch(() => []);
}

function openEditor(kind: 'detect' | 'replace', record?: any) {
  editorKind.value = kind;
  editorJson.value = JSON.stringify(record || {
    pattern: '',
    matchType: 'contains',
    replaceType: 'replace',
    replaceValue: '*',
  }, null, 2);
  editorOpen.value = true;
}

async function saveRule() {
  const payload = JSON.parse(editorJson.value);
  if (editorKind.value === 'detect') {
    await saveAiSensitiveDetectRule(payload);
  } else {
    await saveAiSensitiveReplaceRule(payload);
  }
  editorOpen.value = false;
  await load();
  showSuccess(t('misc.save'));
}

async function removeDetect(record: any) {
  await deleteAiSensitiveDetectRule(record.id);
  await load();
}

async function removeReplace(record: any) {
  await deleteAiSensitiveReplaceRule(record.id);
  await load();
}

async function saveSystemConfig() {
  await updateAiSensitiveSystemConfig({ ...systemConfig });
  showSuccess(t('misc.save'));
}

function formatBlockedDetail(record: any) {
  const first = record?.blockedDetails?.[0];
  if (!first) {
    return '-';
  }
  return [first.type, first.level, first.suggestion].filter(Boolean).join(' / ');
}

function formatGuardCode(record: any) {
  if (typeof record?.guardCode === 'number') {
    return String(record.guardCode);
  }
  return '-';
}

onMounted(async () => {
  await load();
  await queryAudits();
});
</script>

<template>
  <div class="ai-sensitive-page">
    <PageSection v-if="portalUnavailable" :title="t('menu.aiSensitiveManagement')">
      <PortalUnavailableState />
    </PageSection>

    <template v-else>
      <PageSection :title="t('menu.aiSensitiveManagement')">
        <div class="ai-sensitive-page__stats">
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.detectRuleCount') }}</span>
            <strong>{{ status.detectRuleCount || 0 }}</strong>
          </article>
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.replaceRuleCount') }}</span>
            <strong>{{ status.replaceRuleCount || 0 }}</strong>
          </article>
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.auditRecordCount') }}</span>
            <strong>{{ status.auditRecordCount || 0 }}</strong>
          </article>
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.systemDenyEnabled') }}</span>
            <strong>{{ status.systemDenyEnabled ? 'ON' : 'OFF' }}</strong>
          </article>
        </div>
      </PageSection>

      <PageSection :title="t('menu.aiSensitiveManagement')">
        <a-tabs>
          <a-tab-pane key="detect" :tab="t('aiSensitive.tabs.detect')">
            <a-button type="primary" @click="openEditor('detect')">{{ t('aiSensitive.actions.addDetectRule') }}</a-button>
            <a-table :data-source="detectRules" row-key="id" size="small" style="margin-top: 12px">
              <a-table-column key="pattern" data-index="pattern" :title="t('aiSensitive.fields.pattern')" />
              <a-table-column key="matchType" data-index="matchType" :title="t('aiSensitive.fields.matchType')" />
              <a-table-column key="priority" data-index="priority" :title="t('aiSensitive.fields.priority')" />
              <a-table-column key="enabled" data-index="enabled" :title="t('aiSensitive.fields.enabled')" />
              <a-table-column key="actions" :title="t('misc.actions')" width="180">
                <template #default="{ record }">
                  <a-button type="link" size="small" @click="openEditor('detect', record)">{{ t('misc.edit') }}</a-button>
                  <a-button type="link" size="small" danger @click="removeDetect(record)">{{ t('misc.delete') }}</a-button>
                </template>
              </a-table-column>
            </a-table>
          </a-tab-pane>

          <a-tab-pane key="replace" :tab="t('aiSensitive.tabs.replace')">
            <a-button type="primary" @click="openEditor('replace')">{{ t('aiSensitive.actions.addReplaceRule') }}</a-button>
            <a-table :data-source="replaceRules" row-key="id" size="small" style="margin-top: 12px">
              <a-table-column key="pattern" data-index="pattern" :title="t('aiSensitive.fields.pattern')" />
              <a-table-column key="replaceType" data-index="replaceType" :title="t('aiSensitive.fields.replaceType')" />
              <a-table-column key="replaceValue" data-index="replaceValue" :title="t('aiSensitive.fields.replaceValue')" />
              <a-table-column key="enabled" data-index="enabled" :title="t('aiSensitive.fields.enabled')" />
              <a-table-column key="actions" :title="t('misc.actions')" width="180">
                <template #default="{ record }">
                  <a-button type="link" size="small" @click="openEditor('replace', record)">{{ t('misc.edit') }}</a-button>
                  <a-button type="link" size="small" danger @click="removeReplace(record)">{{ t('misc.delete') }}</a-button>
                </template>
              </a-table-column>
            </a-table>
          </a-tab-pane>

          <a-tab-pane key="audit" :tab="t('aiSensitive.tabs.audit')">
            <div class="ai-sensitive-page__audit-bar">
              <a-input v-model:value="auditQuery.consumerName" :placeholder="t('aiSensitive.placeholders.consumerName')" />
              <a-input v-model:value="auditQuery.routeName" :placeholder="t('aiSensitive.placeholders.routeName')" />
              <a-button type="primary" @click="queryAudits">{{ t('misc.search') }}</a-button>
            </div>
            <a-table :data-source="audits" row-key="id" size="small" :scroll="{ x: 1420 }">
              <a-table-column key="requestId" data-index="requestId" :title="t('aiSensitive.fields.requestId')" width="220" />
              <a-table-column key="consumerName" data-index="consumerName" :title="t('aiSensitive.fields.consumerName')" />
              <a-table-column key="routeName" data-index="routeName" :title="t('aiSensitive.fields.routeName')" />
              <a-table-column key="guardCode" title="Guard Code" width="120">
                <template #default="{ record }">
                  <span>{{ formatGuardCode(record) }}</span>
                </template>
              </a-table-column>
              <a-table-column key="matchedRule" data-index="matchedRule" :title="t('aiSensitive.fields.matchedRule')" />
              <a-table-column key="matchedExcerpt" data-index="matchedExcerpt" :title="t('aiSensitive.fields.matchedExcerpt')" width="240" />
              <a-table-column key="blockedDetails" title="Blocked Details" width="240">
                <template #default="{ record }">
                  <span>{{ formatBlockedDetail(record) }}</span>
                </template>
              </a-table-column>
              <a-table-column key="blockedReasonJson" title="原始结果" width="120">
                <template #default="{ record }">
                  <a-popover v-if="record.blockedReasonJson" placement="left">
                    <template #content>
                      <pre class="ai-sensitive-page__audit-json">{{ record.blockedReasonJson }}</pre>
                    </template>
                    <a-button type="link" size="small">查看 JSON</a-button>
                  </a-popover>
                  <span v-else>-</span>
                </template>
              </a-table-column>
            </a-table>
          </a-tab-pane>

          <a-tab-pane key="system" :tab="t('aiSensitive.systemConfig.title')">
            <a-form layout="vertical">
              <a-form-item :label="t('aiSensitive.systemConfig.enabled')">
                <a-switch v-model:checked="systemConfig.systemDenyEnabled" />
              </a-form-item>
              <a-form-item :label="t('aiSensitive.systemConfig.title')">
                <a-textarea v-model:value="systemConfig.dictionaryText" :rows="16" />
              </a-form-item>
              <a-button type="primary" @click="saveSystemConfig">{{ t('misc.save') }}</a-button>
            </a-form>
          </a-tab-pane>
        </a-tabs>
      </PageSection>
    </template>

    <a-drawer v-model:open="editorOpen" :title="t('misc.edit')" width="620">
      <a-textarea v-model:value="editorJson" :rows="24" spellcheck="false" />
      <div class="ai-sensitive-page__drawer-actions">
        <a-button @click="editorOpen = false">{{ t('misc.cancel') }}</a-button>
        <a-button type="primary" @click="saveRule">{{ t('misc.save') }}</a-button>
      </div>
    </a-drawer>
  </div>
</template>

<style scoped>
.ai-sensitive-page {
  display: grid;
  gap: 18px;
}

.ai-sensitive-page__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.ai-sensitive-page__stat {
  padding: 14px 16px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface-soft);
}

.ai-sensitive-page__stat span {
  display: block;
  margin-bottom: 8px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.ai-sensitive-page__audit-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 14px;
}

.ai-sensitive-page__drawer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 18px;
}

.ai-sensitive-page__audit-json {
  max-width: 420px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.5;
}

@media (max-width: 1023px) {
  .ai-sensitive-page__stats {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 767px) {
  .ai-sensitive-page__stats,
  .ai-sensitive-page__audit-bar {
    grid-template-columns: 1fr;
    display: grid;
  }
}
</style>
