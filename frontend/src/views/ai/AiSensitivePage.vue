<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type {
  AiSensitiveDetectRule,
  AiSensitiveReplaceRule,
} from '@/interfaces/ai-sensitive';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import {
  AI_SENSITIVE_DETECT_PRESETS,
  AI_SENSITIVE_JSONPATH_PRESETS,
  AI_SENSITIVE_REPLACE_PRESETS,
  applyJsonpathPresetKeys,
  buildAiSensitiveRuntimeConfig,
  createDefaultAiSensitiveRuntimeSettings,
  listSelectedJsonpathPresetKeys,
  parseAiSensitiveRuntimeSettings,
  splitMultilineValues,
} from '@/features/ai-sensitive/config';
import { showError, showSuccess } from '@/lib/feedback';
import { useI18n } from 'vue-i18n';
import {
  deleteAiSensitiveDetectRule,
  deleteAiSensitiveReplaceRule,
  getAiSensitiveAudits,
  getAiSensitiveDetectRules,
  getAiSensitiveReplaceRules,
  getAiSensitiveRuntimeConfig,
  getAiSensitiveStatus,
  getAiSensitiveSystemConfig,
  reconcileAiSensitiveRules,
  saveAiSensitiveDetectRule,
  saveAiSensitiveReplaceRule,
  updateAiSensitiveRuntimeConfig,
  updateAiSensitiveSystemConfig,
} from '@/services/ai-sensitive';

const { t } = useI18n();
const { portalUnavailable } = usePortalAvailability();

const status = ref<any>({});
const detectRules = ref<AiSensitiveDetectRule[]>([]);
const replaceRules = ref<AiSensitiveReplaceRule[]>([]);
const audits = ref<any[]>([]);

const systemConfig = reactive<any>({
  systemDenyEnabled: false,
  dictionaryText: '',
  updatedAt: undefined,
  updatedBy: '',
});

const runtimeSettings = reactive(createDefaultAiSensitiveRuntimeSettings());

const editorOpen = ref(false);
const editorKind = ref<'detect' | 'replace'>('detect');
const detectPresetKey = ref<string>();
const replacePresetKey = ref<string>();
const selectedJsonpathPresetKeys = ref<string[]>([]);
const runtimeSaving = ref(false);
const editorSaving = ref(false);
const systemDictionaryKeyword = ref('');

const detectForm = reactive<AiSensitiveDetectRule>(createEmptyDetectRule());
const replaceForm = reactive<AiSensitiveReplaceRule>(createEmptyReplaceRule());

const auditQuery = reactive({
  consumerName: '',
  routeName: '',
});

const detectPresetOptions = computed(() => AI_SENSITIVE_DETECT_PRESETS.map((item) => ({
  label: getDetectPresetLabel(item.key),
  value: item.key,
})));

const replacePresetOptions = computed(() => AI_SENSITIVE_REPLACE_PRESETS.map((item) => ({
  label: getReplacePresetLabel(item.key),
  value: item.key,
})));

const jsonpathPresetOptions = computed(() => AI_SENSITIVE_JSONPATH_PRESETS.map((item) => ({
  label: getJsonpathPresetLabel(item.key),
  value: item.key,
})));

const activeDetectPreset = computed(() => AI_SENSITIVE_DETECT_PRESETS.find((item) => item.key === detectPresetKey.value) || null);
const activeReplacePreset = computed(() => AI_SENSITIVE_REPLACE_PRESETS.find((item) => item.key === replacePresetKey.value) || null);
const activeJsonpathPresetDescriptions = computed(() => selectedJsonpathPresetKeys.value
  .map((key) => getJsonpathPresetDescription(key))
  .filter(Boolean));
const boundRouteLinks = computed(() => (status.value?.enabledRoutes || []).map((name: string) => ({
  name,
  path: `/ai/route/config?type=aiRoute&name=${encodeURIComponent(name)}`,
})));
const systemDictionaryEntries = computed(() => splitMultilineValues(systemConfig.dictionaryText));
const trimmedSystemDictionaryKeyword = computed(() => String(systemDictionaryKeyword.value || '').trim());
const systemDictionaryMatchedEntries = computed(() => {
  const keyword = trimmedSystemDictionaryKeyword.value.toLowerCase();
  if (!keyword) {
    return [];
  }
  return systemDictionaryEntries.value
    .filter((entry) => entry.toLowerCase().includes(keyword))
    .slice(0, 20);
});
const systemDictionaryMatchedCount = computed(() => {
  const keyword = trimmedSystemDictionaryKeyword.value.toLowerCase();
  if (!keyword) {
    return 0;
  }
  return systemDictionaryEntries.value.filter((entry) => entry.toLowerCase().includes(keyword)).length;
});

const systemDictionaryWordCount = computed(() => {
  const fromStatus = Number(status.value?.systemDictionaryWordCount || 0);
  if (fromStatus > 0) {
    return fromStatus;
  }
  return splitMultilineValues(systemConfig.dictionaryText).length;
});

async function load() {
  if (portalUnavailable.value) {
    status.value = {};
    detectRules.value = [];
    replaceRules.value = [];
    audits.value = [];
    Object.assign(systemConfig, {
      systemDenyEnabled: false,
      dictionaryText: '',
      updatedAt: undefined,
      updatedBy: '',
    });
    Object.assign(runtimeSettings, createDefaultAiSensitiveRuntimeSettings());
    selectedJsonpathPresetKeys.value = listSelectedJsonpathPresetKeys(runtimeSettings.denyJsonpathText);
    return;
  }

  const [nextStatus, nextDetect, nextReplace, nextSystem, nextRuntime] = await Promise.all([
    getAiSensitiveStatus().catch(() => ({})),
    getAiSensitiveDetectRules().catch(() => []),
    getAiSensitiveReplaceRules().catch(() => []),
    getAiSensitiveSystemConfig().catch(() => ({
      systemDenyEnabled: false,
      dictionaryText: '',
      updatedAt: undefined,
      updatedBy: '',
    })),
    getAiSensitiveRuntimeConfig().catch(() => null),
  ]);

  status.value = nextStatus;
  detectRules.value = nextDetect;
  replaceRules.value = nextReplace;
  Object.assign(systemConfig, nextSystem);
  Object.assign(runtimeSettings, parseAiSensitiveRuntimeSettings(nextRuntime));
  selectedJsonpathPresetKeys.value = listSelectedJsonpathPresetKeys(runtimeSettings.denyJsonpathText);
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

function createEmptyDetectRule(): AiSensitiveDetectRule {
  return {
    pattern: '',
    matchType: 'contains',
    description: '',
    priority: 100,
    enabled: true,
  };
}

function createEmptyReplaceRule(): AiSensitiveReplaceRule {
  return {
    pattern: '',
    replaceType: 'replace',
    replaceValue: '****',
    restore: false,
    description: '',
    priority: 100,
    enabled: true,
  };
}

function getDetectPresetLabel(key?: string) {
  switch (key) {
    case 'apiKey':
      return t('aiSensitive.detectPreset.apiKey.label');
    case 'mobilePhone':
      return t('aiSensitive.detectPreset.mobilePhone.label');
    case 'email':
      return t('aiSensitive.detectPreset.email.label');
    case 'idCard':
      return t('aiSensitive.detectPreset.idCard.label');
    default:
      return '';
  }
}

function getDetectPresetDescription(key?: string) {
  switch (key) {
    case 'apiKey':
      return t('aiSensitive.detectPreset.apiKey.description');
    case 'mobilePhone':
      return t('aiSensitive.detectPreset.mobilePhone.description');
    case 'email':
      return t('aiSensitive.detectPreset.email.description');
    case 'idCard':
      return t('aiSensitive.detectPreset.idCard.description');
    default:
      return '';
  }
}

function getReplacePresetLabel(key?: string) {
  switch (key) {
    case 'mobilePhone':
      return t('aiSensitive.replacePreset.mobilePhone.label');
    case 'email':
      return t('aiSensitive.replacePreset.email.label');
    case 'bankCard':
      return t('aiSensitive.replacePreset.bankCard.label');
    case 'ip':
      return t('aiSensitive.replacePreset.ip.label');
    case 'idCard':
      return t('aiSensitive.replacePreset.idCard.label');
    case 'apiKey':
      return t('aiSensitive.replacePreset.apiKey.label');
    default:
      return '';
  }
}

function getReplacePresetDescription(key?: string) {
  switch (key) {
    case 'mobilePhone':
      return t('aiSensitive.replacePreset.mobilePhone.description');
    case 'email':
      return t('aiSensitive.replacePreset.email.description');
    case 'bankCard':
      return t('aiSensitive.replacePreset.bankCard.description');
    case 'ip':
      return t('aiSensitive.replacePreset.ip.description');
    case 'idCard':
      return t('aiSensitive.replacePreset.idCard.description');
    case 'apiKey':
      return t('aiSensitive.replacePreset.apiKey.description');
    default:
      return '';
  }
}

function getJsonpathPresetLabel(key?: string) {
  switch (key) {
    case 'messagesContent':
      return t('aiSensitive.jsonpathPreset.messagesContent.label');
    case 'userMessages':
      return t('aiSensitive.jsonpathPreset.userMessages.label');
    case 'systemPrompt':
      return t('aiSensitive.jsonpathPreset.systemPrompt.label');
    case 'responsesInput':
      return t('aiSensitive.jsonpathPreset.responsesInput.label');
    default:
      return '';
  }
}

function getJsonpathPresetDescription(key?: string) {
  switch (key) {
    case 'messagesContent':
      return t('aiSensitive.jsonpathPreset.messagesContent.description');
    case 'userMessages':
      return t('aiSensitive.jsonpathPreset.userMessages.description');
    case 'systemPrompt':
      return t('aiSensitive.jsonpathPreset.systemPrompt.description');
    case 'responsesInput':
      return t('aiSensitive.jsonpathPreset.responsesInput.description');
    default:
      return '';
  }
}

function findDetectPreset(record?: Partial<AiSensitiveDetectRule>) {
  return AI_SENSITIVE_DETECT_PRESETS.find((item) => (
    item.pattern === String(record?.pattern || '')
    && item.matchType === record?.matchType
  ));
}

function findReplacePreset(record?: Partial<AiSensitiveReplaceRule>) {
  return AI_SENSITIVE_REPLACE_PRESETS.find((item) => (
    item.pattern === String(record?.pattern || '')
    && item.replaceType === record?.replaceType
    && String(item.replaceValue || '') === String(record?.replaceValue || '')
    && Boolean(item.restore) === Boolean(record?.restore)
  ));
}

function openDetectEditor(record?: AiSensitiveDetectRule) {
  editorKind.value = 'detect';
  detectPresetKey.value = findDetectPreset(record)?.key;
  Object.assign(detectForm, createEmptyDetectRule(), record || {});
  editorOpen.value = true;
}

function openReplaceEditor(record?: AiSensitiveReplaceRule) {
  editorKind.value = 'replace';
  replacePresetKey.value = findReplacePreset(record)?.key;
  Object.assign(replaceForm, createEmptyReplaceRule(), record || {});
  editorOpen.value = true;
}

function applyDetectPreset(presetKey?: string) {
  const preset = AI_SENSITIVE_DETECT_PRESETS.find((item) => item.key === presetKey);
  if (!preset) {
    return;
  }
  Object.assign(detectForm, {
    pattern: preset.pattern,
    matchType: preset.matchType,
    priority: preset.priority ?? detectForm.priority,
    enabled: true,
  });
}

function applyReplacePreset(presetKey?: string) {
  const preset = AI_SENSITIVE_REPLACE_PRESETS.find((item) => item.key === presetKey);
  if (!preset) {
    return;
  }
  Object.assign(replaceForm, {
    pattern: preset.pattern,
    replaceType: preset.replaceType,
    replaceValue: preset.replaceValue || '',
    restore: Boolean(preset.restore),
    priority: preset.priority ?? replaceForm.priority,
    enabled: true,
  });
}

function applySelectedJsonpathPresets(keys: string[]) {
  selectedJsonpathPresetKeys.value = [...keys];
  runtimeSettings.denyJsonpathText = applyJsonpathPresetKeys(keys);
}

function buildDetectPayload(): AiSensitiveDetectRule | null {
  const pattern = String(detectForm.pattern || '').trim();
  if (!pattern) {
    showError(t('aiSensitive.rules.patternRequired'));
    return null;
  }
  if (!detectForm.matchType) {
    showError(t('aiSensitive.rules.matchTypeRequired'));
    return null;
  }
  return {
    id: detectForm.id,
    pattern,
    matchType: detectForm.matchType,
    description: String(detectForm.description || '').trim() || undefined,
    priority: Number(detectForm.priority || 0),
    enabled: detectForm.enabled !== false,
  };
}

function buildReplacePayload(): AiSensitiveReplaceRule | null {
  const pattern = String(replaceForm.pattern || '').trim();
  if (!pattern) {
    showError(t('aiSensitive.rules.patternRequired'));
    return null;
  }
  if (!replaceForm.replaceType) {
    showError(t('aiSensitive.rules.replaceTypeRequired'));
    return null;
  }
  const replaceValue = String(replaceForm.replaceValue || '').trim();
  if (replaceForm.replaceType === 'replace' && !replaceValue) {
    showError(t('aiSensitive.rules.replaceValueRequired'));
    return null;
  }
  return {
    id: replaceForm.id,
    pattern,
    replaceType: replaceForm.replaceType,
    replaceValue: replaceValue || undefined,
    restore: Boolean(replaceForm.restore),
    description: String(replaceForm.description || '').trim() || undefined,
    priority: Number(replaceForm.priority || 0),
    enabled: replaceForm.enabled !== false,
  };
}

async function saveRule() {
  const payload = editorKind.value === 'detect'
    ? buildDetectPayload()
    : buildReplacePayload();
  if (!payload) {
    return;
  }

  try {
    editorSaving.value = true;
    if (editorKind.value === 'detect') {
      await saveAiSensitiveDetectRule(payload as AiSensitiveDetectRule);
      detectRules.value = await getAiSensitiveDetectRules().catch(() => detectRules.value);
    } else {
      await saveAiSensitiveReplaceRule(payload as AiSensitiveReplaceRule);
      replaceRules.value = await getAiSensitiveReplaceRules().catch(() => replaceRules.value);
    }
    status.value = await reconcileAiSensitiveRules().catch(() => status.value);
    editorOpen.value = false;
    showSuccess(t(editorKind.value === 'detect' ? 'aiSensitive.messages.detectSaved' : 'aiSensitive.messages.replaceSaved'));
  } catch (error: any) {
    showError(String(error?.message || error || t('misc.save')));
  } finally {
    editorSaving.value = false;
  }
}

async function removeDetect(record: AiSensitiveDetectRule) {
  try {
    await deleteAiSensitiveDetectRule(Number(record.id));
    detectRules.value = await getAiSensitiveDetectRules().catch(() => detectRules.value);
    status.value = await reconcileAiSensitiveRules().catch(() => status.value);
    showSuccess(t('aiSensitive.messages.detectDeleted'));
  } catch (error: any) {
    showError(String(error?.message || error || t('misc.delete')));
  }
}

async function removeReplace(record: AiSensitiveReplaceRule) {
  try {
    await deleteAiSensitiveReplaceRule(Number(record.id));
    replaceRules.value = await getAiSensitiveReplaceRules().catch(() => replaceRules.value);
    status.value = await reconcileAiSensitiveRules().catch(() => status.value);
    showSuccess(t('aiSensitive.messages.replaceDeleted'));
  } catch (error: any) {
    showError(String(error?.message || error || t('misc.delete')));
  }
}

async function saveRuntimeConfig() {
  try {
    runtimeSaving.value = true;
    await updateAiSensitiveSystemConfig({
      systemDenyEnabled: systemConfig.systemDenyEnabled,
      dictionaryText: systemConfig.dictionaryText,
    });
    await updateAiSensitiveRuntimeConfig(buildAiSensitiveRuntimeConfig({ ...runtimeSettings }));
    await load();
    showSuccess(t('aiSensitive.messages.runtimeConfigSaved'));
  } catch (error: any) {
    showError(String(error?.message || error || t('misc.save')));
  } finally {
    runtimeSaving.value = false;
  }
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

watch(() => runtimeSettings.denyJsonpathText, (value) => {
  selectedJsonpathPresetKeys.value = listSelectedJsonpathPresetKeys(value);
});

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
            <span>{{ t('aiSensitive.status.enabledRouteCount') }}</span>
            <strong>{{ status.enabledRouteCount || 0 }}</strong>
          </article>
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.systemDenyEnabled') }}</span>
            <strong>{{ systemConfig.systemDenyEnabled ? 'ON' : 'OFF' }}</strong>
          </article>
          <article class="ai-sensitive-page__stat">
            <span>{{ t('aiSensitive.status.systemDictionaryWordCount') }}</span>
            <strong>{{ systemDictionaryWordCount }}</strong>
          </article>
        </div>
      </PageSection>

      <PageSection :title="t('menu.aiSensitiveManagement')">
        <a-tabs>
          <a-tab-pane key="runtime" :tab="t('aiSensitive.tabs.runtime')">
            <a-form layout="vertical" class="ai-sensitive-page__runtime-form">
              <section class="ai-sensitive-page__form-section">
                <div class="ai-sensitive-page__section-heading">
                  <h3>{{ t('aiSensitive.runtime.bindingTitle') }}</h3>
                  <p>{{ t('aiSensitive.runtime.bindingHelp') }}</p>
                </div>

                <div class="ai-sensitive-page__binding-summary">
                  <strong>{{ t('aiSensitive.runtime.bindingCount', { count: status.enabledRouteCount || 0 }) }}</strong>
                  <div v-if="boundRouteLinks.length" class="ai-sensitive-page__binding-links">
                    <router-link
                      v-for="route in boundRouteLinks"
                      :key="route.name"
                      :to="route.path"
                      class="ai-sensitive-page__binding-link"
                    >
                      {{ route.name }}
                    </router-link>
                  </div>
                </div>
              </section>

              <section class="ai-sensitive-page__form-section">
                <div class="ai-sensitive-page__section-heading">
                  <h3>{{ t('aiSensitive.runtime.scopeTitle') }}</h3>
                  <p>{{ t('aiSensitive.runtime.scopeHelp') }}</p>
                </div>

                <div class="ai-sensitive-page__switch-grid">
                  <a-form-item :label="t('aiSensitive.fields.denyOpenai')">
                    <a-switch v-model:checked="runtimeSettings.denyOpenai" />
                  </a-form-item>
                  <a-form-item :label="t('aiSensitive.fields.denyRaw')">
                    <a-switch v-model:checked="runtimeSettings.denyRaw" />
                  </a-form-item>
                  <a-form-item :label="t('aiSensitive.systemConfig.enabled')">
                    <a-switch v-model:checked="systemConfig.systemDenyEnabled" />
                  </a-form-item>
                </div>

                <a-form-item :label="t('aiSensitive.fields.denyJsonpath')">
                  <a-select
                    mode="multiple"
                    :value="selectedJsonpathPresetKeys"
                    :options="jsonpathPresetOptions"
                    :placeholder="t('aiSensitive.placeholders.jsonpathPreset')"
                    style="margin-bottom: 12px"
                    @update:value="applySelectedJsonpathPresets"
                  />
                  <div v-if="activeJsonpathPresetDescriptions.length" class="ai-sensitive-page__preset-list">
                    <p
                      v-for="description in activeJsonpathPresetDescriptions"
                      :key="description"
                      class="ai-sensitive-page__preset-help"
                    >
                      {{ description }}
                    </p>
                  </div>
                  <a-textarea
                    v-model:value="runtimeSettings.denyJsonpathText"
                    :rows="4"
                    :placeholder="t('aiSensitive.placeholders.denyJsonpath')"
                  />
                </a-form-item>
              </section>

              <section class="ai-sensitive-page__form-section">
                <div class="ai-sensitive-page__section-heading">
                  <h3>{{ t('aiSensitive.systemConfig.title') }}</h3>
                  <p>{{ t('aiSensitive.systemConfig.dictionaryHelp') }}</p>
                </div>

                <div class="ai-sensitive-page__system-meta">
                  <span>{{ t('aiSensitive.systemConfig.wordCount', { count: systemDictionaryWordCount }) }}</span>
                  <span v-if="systemConfig.updatedBy">{{ t('aiSensitive.systemConfig.updatedBy') }}: {{ systemConfig.updatedBy }}</span>
                </div>

                <a-form-item :label="t('aiSensitive.systemConfig.searchLabel')">
                  <a-input
                    v-model:value="systemDictionaryKeyword"
                    allow-clear
                    :placeholder="t('aiSensitive.systemConfig.searchPlaceholder')"
                  />
                </a-form-item>

                <div class="ai-sensitive-page__system-search-result">
                  <p v-if="!trimmedSystemDictionaryKeyword" class="ai-sensitive-page__preset-help">
                    {{ t('aiSensitive.systemConfig.searchHint') }}
                  </p>
                  <a-alert
                    v-else-if="systemDictionaryMatchedCount > 0"
                    type="success"
                    show-icon
                    :message="t('aiSensitive.systemConfig.searchMatched', { keyword: trimmedSystemDictionaryKeyword })"
                    :description="t('aiSensitive.systemConfig.searchResultCount', { count: systemDictionaryMatchedCount })"
                  />
                  <a-alert
                    v-else
                    type="warning"
                    show-icon
                    :message="t('aiSensitive.systemConfig.searchNotFound', { keyword: trimmedSystemDictionaryKeyword })"
                  />

                  <div v-if="systemDictionaryMatchedEntries.length" class="ai-sensitive-page__system-search-tags">
                    <a-tag v-for="entry in systemDictionaryMatchedEntries" :key="entry">
                      {{ entry }}
                    </a-tag>
                  </div>

                  <p
                    v-if="systemDictionaryMatchedCount > systemDictionaryMatchedEntries.length"
                    class="ai-sensitive-page__preset-help"
                  >
                    {{ t('aiSensitive.systemConfig.searchResultTruncated', { count: systemDictionaryMatchedEntries.length }) }}
                  </p>
                </div>
              </section>

              <section class="ai-sensitive-page__form-section">
                <div class="ai-sensitive-page__section-heading">
                  <h3>{{ t('aiSensitive.runtime.responseTitle') }}</h3>
                  <p>{{ t('aiSensitive.runtime.responseHelp') }}</p>
                </div>

                <div class="ai-sensitive-page__field-grid">
                  <a-form-item :label="t('aiSensitive.fields.denyCode')">
                    <a-input-number v-model:value="runtimeSettings.denyCode" :min="100" :max="599" style="width: 100%" />
                  </a-form-item>
                  <a-form-item :label="t('aiSensitive.fields.denyContentType')">
                    <a-input v-model:value="runtimeSettings.denyContentType" :placeholder="t('aiSensitive.placeholders.denyContentType')" />
                  </a-form-item>
                </div>

                <a-form-item :label="t('aiSensitive.fields.denyMessage')">
                  <a-textarea
                    v-model:value="runtimeSettings.denyMessage"
                    :rows="3"
                    :placeholder="t('aiSensitive.placeholders.denyMessage')"
                  />
                </a-form-item>
                <a-form-item :label="t('aiSensitive.fields.denyRawMessage')">
                  <a-textarea
                    v-model:value="runtimeSettings.denyRawMessage"
                    :rows="4"
                    :placeholder="t('aiSensitive.placeholders.denyRawMessage')"
                  />
                </a-form-item>
              </section>

              <div class="ai-sensitive-page__runtime-actions">
                <a-button type="primary" :loading="runtimeSaving" @click="saveRuntimeConfig">
                  {{ t('aiSensitive.actions.saveRuntimeConfig') }}
                </a-button>
              </div>
            </a-form>
          </a-tab-pane>

          <a-tab-pane key="detect" :tab="t('aiSensitive.tabs.detect')">
            <div class="ai-sensitive-page__toolbar">
              <a-button type="primary" @click="openDetectEditor()">
                {{ t('aiSensitive.actions.addDetectRule') }}
              </a-button>
            </div>

            <a-table :data-source="detectRules" row-key="id" size="small">
              <a-table-column key="pattern" data-index="pattern" :title="t('aiSensitive.fields.pattern')" />
              <a-table-column key="matchType" data-index="matchType" :title="t('aiSensitive.fields.matchType')" />
              <a-table-column key="priority" data-index="priority" :title="t('aiSensitive.fields.priority')" width="120" />
              <a-table-column key="enabled" :title="t('aiSensitive.fields.enabled')" width="100">
                <template #default="{ record }">
                  <span>{{ record.enabled === false ? 'OFF' : 'ON' }}</span>
                </template>
              </a-table-column>
              <a-table-column key="description" data-index="description" :title="t('aiSensitive.fields.description')" />
              <a-table-column key="actions" :title="t('misc.actions')" width="180">
                <template #default="{ record }">
                  <a-button type="link" size="small" @click="openDetectEditor(record)">{{ t('misc.edit') }}</a-button>
                  <a-popconfirm :title="t('aiSensitive.messages.deleteDetectConfirm')" @confirm="removeDetect(record)">
                    <a-button type="link" size="small" danger>{{ t('misc.delete') }}</a-button>
                  </a-popconfirm>
                </template>
              </a-table-column>
            </a-table>
          </a-tab-pane>

          <a-tab-pane key="replace" :tab="t('aiSensitive.tabs.replace')">
            <div class="ai-sensitive-page__toolbar">
              <a-button type="primary" @click="openReplaceEditor()">
                {{ t('aiSensitive.actions.addReplaceRule') }}
              </a-button>
            </div>

            <a-table :data-source="replaceRules" row-key="id" size="small">
              <a-table-column key="pattern" data-index="pattern" :title="t('aiSensitive.fields.pattern')" />
              <a-table-column key="replaceType" data-index="replaceType" :title="t('aiSensitive.fields.replaceType')" />
              <a-table-column key="replaceValue" data-index="replaceValue" :title="t('aiSensitive.fields.replaceValue')" />
              <a-table-column key="restore" :title="t('aiSensitive.fields.restore')" width="100">
                <template #default="{ record }">
                  <span>{{ record.restore ? 'ON' : 'OFF' }}</span>
                </template>
              </a-table-column>
              <a-table-column key="enabled" :title="t('aiSensitive.fields.enabled')" width="100">
                <template #default="{ record }">
                  <span>{{ record.enabled === false ? 'OFF' : 'ON' }}</span>
                </template>
              </a-table-column>
              <a-table-column key="actions" :title="t('misc.actions')" width="180">
                <template #default="{ record }">
                  <a-button type="link" size="small" @click="openReplaceEditor(record)">{{ t('misc.edit') }}</a-button>
                  <a-popconfirm :title="t('aiSensitive.messages.deleteReplaceConfirm')" @confirm="removeReplace(record)">
                    <a-button type="link" size="small" danger>{{ t('misc.delete') }}</a-button>
                  </a-popconfirm>
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
        </a-tabs>
      </PageSection>
    </template>

    <a-drawer
      v-model:open="editorOpen"
      :title="t(editorKind === 'detect'
        ? (detectForm.id ? 'aiSensitive.modals.editDetect' : 'aiSensitive.modals.addDetect')
        : (replaceForm.id ? 'aiSensitive.modals.editReplace' : 'aiSensitive.modals.addReplace'))"
      width="640"
    >
      <a-form v-if="editorKind === 'detect'" layout="vertical">
        <a-form-item :label="t('aiSensitive.actions.selectDetectPreset')">
          <a-select
            v-model:value="detectPresetKey"
            allow-clear
            :options="detectPresetOptions"
            :placeholder="t('aiSensitive.placeholders.detectPreset')"
            @change="applyDetectPreset"
          />
          <p v-if="activeDetectPreset" class="ai-sensitive-page__preset-help">
            {{ getDetectPresetDescription(activeDetectPreset.key) }}
          </p>
        </a-form-item>
        <a-form-item :label="t('aiSensitive.fields.pattern')">
          <a-input v-model:value="detectForm.pattern" />
        </a-form-item>
        <div class="ai-sensitive-page__field-grid">
          <a-form-item :label="t('aiSensitive.fields.matchType')">
            <a-select v-model:value="detectForm.matchType">
              <a-select-option value="contains">{{ t('aiSensitive.matchTypes.contains') }}</a-select-option>
              <a-select-option value="exact">{{ t('aiSensitive.matchTypes.exact') }}</a-select-option>
              <a-select-option value="regex">{{ t('aiSensitive.matchTypes.regex') }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="t('aiSensitive.fields.priority')">
            <a-input-number v-model:value="detectForm.priority" :min="0" style="width: 100%" />
          </a-form-item>
        </div>
        <a-form-item :label="t('aiSensitive.fields.description')">
          <a-input v-model:value="detectForm.description" />
        </a-form-item>
        <a-form-item :label="t('aiSensitive.fields.enabled')">
          <a-switch v-model:checked="detectForm.enabled" />
        </a-form-item>
      </a-form>

      <a-form v-else layout="vertical">
        <a-form-item :label="t('aiSensitive.fields.replacePreset')">
          <a-select
            v-model:value="replacePresetKey"
            allow-clear
            :options="replacePresetOptions"
            :placeholder="t('aiSensitive.placeholders.replacePreset')"
            @change="applyReplacePreset"
          />
          <p v-if="activeReplacePreset" class="ai-sensitive-page__preset-help">
            {{ getReplacePresetDescription(activeReplacePreset.key) }}
          </p>
        </a-form-item>
        <a-form-item :label="t('aiSensitive.fields.pattern')">
          <a-input v-model:value="replaceForm.pattern" />
        </a-form-item>
        <div class="ai-sensitive-page__field-grid">
          <a-form-item :label="t('aiSensitive.fields.replaceType')">
            <a-select v-model:value="replaceForm.replaceType">
              <a-select-option value="replace">{{ t('aiSensitive.replaceTypes.replace') }}</a-select-option>
              <a-select-option value="hash">{{ t('aiSensitive.replaceTypes.hash') }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="t('aiSensitive.fields.priority')">
            <a-input-number v-model:value="replaceForm.priority" :min="0" style="width: 100%" />
          </a-form-item>
        </div>
        <a-form-item :label="t('aiSensitive.fields.replaceValue')">
          <a-input v-model:value="replaceForm.replaceValue" />
        </a-form-item>
        <div class="ai-sensitive-page__switch-grid">
          <a-form-item :label="t('aiSensitive.fields.restore')">
            <a-switch v-model:checked="replaceForm.restore" />
          </a-form-item>
          <a-form-item :label="t('aiSensitive.fields.enabled')">
            <a-switch v-model:checked="replaceForm.enabled" />
          </a-form-item>
        </div>
        <a-form-item :label="t('aiSensitive.fields.description')">
          <a-input v-model:value="replaceForm.description" />
        </a-form-item>
      </a-form>

      <div class="ai-sensitive-page__drawer-actions">
        <a-button @click="editorOpen = false">{{ t('misc.cancel') }}</a-button>
        <a-button type="primary" :loading="editorSaving" @click="saveRule">{{ t('misc.save') }}</a-button>
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
  grid-template-columns: repeat(6, minmax(0, 1fr));
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

.ai-sensitive-page__runtime-form {
  display: grid;
  gap: 16px;
}

.ai-sensitive-page__form-section {
  padding: 16px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface-soft);
}

.ai-sensitive-page__section-heading {
  margin-bottom: 12px;
}

.ai-sensitive-page__section-heading h3 {
  margin: 0 0 6px;
  font-size: 15px;
}

.ai-sensitive-page__section-heading p,
.ai-sensitive-page__preset-help {
  margin: 0;
  color: var(--portal-text-soft);
  font-size: 12px;
  line-height: 1.6;
}

.ai-sensitive-page__binding-summary {
  display: grid;
  gap: 10px;
}

.ai-sensitive-page__binding-links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ai-sensitive-page__binding-link {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border: 1px solid var(--portal-border);
  border-radius: 999px;
  color: var(--portal-primary);
  background: rgba(24, 144, 255, 0.08);
}

.ai-sensitive-page__switch-grid,
.ai-sensitive-page__field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.ai-sensitive-page__preset-list {
  display: grid;
  gap: 6px;
  margin-bottom: 12px;
}

.ai-sensitive-page__system-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 12px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.ai-sensitive-page__system-search-result {
  display: grid;
  gap: 12px;
}

.ai-sensitive-page__system-search-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ai-sensitive-page__toolbar,
.ai-sensitive-page__runtime-actions,
.ai-sensitive-page__drawer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.ai-sensitive-page__toolbar {
  justify-content: flex-start;
  margin-bottom: 12px;
}

.ai-sensitive-page__audit-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 14px;
}

.ai-sensitive-page__drawer-actions {
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

@media (max-width: 1279px) {
  .ai-sensitive-page__stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 1023px) {
  .ai-sensitive-page__stats,
  .ai-sensitive-page__switch-grid,
  .ai-sensitive-page__field-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 767px) {
  .ai-sensitive-page__stats,
  .ai-sensitive-page__switch-grid,
  .ai-sensitive-page__field-grid,
  .ai-sensitive-page__audit-bar {
    display: grid;
    grid-template-columns: 1fr;
  }
}
</style>
