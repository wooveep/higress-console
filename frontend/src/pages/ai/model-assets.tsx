import { LlmProvider } from '@/interfaces/llm-provider';
import {
  ModelAsset,
  ModelAssetBinding,
  ModelAssetOptions,
  ModelBindingPriceVersion,
  ModelBindingPricing,
  MODEL_ASSET_PRESET_TAGS,
  ProviderModelOption,
} from '@/interfaces/model-asset';
import { AssetGrantRecord, OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import { getLlmProviders } from '@/services/llm-provider';
import {
  createModelAsset,
  createModelBinding,
  getModelAssets,
  getModelAssetOptions,
  getModelBindingPriceVersions,
  restoreModelBindingPriceVersion,
  publishModelBinding,
  unpublishModelBinding,
  updateModelAsset,
  updateModelBinding,
} from '@/services/model-asset';
import {
  listAssetGrants,
  listOrgAccounts,
  listOrgDepartmentsTree,
  replaceAssetGrants,
} from '@/services/organization';
import { PlusOutlined, RedoOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { useRequest } from 'ahooks';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { USER_LEVELS } from '@/utils/consumer-level';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;
const MODEL_BINDING_ASSET_TYPE = 'model_binding';

type AssetFormValues = {
  assetId: string;
  canonicalName?: string;
  displayName?: string;
  intro?: string;
  tags?: string[];
  modalities?: string[];
  features?: string[];
  requestKinds?: string[];
};

type BindingFormValues = {
  bindingId: string;
  modelId?: string;
  providerName?: string;
  targetModel?: string;
  protocol?: string;
  endpoint?: string;
  rpm?: number;
  tpm?: number;
  contextWindow?: number;
  currency?: string;
  supportsPromptCaching?: boolean;
  inputCostPerToken?: number;
  outputCostPerToken?: number;
  inputCostPerRequest?: number;
  cacheCreationInputTokenCost?: number;
  cacheCreationInputTokenCostAbove1hr?: number;
  cacheReadInputTokenCost?: number;
  inputCostPerTokenAbove200kTokens?: number;
  outputCostPerTokenAbove200kTokens?: number;
  cacheCreationInputTokenCostAbove200kTokens?: number;
  cacheReadInputTokenCostAbove200kTokens?: number;
  outputCostPerImage?: number;
  outputCostPerImageToken?: number;
  inputCostPerImage?: number;
  inputCostPerImageToken?: number;
};

const pricingFieldGroups: Array<{
  title: string;
  fields: Array<{ name: keyof BindingFormValues; label: string; step?: number }>;
}> = [
  {
    title: '基础价',
    fields: [
      { name: 'inputCostPerToken', label: '输入 Token 单价', step: 0.000001 },
      { name: 'outputCostPerToken', label: '输出 Token 单价', step: 0.000001 },
      { name: 'inputCostPerRequest', label: '按请求计价', step: 0.000001 },
      { name: 'cacheCreationInputTokenCost', label: 'Cache 写入 Token 单价', step: 0.000001 },
      { name: 'cacheCreationInputTokenCostAbove1hr', label: 'Cache 写入 Token 单价（>1h）', step: 0.000001 },
      { name: 'cacheReadInputTokenCost', label: 'Cache 读取 Token 单价', step: 0.000001 },
      { name: 'outputCostPerImage', label: '输出图片单价', step: 0.000001 },
      { name: 'outputCostPerImageToken', label: '输出图片 Token 单价', step: 0.000001 },
      { name: 'inputCostPerImage', label: '输入图片单价', step: 0.000001 },
      { name: 'inputCostPerImageToken', label: '输入图片 Token 单价', step: 0.000001 },
    ],
  },
  {
    title: 'above_200k',
    fields: [
      { name: 'inputCostPerTokenAbove200kTokens', label: '输入 Token 单价（>200k）', step: 0.000001 },
      { name: 'outputCostPerTokenAbove200kTokens', label: '输出 Token 单价（>200k）', step: 0.000001 },
      {
        name: 'cacheCreationInputTokenCostAbove200kTokens',
        label: 'Cache 写入 Token 单价（>200k）',
        step: 0.000001,
      },
      {
        name: 'cacheReadInputTokenCostAbove200kTokens',
        label: 'Cache 读取 Token 单价（>200k）',
        step: 0.000001,
      },
    ],
  },
];

const statusColorMap: Record<string, string> = {
  draft: 'default',
  published: 'green',
  unpublished: 'orange',
  active: 'green',
  inactive: 'default',
  disabled: 'red',
};

function buildPricing(values: BindingFormValues): ModelBindingPricing {
  const pricing: ModelBindingPricing = {
    currency: values.currency || 'CNY',
    supportsPromptCaching: values.supportsPromptCaching,
  };
  const numericFields: Array<keyof BindingFormValues> = [
    'inputCostPerToken',
    'outputCostPerToken',
    'inputCostPerRequest',
    'cacheCreationInputTokenCost',
    'cacheCreationInputTokenCostAbove1hr',
    'cacheReadInputTokenCost',
    'inputCostPerTokenAbove200kTokens',
    'outputCostPerTokenAbove200kTokens',
    'cacheCreationInputTokenCostAbove200kTokens',
    'cacheReadInputTokenCostAbove200kTokens',
    'outputCostPerImage',
    'outputCostPerImageToken',
    'inputCostPerImage',
    'inputCostPerImageToken',
  ];
  numericFields.forEach((field) => {
    const value = values[field];
    if (typeof value === 'number') {
      (pricing as any)[field] = value;
    }
  });
  return pricing;
}

function describePricing(pricing?: ModelBindingPricing) {
  if (!pricing) {
    return '-';
  }
  const items: string[] = [];
  if (typeof pricing.inputCostPerToken === 'number') {
    items.push(`输入 ${pricing.inputCostPerToken}`);
  }
  if (typeof pricing.outputCostPerToken === 'number') {
    items.push(`输出 ${pricing.outputCostPerToken}`);
  }
  if (typeof pricing.inputCostPerRequest === 'number') {
    items.push(`请求 ${pricing.inputCostPerRequest}`);
  }
  if (typeof pricing.inputCostPerTokenAbove200kTokens === 'number') {
    items.push(`输入>200k ${pricing.inputCostPerTokenAbove200kTokens}`);
  }
  if (typeof pricing.outputCostPerTokenAbove200kTokens === 'number') {
    items.push(`输出>200k ${pricing.outputCostPerTokenAbove200kTokens}`);
  }
  if (pricing.supportsPromptCaching) {
    items.push('支持缓存');
  }
  return items.length ? items.join(' / ') : '-';
}

function toAssetFormValues(asset?: ModelAsset): AssetFormValues {
  return {
    assetId: asset?.assetId || '',
    canonicalName: asset?.canonicalName,
    displayName: asset?.displayName,
    intro: asset?.intro,
    tags: asset?.tags || [],
    modalities: asset?.capabilities?.modalities || [],
    features: asset?.capabilities?.features || [],
    requestKinds: asset?.capabilities?.requestKinds || [],
  };
}

function toBindingFormValues(binding?: ModelAssetBinding): BindingFormValues {
  const pricing = binding?.pricing || {};
  return {
    bindingId: binding?.bindingId || '',
    modelId: binding?.modelId,
    providerName: binding?.providerName,
    targetModel: binding?.targetModel,
    protocol: binding?.protocol || 'openai/v1',
    endpoint: binding?.endpoint,
    rpm: binding?.limits?.rpm,
    tpm: binding?.limits?.tpm,
    contextWindow: binding?.limits?.contextWindow,
    currency: pricing.currency || 'CNY',
    supportsPromptCaching: pricing.supportsPromptCaching,
    inputCostPerToken: pricing.inputCostPerToken,
    outputCostPerToken: pricing.outputCostPerToken,
    inputCostPerRequest: pricing.inputCostPerRequest,
    cacheCreationInputTokenCost: pricing.cacheCreationInputTokenCost,
    cacheCreationInputTokenCostAbove1hr: pricing.cacheCreationInputTokenCostAbove1hr,
    cacheReadInputTokenCost: pricing.cacheReadInputTokenCost,
    inputCostPerTokenAbove200kTokens: pricing.inputCostPerTokenAbove200kTokens,
    outputCostPerTokenAbove200kTokens: pricing.outputCostPerTokenAbove200kTokens,
    cacheCreationInputTokenCostAbove200kTokens: pricing.cacheCreationInputTokenCostAbove200kTokens,
    cacheReadInputTokenCostAbove200kTokens: pricing.cacheReadInputTokenCostAbove200kTokens,
    outputCostPerImage: pricing.outputCostPerImage,
    outputCostPerImageToken: pricing.outputCostPerImageToken,
    inputCostPerImage: pricing.inputCostPerImage,
    inputCostPerImageToken: pricing.inputCostPerImageToken,
  };
}

function flattenDepartmentOptions(nodes: OrgDepartmentNode[], level = 0): Array<{ label: string; value: string }> {
  return (nodes || []).flatMap((item) => {
    const prefix = level > 0 ? `${'  '.repeat(level)}- ` : '';
    return [
      { label: `${prefix}${item.name}`, value: item.departmentId },
      ...flattenDepartmentOptions(item.children || [], level + 1),
    ];
  });
}

const ModelAssetsPage: React.FC = () => {
  const [assets, setAssets] = useState<ModelAsset[]>([]);
  const [providers, setProviders] = useState<LlmProvider[]>([]);
  const [assetOptions, setAssetOptions] = useState<ModelAssetOptions>({
    capabilities: { modalities: [], features: [], requestKinds: [] },
    providerModels: [],
  });
  const [selectedAssetId, setSelectedAssetId] = useState<string>();
  const [assetDrawerOpen, setAssetDrawerOpen] = useState(false);
  const [bindingDrawerOpen, setBindingDrawerOpen] = useState(false);
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [grantDrawerOpen, setGrantDrawerOpen] = useState(false);
  const [editingAsset, setEditingAsset] = useState<ModelAsset>();
  const [editingBinding, setEditingBinding] = useState<ModelAssetBinding>();
  const [historyBinding, setHistoryBinding] = useState<ModelAssetBinding>();
  const [grantBinding, setGrantBinding] = useState<ModelAssetBinding>();
  const [priceVersions, setPriceVersions] = useState<ModelBindingPriceVersion[]>([]);
  const [priceVersionsLoading, setPriceVersionsLoading] = useState(false);
  const [orgAccounts, setOrgAccounts] = useState<OrgAccountRecord[]>([]);
  const [departmentTree, setDepartmentTree] = useState<OrgDepartmentNode[]>([]);
  const [grantConsumers, setGrantConsumers] = useState<string[]>([]);
  const [grantDepartments, setGrantDepartments] = useState<string[]>([]);
  const [grantUserLevels, setGrantUserLevels] = useState<string[]>([]);
  const [grantLoading, setGrantLoading] = useState(false);
  const [grantSaving, setGrantSaving] = useState(false);
  const [assetForm] = Form.useForm<AssetFormValues>();
  const [bindingForm] = Form.useForm<BindingFormValues>();

  const selectedAsset = useMemo(
    () => assets.find((item) => item.assetId === selectedAssetId),
    [assets, selectedAssetId],
  );
  const selectedBindings = selectedAsset?.bindings || [];
  const activePriceVersion = useMemo(
    () => priceVersions.find((item) => item.active || item.status === 'active'),
    [priceVersions],
  );

  const { loading: assetsLoading, run: runAssets } = useRequest(getModelAssets, {
    manual: true,
    onSuccess: (result) => {
      setAssets(result || []);
      setSelectedAssetId((current) => {
        if (current && result?.some((item) => item.assetId === current)) {
          return current;
        }
        return result?.[0]?.assetId;
      });
    },
  });

  const { loading: providersLoading, run: runProviders } = useRequest(getLlmProviders, {
    manual: true,
    onSuccess: (result) => {
      setProviders(result || []);
    },
  });

  const { run: runAssetOptions } = useRequest(getModelAssetOptions, {
    manual: true,
    onSuccess: (result) => {
      setAssetOptions(
        result || {
          capabilities: { modalities: [], features: [], requestKinds: [] },
          providerModels: [],
        },
      );
    },
  });

  useEffect(() => {
    runAssets();
    runProviders();
    runAssetOptions();
    listOrgAccounts()
      .then((result) => setOrgAccounts(result || []))
      .catch(() => undefined);
    listOrgDepartmentsTree()
      .then((result) => setDepartmentTree(result || []))
      .catch(() => undefined);
  }, []);

  const refreshAssets = async () => {
    await runAssets();
  };

  const refreshPriceVersions = async (assetId?: string, bindingId?: string) => {
    if (!assetId || !bindingId) {
      setPriceVersions([]);
      return;
    }
    setPriceVersionsLoading(true);
    try {
      const result = await getModelBindingPriceVersions(assetId, bindingId);
      setPriceVersions(result || []);
    } finally {
      setPriceVersionsLoading(false);
    }
  };

  const openCreateAssetDrawer = () => {
    setEditingAsset(undefined);
    assetForm.setFieldsValue(toAssetFormValues());
    setAssetDrawerOpen(true);
  };

  const openEditAssetDrawer = (asset: ModelAsset) => {
    setEditingAsset(asset);
    assetForm.setFieldsValue(toAssetFormValues(asset));
    setAssetDrawerOpen(true);
  };

  const saveAsset = async () => {
    const values = await assetForm.validateFields();
    const payload: ModelAsset = {
      assetId: values.assetId,
      canonicalName: values.canonicalName,
      displayName: values.displayName,
      intro: values.intro,
      tags: values.tags || [],
      capabilities: {
        modalities: values.modalities || [],
        features: values.features || [],
        requestKinds: values.requestKinds || [],
      },
    };
    if (editingAsset) {
      await updateModelAsset(editingAsset.assetId, payload);
      message.success('模型资产已更新');
    } else {
      await createModelAsset(payload);
      message.success('模型资产已创建');
    }
    setAssetDrawerOpen(false);
    assetForm.resetFields();
    await refreshAssets();
  };

  const openCreateBindingDrawer = async () => {
    if (!selectedAsset) {
      message.warning('请先选择一个模型资产');
      return;
    }
    setEditingBinding(undefined);
    bindingForm.setFieldsValue(toBindingFormValues());
    setPriceVersions([]);
    setBindingDrawerOpen(true);
  };

  const openEditBindingDrawer = async (binding: ModelAssetBinding) => {
    setEditingBinding(binding);
    bindingForm.setFieldsValue(toBindingFormValues(binding));
    setBindingDrawerOpen(true);
    await refreshPriceVersions(binding.assetId || selectedAssetId, binding.bindingId);
  };

  const saveBinding = async () => {
    if (!selectedAssetId) {
      return;
    }
    const values = await bindingForm.validateFields();
    const payload: ModelAssetBinding = {
      bindingId: values.bindingId,
      modelId: values.modelId,
      providerName: values.providerName,
      targetModel: values.targetModel,
      protocol: values.protocol,
      endpoint: values.endpoint,
      pricing: buildPricing(values),
      limits: {
        rpm: values.rpm,
        tpm: values.tpm,
        contextWindow: values.contextWindow,
      },
    };
    if (editingBinding) {
      await updateModelBinding(selectedAssetId, editingBinding.bindingId, payload);
      message.success('发布绑定已更新');
    } else {
      await createModelBinding(selectedAssetId, payload);
      message.success('发布绑定已创建');
    }
    setBindingDrawerOpen(false);
    bindingForm.resetFields();
    setEditingBinding(undefined);
    setPriceVersions([]);
    await refreshAssets();
  };

  const handlePublish = async (binding: ModelAssetBinding) => {
    const assetId = binding.assetId || selectedAssetId;
    if (!assetId) {
      return;
    }
    await publishModelBinding(assetId, binding.bindingId);
    message.success('绑定已发布');
    await refreshAssets();
    if (editingBinding?.bindingId === binding.bindingId) {
      await refreshPriceVersions(assetId, binding.bindingId);
    }
  };

  const handleUnpublish = async (binding: ModelAssetBinding) => {
    const assetId = binding.assetId || selectedAssetId;
    if (!assetId) {
      return;
    }
    await unpublishModelBinding(assetId, binding.bindingId);
    message.success('绑定已下架');
    await refreshAssets();
    if (editingBinding?.bindingId === binding.bindingId) {
      await refreshPriceVersions(assetId, binding.bindingId);
    }
  };

  const openHistoryDrawer = async (binding: ModelAssetBinding) => {
    setHistoryBinding(binding);
    setHistoryDrawerOpen(true);
    await refreshPriceVersions(binding.assetId || selectedAssetId, binding.bindingId);
  };

  const openGrantDrawer = async (binding: ModelAssetBinding) => {
    setGrantBinding(binding);
    setGrantDrawerOpen(true);
    setGrantLoading(true);
    try {
      const grants = await listAssetGrants(MODEL_BINDING_ASSET_TYPE, binding.bindingId);
      setGrantConsumers(
        (grants || []).filter((item) => item.subjectType === 'consumer').map((item) => item.subjectId || ''),
      );
      setGrantDepartments(
        (grants || []).filter((item) => item.subjectType === 'department').map((item) => item.subjectId || ''),
      );
      setGrantUserLevels(
        (grants || []).filter((item) => item.subjectType === 'user_level').map((item) => item.subjectId || ''),
      );
    } finally {
      setGrantLoading(false);
    }
  };

  const saveGrantAssignments = async () => {
    if (!grantBinding) {
      return;
    }
    setGrantSaving(true);
    try {
      const grants: AssetGrantRecord[] = [
        ...grantConsumers.map((subjectId) => ({
          assetType: MODEL_BINDING_ASSET_TYPE,
          assetId: grantBinding.bindingId,
          subjectType: 'consumer',
          subjectId,
        })),
        ...grantDepartments.map((subjectId) => ({
          assetType: MODEL_BINDING_ASSET_TYPE,
          assetId: grantBinding.bindingId,
          subjectType: 'department',
          subjectId,
        })),
        ...grantUserLevels.map((subjectId) => ({
          assetType: MODEL_BINDING_ASSET_TYPE,
          assetId: grantBinding.bindingId,
          subjectType: 'user_level',
          subjectId,
        })),
      ];
      await replaceAssetGrants(MODEL_BINDING_ASSET_TYPE, grantBinding.bindingId, grants);
      message.success('绑定授权已更新');
      setGrantDrawerOpen(false);
    } finally {
      setGrantSaving(false);
    }
  };

  const departmentOptions = useMemo(() => flattenDepartmentOptions(departmentTree), [departmentTree]);
  const consumerOptions = useMemo(
    () =>
      (orgAccounts || []).map((item) => ({
        label: item.displayName ? `${item.displayName} (${item.consumerName})` : item.consumerName,
        value: item.consumerName,
      })),
    [orgAccounts],
  );
  const userLevelOptions = useMemo(
    () => USER_LEVELS.map((level) => ({ label: `等级 ${level}`, value: level })),
    [],
  );
  const editingAssetHasLegacyTags = useMemo(
    () => !!editingAsset?.tags?.some((tag) => !MODEL_ASSET_PRESET_TAGS.includes(tag as any)),
    [editingAsset],
  );
  const capabilityValueSets = useMemo(
    () => ({
      modalities: new Set(assetOptions.capabilities?.modalities || []),
      features: new Set(assetOptions.capabilities?.features || []),
      requestKinds: new Set(assetOptions.capabilities?.requestKinds || []),
    }),
    [assetOptions],
  );
  const editingAssetHasLegacyCapabilities = useMemo(() => {
    if (!editingAsset) {
      return false;
    }
    const modalities = editingAsset.capabilities?.modalities || [];
    const features = editingAsset.capabilities?.features || [];
    const requestKinds = editingAsset.capabilities?.requestKinds || [];
    return (
      modalities.some((item) => !capabilityValueSets.modalities.has(item))
      || features.some((item) => !capabilityValueSets.features.has(item))
      || requestKinds.some((item) => !capabilityValueSets.requestKinds.has(item))
    );
  }, [capabilityValueSets, editingAsset]);
  const providerModelCatalog = useMemo(
    () =>
      (assetOptions.providerModels || []).reduce<Record<string, ProviderModelOption[]>>((accumulator, item) => {
        accumulator[item.providerName] = item.models || [];
        return accumulator;
      }, {}),
    [assetOptions],
  );
  const watchedProviderName = Form.useWatch('providerName', bindingForm) as string | undefined;
  const watchedModelId = Form.useWatch('modelId', bindingForm) as string | undefined;
  const watchedTargetModel = Form.useWatch('targetModel', bindingForm) as string | undefined;
  const currentProviderModels = useMemo(
    () => (watchedProviderName ? providerModelCatalog[watchedProviderName] || [] : []),
    [providerModelCatalog, watchedProviderName],
  );
  const currentProviderUsesCatalog = currentProviderModels.length > 0;
  const currentModelIdOptions = useMemo(() => {
    const options = currentProviderModels.map((item) => ({
      label: item.modelId === item.targetModel ? item.modelId : `${item.modelId} / ${item.targetModel}`,
      value: item.modelId,
    }));
    if (watchedModelId && !currentProviderModels.some((item) => item.modelId === watchedModelId)) {
      options.unshift({ label: `历史值 / ${watchedModelId}`, value: watchedModelId });
    }
    return options;
  }, [currentProviderModels, watchedModelId]);
  const currentTargetModelOptions = useMemo(() => {
    const options = currentProviderModels.map((item) => ({
      label: item.targetModel === item.modelId ? item.targetModel : `${item.targetModel} / ${item.modelId}`,
      value: item.targetModel,
    }));
    if (watchedTargetModel && !currentProviderModels.some((item) => item.targetModel === watchedTargetModel)) {
      options.unshift({ label: `历史值 / ${watchedTargetModel}`, value: watchedTargetModel });
    }
    return options;
  }, [currentProviderModels, watchedTargetModel]);
  const bindingHasLegacyCatalogValue = useMemo(
    () =>
      currentProviderUsesCatalog
      && ((!!watchedModelId && !currentProviderModels.some((item) => item.modelId === watchedModelId))
        || (!!watchedTargetModel && !currentProviderModels.some((item) => item.targetModel === watchedTargetModel))),
    [currentProviderModels, currentProviderUsesCatalog, watchedModelId, watchedTargetModel],
  );

  const syncBindingModelPair = (field: 'modelId' | 'targetModel', selectedValue?: string) => {
    if (!currentProviderUsesCatalog) {
      return;
    }
    const matched = currentProviderModels.find((item) => {
      return field === 'modelId'
        ? item.modelId === selectedValue
        : item.targetModel === selectedValue;
    });
    if (matched) {
      bindingForm.setFieldsValue({
        modelId: matched.modelId,
        targetModel: matched.targetModel,
      });
    }
  };

  const handleRestoreVersion = (version: ModelBindingPriceVersion) => {
    if (!historyBinding) {
      return;
    }
    Modal.confirm({
      title: `恢复价格版本 #${version.versionId}`,
      content: '该操作只会把历史价格回填到当前绑定草稿，不会自动发布。',
      okText: '恢复到草稿',
      cancelText: '取消',
      onOk: async () => {
        const assetId = historyBinding.assetId || selectedAssetId;
        if (!assetId) {
          return;
        }
        await restoreModelBindingPriceVersion(assetId, historyBinding.bindingId, version.versionId);
        message.success('历史版本已恢复到草稿');
        await refreshAssets();
        await refreshPriceVersions(assetId, historyBinding.bindingId);
        if (editingBinding?.bindingId === historyBinding.bindingId) {
          const refreshedAsset = (await getModelAssets()).find((item) => item.assetId === assetId);
          const refreshedBinding = refreshedAsset?.bindings?.find((item) => item.bindingId === historyBinding.bindingId);
          if (refreshedBinding) {
            setEditingBinding(refreshedBinding);
            bindingForm.setFieldsValue(toBindingFormValues(refreshedBinding));
          }
        }
      },
    });
  };

  const assetColumns = [
    {
      title: '展示名',
      dataIndex: 'displayName',
      key: 'displayName',
      render: (_: unknown, record: ModelAsset) => record.displayName || record.canonicalName || record.assetId,
    },
    {
      title: '规范名',
      dataIndex: 'canonicalName',
      key: 'canonicalName',
      render: (value: string) => value || '-',
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => {
        if (!tags?.length) {
          return '-';
        }
        return (
          <Space size={[0, 6]} wrap>
            {tags.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '绑定数',
      dataIndex: 'bindings',
      key: 'bindings',
      width: 90,
      render: (bindings: ModelAssetBinding[]) => bindings?.length || 0,
    },
  ];

  const bindingColumns = [
    {
      title: '模型 ID',
      dataIndex: 'modelId',
      key: 'modelId',
      render: (value: string) => value || '-',
    },
    {
      title: 'Provider',
      dataIndex: 'providerName',
      key: 'providerName',
      render: (value: string) => value || '-',
    },
    {
      title: '目标模型',
      dataIndex: 'targetModel',
      key: 'targetModel',
      render: (value: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{value || 'draft'}</Tag>,
    },
    {
      title: '当前草稿价格',
      dataIndex: 'pricing',
      key: 'pricing',
      render: (value: ModelBindingPricing) => <Text>{describePricing(value)}</Text>,
    },
    {
      title: '操作',
      key: 'action',
      width: 230,
      render: (_: unknown, record: ModelAssetBinding) => (
        <Space size="small" wrap>
          <a onClick={() => openEditBindingDrawer(record)}>编辑</a>
          {record.status === 'published' ? (
            <a onClick={() => handleUnpublish(record)}>下架</a>
          ) : (
            <a onClick={() => handlePublish(record)}>发布</a>
          )}
          <a onClick={() => openGrantDrawer(record)}>授权</a>
          <a onClick={() => openHistoryDrawer(record)}>价格历史</a>
        </Space>
      ),
    },
  ];

  const historyColumns = [
    {
      title: '版本',
      dataIndex: 'versionId',
      key: 'versionId',
      width: 100,
      render: (value: number) => `#${value}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: string, record: ModelBindingPriceVersion) => (
        <Space size="small">
          <Tag color={statusColorMap[value] || 'default'}>{value || '-'}</Tag>
          {record.active ? <Tag color="green">current</Tag> : null}
        </Space>
      ),
    },
    {
      title: '价格摘要',
      dataIndex: 'pricing',
      key: 'pricing',
      render: (value: ModelBindingPricing) => describePricing(value),
    },
    {
      title: '生效时间',
      dataIndex: 'effectiveFrom',
      key: 'effectiveFrom',
      width: 180,
      render: (value: string) => value || '-',
    },
    {
      title: '失效时间',
      dataIndex: 'effectiveTo',
      key: 'effectiveTo',
      width: 180,
      render: (value: string) => value || '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: ModelBindingPriceVersion) => (
        <a onClick={() => handleRestoreVersion(record)}>恢复到草稿</a>
      ),
    },
  ];

  return (
    <PageContainer>
      <Row gutter={16}>
        <Col span={10}>
          <Card
            title="模型资产"
            extra={
              <Space>
                <Button icon={<RedoOutlined />} onClick={refreshAssets} loading={assetsLoading} />
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAssetDrawer}>
                  新建资产
                </Button>
              </Space>
            }
          >
            <Table<ModelAsset>
              rowKey="assetId"
              loading={assetsLoading}
              dataSource={assets}
              columns={assetColumns}
              pagination={false}
              rowClassName={(record) => (record.assetId === selectedAssetId ? 'ant-table-row-selected' : '')}
              onRow={(record) => ({
                onClick: () => setSelectedAssetId(record.assetId),
                onDoubleClick: () => openEditAssetDrawer(record),
              })}
            />
          </Card>
        </Col>
        <Col span={14}>
          <Card
            title={selectedAsset ? `发布绑定 · ${selectedAsset.displayName || selectedAsset.assetId}` : '发布绑定'}
            extra={
              <Space>
                <Button disabled={!selectedAsset} onClick={() => selectedAsset && openEditAssetDrawer(selectedAsset)}>
                  编辑资产
                </Button>
                <Button type="primary" icon={<PlusOutlined />} disabled={!selectedAsset} onClick={openCreateBindingDrawer}>
                  新建绑定
                </Button>
              </Space>
            }
          >
            {selectedAsset ? (
              <>
                <Descriptions size="small" column={1} style={{ marginBottom: 16 }}>
                  <Descriptions.Item label="简介">{selectedAsset.intro || '-'}</Descriptions.Item>
                  <Descriptions.Item label="能力">
                    模态：{selectedAsset.capabilities?.modalities?.join(', ') || '-'}；特性：
                    {selectedAsset.capabilities?.features?.join(', ') || '-'}；请求类型：
                    {selectedAsset.capabilities?.requestKinds?.join(', ') || '-'}
                  </Descriptions.Item>
                </Descriptions>
                <Table<ModelAssetBinding>
                  rowKey="bindingId"
                  dataSource={selectedBindings}
                  columns={bindingColumns}
                  pagination={false}
                />
              </>
            ) : (
              <Alert type="info" showIcon message="还没有模型资产，先在左侧创建一个资产。" />
            )}
          </Card>
        </Col>
      </Row>

      <Drawer
        title={editingAsset ? '编辑模型资产' : '新建模型资产'}
        width={640}
        open={assetDrawerOpen}
        destroyOnClose
        onClose={() => setAssetDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setAssetDrawerOpen(false)}>取消</Button>
            <Button type="primary" onClick={saveAsset}>
              保存
            </Button>
          </Space>
        }
      >
        <Form<AssetFormValues> form={assetForm} layout="vertical">
          {editingAssetHasLegacyTags || editingAssetHasLegacyCapabilities ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="该资产包含历史非预置字段。保存后将按系统预置值收口，未选择的历史值不会保留。"
            />
          ) : null}
          <Form.Item name="assetId" label="资产 ID" rules={[{ required: true, message: '请输入资产 ID' }]}>
            <Input disabled={!!editingAsset} placeholder="例如 gpt-4o-mini" />
          </Form.Item>
          <Form.Item name="canonicalName" label="规范名" rules={[{ required: true, message: '请输入规范名' }]}>
            <Input placeholder="例如 openai/gpt-4o-mini" />
          </Form.Item>
          <Form.Item name="displayName" label="展示名" rules={[{ required: true, message: '请输入展示名' }]}>
            <Input placeholder="例如 GPT-4o mini" />
          </Form.Item>
          <Form.Item name="intro" label="简介">
            <TextArea rows={4} placeholder="对模型的简介、适用场景和备注。" />
          </Form.Item>
          <Form.Item
            name="tags"
            label="标签"
            extra="仅允许选择系统预置标签，用于统一模型标签口径。"
          >
            <Select
              mode="multiple"
              options={MODEL_ASSET_PRESET_TAGS.map((tag) => ({ label: tag, value: tag }))}
              placeholder="请选择系统预置标签"
            />
          </Form.Item>
          <Form.Item name="modalities" label="模态" extra="仅允许选择系统预置模态。">
            <Select
              mode="multiple"
              options={(assetOptions.capabilities?.modalities || []).map((item) => ({ label: item, value: item }))}
              placeholder="请选择模态"
            />
          </Form.Item>
          <Form.Item name="features" label="能力特性" extra="仅允许选择系统预置能力特性。">
            <Select
              mode="multiple"
              options={(assetOptions.capabilities?.features || []).map((item) => ({ label: item, value: item }))}
              placeholder="请选择能力特性"
            />
          </Form.Item>
          <Form.Item name="requestKinds" label="请求类型" extra="仅允许选择系统预置请求类型。">
            <Select
              mode="multiple"
              options={(assetOptions.capabilities?.requestKinds || []).map((item) => ({ label: item, value: item }))}
              placeholder="请选择请求类型"
            />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={editingBinding ? '编辑发布绑定' : '新建发布绑定'}
        width={820}
        open={bindingDrawerOpen}
        destroyOnClose
        onClose={() => {
          setBindingDrawerOpen(false);
          setEditingBinding(undefined);
          setPriceVersions([]);
        }}
        extra={
          <Space>
            {editingBinding ? (
              <Button onClick={() => openHistoryDrawer(editingBinding)}>
                价格历史
              </Button>
            ) : null}
            <Button
              onClick={() => {
                setBindingDrawerOpen(false);
                setEditingBinding(undefined);
                setPriceVersions([]);
              }}
            >
              取消
            </Button>
            <Button type="primary" onClick={saveBinding}>
              保存
            </Button>
          </Space>
        }
      >
        {editingBinding ? (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={`当前状态：${editingBinding.status || 'draft'}`}
            description={
              <>
                <div>发布时间：{editingBinding.publishedAt || '-'}</div>
                <div>下架时间：{editingBinding.unpublishedAt || '-'}</div>
              </>
            }
          />
        ) : null}
        {activePriceVersion ? (
          <Card size="small" title="当前生效价格版本" style={{ marginBottom: 16 }}>
            <Descriptions size="small" column={1}>
              <Descriptions.Item label="版本">#{activePriceVersion.versionId}</Descriptions.Item>
              <Descriptions.Item label="生效时间">{activePriceVersion.effectiveFrom || '-'}</Descriptions.Item>
              <Descriptions.Item label="价格摘要">
                {describePricing(activePriceVersion.pricing)}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        ) : null}
        <Form<BindingFormValues> form={bindingForm} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="bindingId"
                label="绑定 ID"
                rules={[{ required: true, message: '请输入绑定 ID' }]}
              >
                <Input disabled={!!editingBinding} placeholder="例如 gpt-4o-mini-openai" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="modelId"
                label="可展示模型 ID"
                rules={[{ required: true, message: '请输入模型 ID' }]}
              >
                {currentProviderUsesCatalog ? (
                  <Select
                    showSearch
                    options={currentModelIdOptions}
                    placeholder="选择 Provider 目录中的模型 ID"
                    onChange={(value) => syncBindingModelPair('modelId', value)}
                  />
                ) : (
                  <Input placeholder="Portal 对外展示和调用的模型 ID" />
                )}
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="providerName"
                label="Provider"
                rules={[{ required: true, message: '请选择 Provider' }]}
              >
                <Select
                  loading={providersLoading}
                  options={providers.map((item) => ({ label: item.name, value: item.name }))}
                  placeholder="选择已存在的 Provider"
                  showSearch
                  onChange={(providerName) => {
                    const providerModels = providerModelCatalog[providerName] || [];
                    if (!providerModels.length) {
                      return;
                    }
                    const currentModelId = bindingForm.getFieldValue('modelId');
                    const currentTargetModel = bindingForm.getFieldValue('targetModel');
                    const matched =
                      providerModels.find((item) => item.modelId === currentModelId)
                      || providerModels.find((item) => item.targetModel === currentTargetModel);
                    bindingForm.setFieldsValue(
                      matched
                        ? { modelId: matched.modelId, targetModel: matched.targetModel }
                        : { modelId: undefined, targetModel: undefined },
                    );
                  }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="targetModel"
                label="目标模型"
                rules={[{ required: true, message: '请输入目标模型' }]}
              >
                {currentProviderUsesCatalog ? (
                  <Select
                    showSearch
                    options={currentTargetModelOptions}
                    placeholder="选择 Provider 目录中的目标模型"
                    onChange={(value) => syncBindingModelPair('targetModel', value)}
                  />
                ) : (
                  <Input placeholder="Provider 实际请求使用的模型名" />
                )}
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="protocol" label="协议">
                <Input placeholder="默认 openai/v1" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="endpoint" label="入口地址">
                <Input placeholder="可选，留空则使用 Provider 默认入口" />
              </Form.Item>
            </Col>
          </Row>
          {watchedProviderName && !currentProviderUsesCatalog ? (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message="当前 Provider 未配置预置模型目录"
              description="该 Provider 暂未提供系统预置模型列表，模型 ID 和目标模型可继续手填。"
            />
          ) : null}
          {bindingHasLegacyCatalogValue ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="当前绑定包含历史模型值"
              description="当前值不在系统预置模型目录中；如继续保存，建议重新选择预置模型。"
            />
          ) : null}

          <Divider orientation="left">限制</Divider>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="rpm" label="RPM">
                <InputNumber style={{ width: '100%' }} min={0} precision={0} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="tpm" label="TPM">
                <InputNumber style={{ width: '100%' }} min={0} precision={0} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="contextWindow" label="Context Window">
                <InputNumber style={{ width: '100%' }} min={0} precision={0} />
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left">价格</Divider>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="currency" label="币种">
                <Input placeholder="默认 CNY" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="supportsPromptCaching"
                label="支持 Prompt Cache"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          {pricingFieldGroups.map((group) => (
            <React.Fragment key={group.title}>
              <Divider orientation="left">{group.title}</Divider>
              <Row gutter={16}>
                {group.fields.map((field) => (
                  <Col span={12} key={String(field.name)}>
                    <Form.Item name={field.name} label={field.label}>
                      <InputNumber style={{ width: '100%' }} min={0} step={field.step || 0.000001} />
                    </Form.Item>
                  </Col>
                ))}
              </Row>
            </React.Fragment>
          ))}
        </Form>
      </Drawer>

      <Drawer
        title={grantBinding ? `模型可见性授权 · ${grantBinding.modelId || grantBinding.bindingId}` : '模型可见性授权'}
        width={640}
        open={grantDrawerOpen}
        destroyOnClose
        onClose={() => setGrantDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setGrantDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={grantSaving} onClick={saveGrantAssignments}>
              保存
            </Button>
          </Space>
        }
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="未配置授权记录时，已发布模型默认公开可见。配置任意 grant 后，只对命中的 consumer、department 或 user level 可见。"
        />
        <Form layout="vertical">
          <Form.Item label="允许访问的 Consumer">
            <Select
              mode="multiple"
              loading={grantLoading}
              options={consumerOptions}
              value={grantConsumers}
              onChange={setGrantConsumers}
              placeholder="选择允许访问的账号"
              showSearch
            />
          </Form.Item>
          <Form.Item label="允许访问的 Department">
            <Select
              mode="multiple"
              loading={grantLoading}
              options={departmentOptions}
              value={grantDepartments}
              onChange={setGrantDepartments}
              placeholder="选择允许访问的部门子树"
              showSearch
            />
          </Form.Item>
          <Form.Item label="允许访问的用户等级">
            <Select
              mode="multiple"
              loading={grantLoading}
              options={userLevelOptions}
              value={grantUserLevels}
              onChange={setGrantUserLevels}
              placeholder="选择允许访问的用户等级"
            />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={historyBinding ? `价格历史 · ${historyBinding.modelId || historyBinding.bindingId}` : '价格历史'}
        width={920}
        open={historyDrawerOpen}
        destroyOnClose
        onClose={() => {
          setHistoryDrawerOpen(false);
          setHistoryBinding(undefined);
        }}
      >
        <Paragraph type="secondary">
          历史版本只支持“恢复到草稿”，不会直接把旧版本重新激活。回退流程固定为 `restore -&gt; publish`。
        </Paragraph>
        <Table<ModelBindingPriceVersion>
          rowKey="versionId"
          loading={priceVersionsLoading}
          dataSource={priceVersions}
          columns={historyColumns}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  );
};

export default ModelAssetsPage;
