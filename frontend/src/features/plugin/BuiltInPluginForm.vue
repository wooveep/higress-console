<script setup lang="ts">
import { watch } from 'vue';
import type { HeaderModifyConfig } from '@/interfaces/route';
import { useI18n } from 'vue-i18n';

type HeaderRow = {
  headerType: 'request' | 'response';
  actionType: 'add' | 'set' | 'remove';
  key: string;
  value?: string;
};

const props = defineProps<{
  pluginName: string;
  targetDetail: Record<string, any> | null;
  state: Record<string, any>;
}>();

const { t } = useI18n();

function normalizeRewriteSource(source: any) {
  return {
    enabled: Boolean(source?.enabled),
    path: source?.path || '',
    host: source?.host || '',
    matchType: props.targetDetail?.path?.matchType || props.targetDetail?.pathPredicate?.matchType || '',
    originPath: props.targetDetail?.path?.matchValue || props.targetDetail?.pathPredicate?.matchValue || '',
    originHost: Array.isArray(props.targetDetail?.domains) ? props.targetDetail?.domains.join(', ') : '',
  };
}

function headerRowsFromConfig(config?: HeaderModifyConfig | any): HeaderRow[] {
  const source = config || {};
  const rows: HeaderRow[] = [];
  ['request', 'response'].forEach((headerType) => {
    const stage = source?.[headerType] || {};
    (stage.add || []).forEach((item: any) => rows.push({ headerType: headerType as HeaderRow['headerType'], actionType: 'add', key: item.key, value: item.value }));
    (stage.set || []).forEach((item: any) => rows.push({ headerType: headerType as HeaderRow['headerType'], actionType: 'set', key: item.key, value: item.value }));
    (stage.remove || stage.delete || []).forEach((item: string) => rows.push({ headerType: headerType as HeaderRow['headerType'], actionType: 'remove', key: item, value: '' }));
  });
  return rows.length ? rows : [{ headerType: 'request', actionType: 'add', key: '', value: '' }];
}

function normalizeHeaderSource(source?: any) {
  return {
    enabled: Boolean(source?.enabled),
    rows: headerRowsFromConfig(source),
  };
}

function normalizeCorsSource(source?: any) {
  return {
    enabled: Boolean(source?.enabled),
    allowOrigins: Array.isArray(source?.allowOrigins) ? source.allowOrigins.join(';') : '*',
    allowMethods: Array.isArray(source?.allowMethods) ? source.allowMethods : ['GET', 'PUT', 'POST', 'HEAD', 'DELETE', 'PATCH', 'OPTIONS'],
    allowHeaders: Array.isArray(source?.allowHeaders) ? source.allowHeaders.join(';') : '*',
    exposeHeaders: Array.isArray(source?.exposeHeaders || source?.exposeHeader) ? (source.exposeHeaders || source.exposeHeader).join(';') : '*',
    allowCredentials: Boolean(source?.allowCredentials),
    maxAge: source?.maxAge ?? source?.maxAgent ?? 86400,
  };
}

function normalizeRetriesSource(source?: any) {
  const retrySource = source || {};
  return {
    enabled: Boolean(retrySource?.enabled),
    attempts: retrySource?.attempts ?? retrySource?.attempt ?? 3,
    conditions: retrySource?.conditions || (retrySource?.retryOn ? String(retrySource.retryOn).split(',') : ['error', 'timeout']),
    timeout: retrySource?.timeout ?? 5,
  };
}

function syncState() {
  if (props.pluginName === 'rewrite') {
    Object.assign(props.state, normalizeRewriteSource(props.targetDetail?.rewrite));
    return;
  }
  if (props.pluginName === 'headerModify') {
    Object.assign(props.state, normalizeHeaderSource(props.targetDetail?.headerModify || props.targetDetail?.headerControl));
    return;
  }
  if (props.pluginName === 'cors') {
    Object.assign(props.state, normalizeCorsSource(props.targetDetail?.cors));
    return;
  }
  if (props.pluginName === 'retries') {
    Object.assign(props.state, normalizeRetriesSource(props.targetDetail?.proxyNextUpstream || props.targetDetail?.retries));
  }
}

watch(() => [props.pluginName, props.targetDetail], syncState, { immediate: true, deep: true });

defineExpose({
  serialize() {
    if (props.pluginName === 'rewrite') {
      return {
        rewrite: {
          enabled: Boolean(props.state.enabled),
          path: props.state.path || '',
          host: props.state.host || '',
        },
      };
    }

    if (props.pluginName === 'headerModify') {
      const headerModify = {
        enabled: Boolean(props.state.enabled),
        request: { add: [] as any[], set: [] as any[], remove: [] as string[] },
        response: { add: [] as any[], set: [] as any[], remove: [] as string[] },
      };
      (Array.isArray(props.state.rows) ? props.state.rows : []).forEach((item) => {
        if (!item.key) {
          return;
        }
        const target = headerModify[item.headerType][item.actionType];
        if (item.actionType === 'remove') {
          target.push(item.key);
          return;
        }
        target.push({
          key: item.key,
          value: item.value || '',
        });
      });
      return {
        headerModify,
        headerControl: headerModify,
      };
    }

    if (props.pluginName === 'cors') {
      return {
        cors: {
          enabled: Boolean(props.state.enabled),
          allowOrigins: String(props.state.allowOrigins || '').split(';').map((item) => item.trim()).filter(Boolean),
          allowMethods: Array.isArray(props.state.allowMethods) ? props.state.allowMethods : [],
          allowHeaders: String(props.state.allowHeaders || '').split(';').map((item) => item.trim()).filter(Boolean),
          exposeHeaders: String(props.state.exposeHeaders || '').split(';').map((item) => item.trim()).filter(Boolean),
          allowCredentials: Boolean(props.state.allowCredentials),
          maxAge: props.state.maxAge || 86400,
        },
      };
    }

    const retries = {
      enabled: Boolean(props.state.enabled),
      attempts: props.state.attempts || 3,
      conditions: Array.isArray(props.state.conditions) ? props.state.conditions : [],
      timeout: props.state.timeout || 5,
    };

    return {
      retries,
      proxyNextUpstream: retries,
    };
  },
});
</script>

<template>
  <div class="built-in-plugin-form">
    <a-form layout="vertical">
      <a-form-item :label="t('plugins.configForm.routeEnableStatus')">
        <a-switch v-model:checked="state.enabled" />
      </a-form-item>
      <a-alert
        type="info"
        show-icon
        :message="t('plugins.configForm.routeEnableHint')"
      />
    </a-form>
  </div>
</template>

<style scoped>
.built-in-plugin-form {
  display: grid;
  gap: 16px;
}
</style>
