<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import SecretMaskText from '@/components/common/SecretMaskText.vue';
import { addLlmProvider, deleteLlmProvider, getLlmProviders, updateLlmProvider } from '@/services/llm-provider';
import { joinLines, safeParseJson, splitLines, stringifyPretty } from '@/lib/portal';
import { showSuccess } from '@/lib/feedback';

const loading = ref(false);
const search = ref('');
const rows = ref<any[]>([]);
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<any>(null);
const deleting = ref<any>(null);

const formState = reactive({
  name: '',
  type: '',
  protocol: 'openai/v1',
  proxyName: '',
  tokensText: '',
  rawConfigsJson: '{}',
});

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.type, item.protocol, item.proxyName].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

const rawConfigsGuide = `常用 rawConfigs 字段：
- providerDomain: 统一覆写上游域名
- providerBasePath: 统一追加上游基础路径，必须以 / 开头
- promoteThinkingOnEmpty: 在 content 为空时提升 reasoning_content
- hiclawMode: 开启后联动思维补齐能力
- bedrockPromptCachePointPositions: Bedrock Prompt Cache 注入位置
- promptCacheRetention: Bedrock 默认 prompt_cache_retention，支持 in_memory / 24h`;

const rawConfigsExamples = `Vertex OAuth
{
  "vertexRegion": "asia-east1",
  "vertexProjectId": "demo-project",
  "vertexAuthKey": "{\\"type\\":\\"service_account\\",\\"client_email\\":\\"demo@example.com\\",\\"private_key_id\\":\\"key-id\\",\\"private_key\\":\\"-----BEGIN PRIVATE KEY-----\\\\n...\\\\n-----END PRIVATE KEY-----\\\\n\\",\\"token_uri\\":\\"https://oauth2.googleapis.com/token\\"}"
}

Vertex Express Mode(API Key)
{
  "vertexRegion": "asia-east1",
  "providerBasePath": "/v1beta1"
}

Claude / Gemini 自定义域名
{
  "providerDomain": "llm-proxy.example.com",
  "providerBasePath": "/anthropic"
}

Bedrock Prompt Cache
{
  "awsRegion": "us-west-2",
  "awsAccessKey": "AKIA...",
  "awsSecretKey": "secret",
  "promptCacheRetention": "in_memory",
  "bedrockPromptCachePointPositions": {
    "systemPrompt": true,
    "lastUserMessage": true
  }
}`;

async function load() {
  loading.value = true;
  try {
    rows.value = await getLlmProviders().catch(() => []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    name: record?.name || '',
    type: record?.type || '',
    protocol: record?.protocol || 'openai/v1',
    proxyName: record?.proxyName || '',
    tokensText: joinLines(record?.tokens),
    rawConfigsJson: stringifyPretty(record?.rawConfigs || {}),
  });
  drawerOpen.value = true;
}

async function submit() {
  const payload = {
    ...(editing.value?.version ? { version: editing.value.version } : {}),
    name: formState.name,
    type: formState.type,
    protocol: formState.protocol,
    proxyName: formState.proxyName || undefined,
    tokens: splitLines(formState.tokensText),
    rawConfigs: safeParseJson(formState.rawConfigsJson, {}),
  };
  if (editing.value) {
    await updateLlmProvider(payload as any);
  } else {
    await addLlmProvider(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteLlmProvider(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="AI 服务提供者管理">
    <ListToolbar v-model:search="search" search-placeholder="搜索名称、类型、协议" create-text="新增 Provider" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 980 }">
      <a-table-column key="type" data-index="type" title="类型" />
      <a-table-column key="name" data-index="name" title="名称" />
      <a-table-column key="protocol" data-index="protocol" title="协议" />
      <a-table-column key="proxyName" data-index="proxyName" title="代理服务" />
      <a-table-column key="tokens" title="Tokens" width="220">
        <template #default="{ record }">
          <div class="provider-page__tokens">
            <SecretMaskText v-for="token in record.tokens || []" :key="token" :value="token" />
            <span v-if="!(record.tokens || []).length">-</span>
          </div>
        </template>
      </a-table-column>
      <a-table-column key="actions" title="操作" width="180">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <a-drawer v-model:open="drawerOpen" width="720" :title="editing ? '编辑 Provider' : '新增 Provider'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="formState.name" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="类型"><a-input v-model:value="formState.type" /></a-form-item>
        <a-form-item label="协议"><a-input v-model:value="formState.protocol" /></a-form-item>
        <a-form-item label="代理服务"><a-input v-model:value="formState.proxyName" /></a-form-item>
        <a-form-item label="Tokens（一行一个）"><a-textarea v-model:value="formState.tokensText" :rows="6" /></a-form-item>
        <a-form-item label="rawConfigs(JSON)">
          <a-textarea v-model:value="formState.rawConfigsJson" :rows="10" spellcheck="false" />
          <a-alert type="info" show-icon style="margin-top: 12px" message="rawConfigs 补充说明" :description="rawConfigsGuide" />
          <div class="provider-page__examples">
            <div class="provider-page__examples-title">示例</div>
            <pre>{{ rawConfigsExamples }}</pre>
          </div>
        </a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>

<style scoped>
.provider-page__tokens {
  display: grid;
  gap: 6px;
}

.provider-page__examples {
  margin-top: 12px;
  padding: 12px 14px;
  border: 1px solid var(--portal-border);
  border-radius: 12px;
  background: var(--portal-surface-soft);
}

.provider-page__examples-title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--portal-text-soft);
}

.provider-page__examples pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.6;
}
</style>
