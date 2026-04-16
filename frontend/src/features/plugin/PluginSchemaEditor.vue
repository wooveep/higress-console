<script setup lang="ts">
import { computed } from 'vue';
import { PlusOutlined, MinusCircleOutlined } from '@ant-design/icons-vue';
import { createSchemaDefaultValue, getLocalizedSchemaText, type SchemaNode } from './plugin-config';

defineOptions({
  name: 'PluginSchemaEditor',
});

const props = defineProps<{
  schema: SchemaNode;
  state: Record<string, any>;
  locale: string;
  level?: number;
  allowCustomFields?: boolean;
}>();

let customFieldCounter = 0;

const fields = computed(() => Object.entries(props.schema.properties || {}).map(([key, node]) => ({
  key,
  node,
  title: getLocalizedSchemaText(node, 'title', props.locale, key),
  description: getLocalizedSchemaText(node, 'description', props.locale, ''),
  required: (props.schema.required || []).includes(key),
})));

const customFieldKeys = computed(() => {
  if (!props.allowCustomFields || (props.level || 0) !== 0) {
    return [];
  }
  const definedKeys = new Set(fields.value.map((item) => item.key));
  return Object.keys(props.state || {}).filter((key) => !definedKeys.has(key));
});

function ensureObject(key: string) {
  if (!props.state[key] || typeof props.state[key] !== 'object' || Array.isArray(props.state[key])) {
    props.state[key] = {};
  }
  return props.state[key];
}

function ensureArray(key: string) {
  if (!Array.isArray(props.state[key])) {
    props.state[key] = [];
  }
  return props.state[key] as any[];
}

function addArrayItem(key: string, itemSchema?: SchemaNode) {
  ensureArray(key).push(createSchemaDefaultValue(itemSchema));
}

function removeArrayItem(key: string, index: number) {
  ensureArray(key).splice(index, 1);
}

function getPrimitiveArrayItemValue(item: any) {
  if (typeof item === 'boolean') {
    return item;
  }
  if (typeof item === 'number') {
    return item;
  }
  return item ?? '';
}

function updatePrimitiveArrayItem(key: string, index: number, value: any) {
  ensureArray(key)[index] = value;
}

function addCustomField() {
  customFieldCounter += 1;
  props.state[`custom_key_${customFieldCounter}`] = '';
}

function renameCustomField(previousKey: string, nextKey: string) {
  const trimmed = String(nextKey || '').trim();
  if (!trimmed || trimmed === previousKey) {
    return;
  }
  if (Object.prototype.hasOwnProperty.call(props.state, trimmed)) {
    return;
  }
  props.state[trimmed] = props.state[previousKey];
  delete props.state[previousKey];
}

function updateCustomFieldValue(key: string, value: any) {
  props.state[key] = value;
}

function removeCustomField(key: string) {
  delete props.state[key];
}
</script>

<template>
  <div class="plugin-schema-editor" :data-level="level || 0">
    <section
      v-if="allowCustomFields && (level || 0) === 0"
      class="plugin-schema-editor__group"
    >
      <header class="plugin-schema-editor__group-header">
        <div>
          <div class="plugin-schema-editor__label">
            <span>自定义参数</span>
          </div>
          <p class="plugin-schema-editor__desc">用于补充 Schema 之外的扩展配置，左侧参数名，右侧参数值。</p>
        </div>
        <a-button type="link" size="small" @click="addCustomField">
          <template #icon><PlusOutlined /></template>
          添加
        </a-button>
      </header>

      <div v-if="customFieldKeys.length" class="plugin-schema-editor__array">
        <div
          v-for="key in customFieldKeys"
          :key="key"
          class="plugin-schema-editor__custom-row"
        >
          <a-input
            :value="key"
            placeholder="参数名"
            @change="(event) => renameCustomField(key, event?.target?.value)"
          />
          <a-input
            :value="state[key]"
            placeholder="参数值"
            @update:value="(value) => updateCustomFieldValue(key, value)"
          />
          <a-button type="text" size="small" danger @click="removeCustomField(key)">
            <template #icon><MinusCircleOutlined /></template>
          </a-button>
        </div>
      </div>
    </section>

    <div v-for="field in fields" :key="field.key" class="plugin-schema-editor__field">
      <template v-if="field.node.type === 'object'">
        <section class="plugin-schema-editor__group">
          <header class="plugin-schema-editor__group-header">
            <div>
              <div class="plugin-schema-editor__label">
                <span>{{ field.title }}</span>
                <span v-if="field.required" class="plugin-schema-editor__required">*</span>
              </div>
              <p v-if="field.description" class="plugin-schema-editor__desc">{{ field.description }}</p>
            </div>
          </header>
          <PluginSchemaEditor
            :schema="field.node"
            :state="ensureObject(field.key)"
            :locale="locale"
            :level="(level || 0) + 1"
            :allow-custom-fields="false"
          />
        </section>
      </template>

      <template v-else-if="field.node.type === 'array'">
        <section class="plugin-schema-editor__group">
          <header class="plugin-schema-editor__group-header">
            <div>
              <div class="plugin-schema-editor__label">
                <span>{{ field.title }}</span>
                <span v-if="field.required" class="plugin-schema-editor__required">*</span>
              </div>
              <p v-if="field.description" class="plugin-schema-editor__desc">{{ field.description }}</p>
            </div>
            <a-button type="link" size="small" @click="addArrayItem(field.key, field.node.items)">
              <template #icon><PlusOutlined /></template>
              添加
            </a-button>
          </header>

          <div v-if="field.node.items?.type === 'object'" class="plugin-schema-editor__array">
            <article
              v-for="(item, index) in ensureArray(field.key)"
              :key="`${field.key}-${index}`"
              class="plugin-schema-editor__array-card"
            >
              <div class="plugin-schema-editor__array-header">
                <span>#{{
                  index + 1
                }}</span>
                <a-button type="text" size="small" danger @click="removeArrayItem(field.key, index)">
                  <template #icon><MinusCircleOutlined /></template>
                </a-button>
              </div>
              <PluginSchemaEditor
                :schema="field.node.items || { type: 'object', properties: {} }"
                :state="item"
                :locale="locale"
                :level="(level || 0) + 1"
                :allow-custom-fields="false"
              />
            </article>
          </div>

          <div v-else class="plugin-schema-editor__array">
            <div
              v-for="(item, index) in ensureArray(field.key)"
              :key="`${field.key}-${index}`"
              class="plugin-schema-editor__array-row"
            >
              <a-select
                v-if="Array.isArray(field.node.items?.enum) && field.node.items?.enum?.length"
                :value="item"
                style="width: 100%"
                @update:value="(value) => updatePrimitiveArrayItem(field.key, index, value)"
              >
                <a-select-option
                  v-for="option in field.node.items?.enum || []"
                  :key="String(option)"
                  :value="option"
                >
                  {{ option }}
                </a-select-option>
              </a-select>
              <a-switch
                v-else-if="field.node.items?.type === 'boolean'"
                :checked="Boolean(item)"
                @update:checked="(value) => updatePrimitiveArrayItem(field.key, index, value)"
              />
              <a-input-number
                v-else-if="field.node.items?.type === 'integer' || field.node.items?.type === 'number'"
                :value="Number.isFinite(item) ? item : undefined"
                style="width: 100%"
                @update:value="(value) => updatePrimitiveArrayItem(field.key, index, value)"
              />
              <a-input
                v-else
                :value="getPrimitiveArrayItemValue(item)"
                @update:value="(value) => updatePrimitiveArrayItem(field.key, index, value)"
              />
              <a-button type="text" size="small" danger @click="removeArrayItem(field.key, index)">
                <template #icon><MinusCircleOutlined /></template>
              </a-button>
            </div>
          </div>
        </section>
      </template>

      <template v-else>
        <div class="plugin-schema-editor__control">
          <div class="plugin-schema-editor__label">
            <span>{{ field.title }}</span>
            <span v-if="field.required" class="plugin-schema-editor__required">*</span>
          </div>
          <p v-if="field.description" class="plugin-schema-editor__desc">{{ field.description }}</p>

          <a-select
            v-if="Array.isArray(field.node.enum) && field.node.enum.length"
            v-model:value="state[field.key]"
            allow-clear
          >
            <a-select-option v-for="option in field.node.enum" :key="String(option)" :value="option">
              {{ option }}
            </a-select-option>
          </a-select>
          <a-switch v-else-if="field.node.type === 'boolean'" v-model:checked="state[field.key]" />
          <a-input-number
            v-else-if="field.node.type === 'integer' || field.node.type === 'number'"
            v-model:value="state[field.key]"
            style="width: 100%"
          />
          <a-textarea
            v-else-if="field.node.format === 'textarea' || (field.description && field.description.length > 80)"
            v-model:value="state[field.key]"
            :rows="4"
          />
          <a-input v-else v-model:value="state[field.key]" />
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.plugin-schema-editor {
  display: grid;
  gap: 14px;
}

.plugin-schema-editor__field,
.plugin-schema-editor__group,
.plugin-schema-editor__array-card {
  min-width: 0;
}

.plugin-schema-editor__control,
.plugin-schema-editor__group,
.plugin-schema-editor__array-card {
  padding: 14px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface);
}

.plugin-schema-editor__group-header,
.plugin-schema-editor__array-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.plugin-schema-editor__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--portal-text);
}

.plugin-schema-editor__required {
  color: #ff4d4f;
}

.plugin-schema-editor__desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--portal-text-soft);
}

.plugin-schema-editor__array {
  display: grid;
  gap: 10px;
}

.plugin-schema-editor__array-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.plugin-schema-editor__custom-row {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr) auto;
  align-items: center;
  gap: 8px;
}
</style>
