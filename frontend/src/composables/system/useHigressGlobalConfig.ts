import { computed, shallowRef } from 'vue';
import type { HigressGlobalConfigFormState } from '@/interfaces/system';
import { showSuccess } from '@/lib/feedback';
import { getAIGatewayConfig, updateAIGatewayConfig } from '@/services/system';
import {
  createDefaultHigressGlobalConfigFormState,
  mergeKnownConfigIntoRoot,
  parseHigressGlobalConfig,
  parseHigressYamlRoot,
  serializeHigressRoot,
  validateHigressConfig,
} from './higress-global-config-utils';

export function useHigressGlobalConfig() {
  const loading = shallowRef(false);
  const saving = shallowRef(false);
  const formState = shallowRef<HigressGlobalConfigFormState>(createDefaultHigressGlobalConfigFormState());
  const rawYaml = shallowRef('');
  const currentRoot = shallowRef<Record<string, any>>({});
  const parseError = shallowRef('');
  const saveError = shallowRef('');
  const validationErrors = shallowRef<string[]>([]);
  const lastSavedRawYaml = shallowRef('');

  const dirty = computed(() => normalizeYaml(rawYaml.value) !== normalizeYaml(lastSavedRawYaml.value));
  const saveDisabled = computed(() => saving.value || !dirty.value || Boolean(parseError.value) || validationErrors.value.length > 0);

  async function load() {
    loading.value = true;
    saveError.value = '';
    try {
      const raw = await getAIGatewayConfig().catch(() => '');
      applyParsedYaml(typeof raw === 'string' && raw.trim() ? raw : createDefaultRawYaml());
    } finally {
      loading.value = false;
    }
  }

  async function save() {
    if (saveDisabled.value) {
      return;
    }
    saving.value = true;
    saveError.value = '';
    try {
      const updated = await updateAIGatewayConfig(rawYaml.value);
      applyParsedYaml(typeof updated === 'string' ? updated : rawYaml.value);
      showSuccess('保存成功');
    } catch (error: any) {
      saveError.value = error?.response?.data?.message || error?.message || '保存失败';
    } finally {
      saving.value = false;
    }
  }

  function updateForm(nextFormState: HigressGlobalConfigFormState) {
    const mergedRoot = mergeKnownConfigIntoRoot(currentRoot.value, nextFormState);
    formState.value = cloneFormState(nextFormState);
    currentRoot.value = mergedRoot;
    rawYaml.value = serializeHigressRoot(mergedRoot);
    parseError.value = '';
    saveError.value = '';
    validationErrors.value = validateHigressConfig(mergedRoot, formState.value);
  }

  function updateRawYaml(nextRawYaml: string) {
    rawYaml.value = nextRawYaml;
    saveError.value = '';

    try {
      const parsedRoot = parseHigressYamlRoot(nextRawYaml);
      const parsed = parseHigressGlobalConfig(nextRawYaml);
      currentRoot.value = parsedRoot;
      formState.value = cloneFormState(parsed.knownConfig);
      parseError.value = '';
      validationErrors.value = validateHigressConfig(parsedRoot, formState.value);
    } catch (error: any) {
      parseError.value = error?.message || 'YAML 解析失败';
      validationErrors.value = [];
    }
  }

  function applyParsedYaml(raw: string) {
    const parsed = parseHigressGlobalConfig(raw);
    currentRoot.value = parseHigressYamlRoot(parsed.rawYaml);
    formState.value = cloneFormState(parsed.knownConfig);
    rawYaml.value = parsed.rawYaml;
    lastSavedRawYaml.value = parsed.rawYaml;
    parseError.value = '';
    saveError.value = '';
    validationErrors.value = validateHigressConfig(currentRoot.value, formState.value);
  }

  return {
    loading,
    saving,
    formState,
    rawYaml,
    parseError,
    saveError,
    validationErrors,
    dirty,
    saveDisabled,
    load,
    save,
    updateForm,
    updateRawYaml,
  };
}

function normalizeYaml(value: string) {
  return value.trim();
}

function cloneFormState(value: HigressGlobalConfigFormState) {
  return JSON.parse(JSON.stringify(value)) as HigressGlobalConfigFormState;
}

function createDefaultRawYaml() {
  const defaults = createDefaultHigressGlobalConfigFormState();
  return serializeHigressRoot(mergeKnownConfigIntoRoot({}, defaults));
}
