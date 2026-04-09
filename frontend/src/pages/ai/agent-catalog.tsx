import { AgentCatalogOptionServer, AgentCatalogOptions, AgentCatalogRecord } from '@/interfaces/agent-catalog';
import { AssetGrantRecord, OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import {
  createAgentCatalog,
  getAgentCatalogs,
  getAgentCatalogOptions,
  publishAgentCatalog,
  unpublishAgentCatalog,
  updateAgentCatalog,
} from '@/services/agent-catalog';
import {
  listAssetGrants,
  listOrgAccounts,
  listOrgDepartmentsTree,
  replaceAssetGrants,
} from '@/services/organization';
import { USER_LEVELS } from '@/utils/consumer-level';
import { CopyOutlined, PlusOutlined, RedoOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';

const { Paragraph, Text } = Typography;
const { TextArea } = Input;
const AGENT_CATALOG_ASSET_TYPE = 'agent_catalog';

type AssetFormValues = {
  agentId: string;
  canonicalName?: string;
  displayName?: string;
  intro?: string;
  description?: string;
  iconUrl?: string;
  tags?: string[];
  mcpServerName?: string;
};

const statusColorMap: Record<string, string> = {
  draft: 'default',
  published: 'green',
  unpublished: 'orange',
};

function flattenDepartmentNodes(nodes: OrgDepartmentNode[], bucket: OrgDepartmentNode[] = []): OrgDepartmentNode[] {
  (nodes || []).forEach((node) => {
    bucket.push(node);
    if (node.children?.length) {
      flattenDepartmentNodes(node.children, bucket);
    }
  });
  return bucket;
}

function toAssetFormValues(asset?: AgentCatalogRecord): AssetFormValues {
  return {
    agentId: asset?.agentId || '',
    canonicalName: asset?.canonicalName,
    displayName: asset?.displayName,
    intro: asset?.intro,
    description: asset?.description,
    iconUrl: asset?.iconUrl,
    tags: asset?.tags || [],
    mcpServerName: asset?.mcpServerName,
  };
}

function buildConsoleCopyUrl(pathname: string) {
  if (typeof window === 'undefined') {
    return pathname;
  }
  return `${window.location.origin}${pathname}`;
}

const AgentCatalogPage: React.FC = () => {
  const [assets, setAssets] = useState<AgentCatalogRecord[]>([]);
  const [assetOptions, setAssetOptions] = useState<AgentCatalogOptions>({ servers: [] });
  const [accounts, setAccounts] = useState<OrgAccountRecord[]>([]);
  const [departments, setDepartments] = useState<OrgDepartmentNode[]>([]);
  const [selectedAssetId, setSelectedAssetId] = useState<string>();
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [assetDrawerOpen, setAssetDrawerOpen] = useState(false);
  const [editingAsset, setEditingAsset] = useState<AgentCatalogRecord>();
  const [savingAsset, setSavingAsset] = useState(false);
  const [grantDrawerOpen, setGrantDrawerOpen] = useState(false);
  const [grantLoading, setGrantLoading] = useState(false);
  const [grantSaving, setGrantSaving] = useState(false);
  const [grantConsumers, setGrantConsumers] = useState<string[]>([]);
  const [grantDepartments, setGrantDepartments] = useState<string[]>([]);
  const [grantUserLevels, setGrantUserLevels] = useState<string[]>([]);
  const [grantAsset, setGrantAsset] = useState<AgentCatalogRecord>();

  const [assetForm] = Form.useForm<AssetFormValues>();

  const selectedAsset = useMemo(
    () => assets.find((item) => item.agentId === selectedAssetId) || assets[0],
    [assets, selectedAssetId],
  );

  const selectedServer = useMemo<AgentCatalogOptionServer | undefined>(() => {
    const mcpServerName = assetForm.getFieldValue('mcpServerName') || selectedAsset?.mcpServerName;
    return (assetOptions.servers || []).find((item) => item.mcpServerName === mcpServerName);
  }, [assetForm, assetOptions.servers, selectedAsset?.mcpServerName]);

  const departmentOptions = useMemo(
    () =>
      flattenDepartmentNodes(departments).map((item) => ({
        label: item.name,
        value: item.departmentId,
      })),
    [departments],
  );

  const refreshAssets = async (preferredAgentId?: string) => {
    setAssetsLoading(true);
    try {
      const [nextAssets, nextOptions, nextAccounts, nextDepartments] = await Promise.all([
        getAgentCatalogs(),
        getAgentCatalogOptions(),
        listOrgAccounts(),
        listOrgDepartmentsTree(),
      ]);
      setAssets(nextAssets || []);
      setAssetOptions(nextOptions || { servers: [] });
      setAccounts(nextAccounts || []);
      setDepartments(nextDepartments || []);
      const fallbackId = preferredAgentId || selectedAssetId;
      const matched = (nextAssets || []).find((item) => item.agentId === fallbackId);
      setSelectedAssetId(matched?.agentId || nextAssets?.[0]?.agentId);
    } finally {
      setAssetsLoading(false);
    }
  };

  useEffect(() => {
    refreshAssets();
  }, []);

  useEffect(() => {
    if (selectedAsset && !selectedAssetId) {
      setSelectedAssetId(selectedAsset.agentId);
    }
  }, [selectedAsset, selectedAssetId]);

  const openCreateAssetDrawer = () => {
    setEditingAsset(undefined);
    assetForm.setFieldsValue(toAssetFormValues());
    setAssetDrawerOpen(true);
  };

  const openEditAssetDrawer = (asset: AgentCatalogRecord) => {
    setEditingAsset(asset);
    assetForm.setFieldsValue(toAssetFormValues(asset));
    setAssetDrawerOpen(true);
  };

  const saveAsset = async () => {
    const values = await assetForm.validateFields();
    setSavingAsset(true);
    try {
      const payload: AgentCatalogRecord = {
        agentId: values.agentId,
        canonicalName: values.canonicalName,
        displayName: values.displayName,
        intro: values.intro,
        description: values.description,
        iconUrl: values.iconUrl,
        tags: values.tags,
        mcpServerName: values.mcpServerName,
      };
      const saved = editingAsset
        ? await updateAgentCatalog(editingAsset.agentId, payload)
        : await createAgentCatalog(payload);
      message.success(editingAsset ? '智能体目录已更新' : '智能体目录已创建');
      setAssetDrawerOpen(false);
      await refreshAssets(saved.agentId);
    } finally {
      setSavingAsset(false);
    }
  };

  const handlePublish = (asset: AgentCatalogRecord) => {
    Modal.confirm({
      title: `发布智能体 · ${asset.displayName || asset.agentId}`,
      content: '发布后 Portal 会展示该智能体，且会把当前 grant 投影到对应 MCP Server 的 consumerAuthInfo。',
      okText: '发布',
      cancelText: '取消',
      onOk: async () => {
        await publishAgentCatalog(asset.agentId);
        message.success('智能体已发布');
        await refreshAssets(asset.agentId);
      },
    });
  };

  const handleUnpublish = (asset: AgentCatalogRecord) => {
    Modal.confirm({
      title: `下架智能体 · ${asset.displayName || asset.agentId}`,
      content: '下架只会更新目录状态，不会删除底层 MCP Server 配置。',
      okText: '下架',
      cancelText: '取消',
      onOk: async () => {
        await unpublishAgentCatalog(asset.agentId);
        message.success('智能体已下架');
        await refreshAssets(asset.agentId);
      },
    });
  };

  const openGrantDrawer = async (asset: AgentCatalogRecord) => {
    setGrantAsset(asset);
    setGrantDrawerOpen(true);
    setGrantLoading(true);
    try {
      const grants = await listAssetGrants(AGENT_CATALOG_ASSET_TYPE, asset.agentId);
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
    if (!grantAsset) {
      return;
    }
    setGrantSaving(true);
    try {
      const grants: AssetGrantRecord[] = [
        ...grantConsumers.map((subjectId) => ({
          assetType: AGENT_CATALOG_ASSET_TYPE,
          assetId: grantAsset.agentId,
          subjectType: 'consumer',
          subjectId,
        })),
        ...grantDepartments.map((subjectId) => ({
          assetType: AGENT_CATALOG_ASSET_TYPE,
          assetId: grantAsset.agentId,
          subjectType: 'department',
          subjectId,
        })),
        ...grantUserLevels.map((subjectId) => ({
          assetType: AGENT_CATALOG_ASSET_TYPE,
          assetId: grantAsset.agentId,
          subjectType: 'user_level',
          subjectId,
        })),
      ];
      await replaceAssetGrants(AGENT_CATALOG_ASSET_TYPE, grantAsset.agentId, grants);
      message.success('智能体可见性授权已保存');
      setGrantDrawerOpen(false);
    } finally {
      setGrantSaving(false);
    }
  };

  const copyText = async (text: string, label: string) => {
    await navigator.clipboard.writeText(text);
    message.success(`${label}已复制`);
  };

  const assetColumns = [
    {
      title: '展示名',
      dataIndex: 'displayName',
      key: 'displayName',
      render: (_: unknown, record: AgentCatalogRecord) => record.displayName || record.canonicalName || record.agentId,
    },
    {
      title: 'MCP Server',
      dataIndex: 'mcpServerName',
      key: 'mcpServerName',
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
      title: '工具数',
      dataIndex: 'toolCount',
      key: 'toolCount',
      width: 90,
      render: (value: number) => value ?? 0,
    },
  ];

  const httpPath = selectedAsset?.mcpServerName ? `/mcp-servers/${selectedAsset.mcpServerName}` : '';
  const ssePath = selectedAsset?.mcpServerName ? `/mcp-servers/${selectedAsset.mcpServerName}/sse` : '';

  return (
    <PageContainer>
      <Row gutter={16}>
        <Col span={10}>
          <Card
            title="智能体目录"
            extra={
              <Space>
                <Button icon={<RedoOutlined />} onClick={() => refreshAssets()} loading={assetsLoading} />
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAssetDrawer}>
                  新建智能体
                </Button>
              </Space>
            }
          >
            <Table<AgentCatalogRecord>
              rowKey="agentId"
              loading={assetsLoading}
              dataSource={assets}
              columns={assetColumns}
              pagination={false}
              rowClassName={(record) => (record.agentId === selectedAsset?.agentId ? 'ant-table-row-selected' : '')}
              onRow={(record) => ({
                onClick: () => setSelectedAssetId(record.agentId),
                onDoubleClick: () => openEditAssetDrawer(record),
              })}
            />
          </Card>
        </Col>
        <Col span={14}>
          <Card
            title={selectedAsset ? `智能体详情 · ${selectedAsset.displayName || selectedAsset.agentId}` : '智能体详情'}
            extra={
              selectedAsset ? (
                <Space wrap>
                  <Button onClick={() => openEditAssetDrawer(selectedAsset)}>编辑</Button>
                  <Button onClick={() => openGrantDrawer(selectedAsset)}>授权</Button>
                  {selectedAsset.status === 'published' ? (
                    <Button onClick={() => handleUnpublish(selectedAsset)}>下架</Button>
                  ) : (
                    <Button type="primary" onClick={() => handlePublish(selectedAsset)}>
                      发布
                    </Button>
                  )}
                </Space>
              ) : null
            }
          >
            {selectedAsset ? (
              <>
                <Descriptions size="small" column={1} style={{ marginBottom: 16 }}>
                  <Descriptions.Item label="规范名">{selectedAsset.canonicalName || '-'}</Descriptions.Item>
                  <Descriptions.Item label="简介">{selectedAsset.intro || '-'}</Descriptions.Item>
                  <Descriptions.Item label="描述">
                    <Paragraph style={{ marginBottom: 0 }}>{selectedAsset.description || '-'}</Paragraph>
                  </Descriptions.Item>
                  <Descriptions.Item label="标签">
                    {selectedAsset.tags?.length ? (
                      <Space size={[0, 6]} wrap>
                        {selectedAsset.tags.map((tag) => (
                          <Tag key={tag}>{tag}</Tag>
                        ))}
                      </Space>
                    ) : (
                      '-'
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item label="MCP Server">{selectedAsset.mcpServerName || '-'}</Descriptions.Item>
                  <Descriptions.Item label="传输协议">
                    {selectedAsset.transportTypes?.length ? selectedAsset.transportTypes.join(', ') : 'http, sse'}
                  </Descriptions.Item>
                  <Descriptions.Item label="resource">
                    {selectedAsset.resourceSummary || '当前未声明，需在 MCP 源配置中维护。'}
                  </Descriptions.Item>
                  <Descriptions.Item label="prompt">
                    {selectedAsset.promptSummary || '当前未声明，需在 MCP 源配置中维护。'}
                  </Descriptions.Item>
                </Descriptions>

                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <Card size="small" title="接入地址复制">
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                        <Text code>{httpPath || '-'}</Text>
                        <Button
                          icon={<CopyOutlined />}
                          size="small"
                          disabled={!httpPath}
                          onClick={() => copyText(buildConsoleCopyUrl(httpPath), 'HTTP 地址')}
                        >
                          复制 HTTP
                        </Button>
                      </Space>
                      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                        <Text code>{ssePath || '-'}</Text>
                        <Button
                          icon={<CopyOutlined />}
                          size="small"
                          disabled={!ssePath}
                          onClick={() => copyText(buildConsoleCopyUrl(ssePath), 'SSE 地址')}
                        >
                          复制 SSE
                        </Button>
                      </Space>
                    </Space>
                  </Card>

                  <Alert
                    type={selectedAsset.status === 'published' ? 'success' : 'info'}
                    showIcon
                    message={
                      selectedAsset.status === 'published'
                        ? '已发布智能体会出现在 Portal 智能体广场中，并按 grant 过滤可见性。'
                        : '当前还未发布，Portal 不会展示该智能体。'
                    }
                    description="resource / prompt 首版只做展示与接入说明，不提供在线编辑。"
                  />
                </Space>
              </>
            ) : (
              <Alert type="info" showIcon message="还没有智能体目录，先在左侧创建一个智能体。" />
            )}
          </Card>
        </Col>
      </Row>

      <Drawer
        title={editingAsset ? '编辑智能体目录' : '新建智能体目录'}
        width={640}
        open={assetDrawerOpen}
        destroyOnClose
        onClose={() => setAssetDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setAssetDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={savingAsset} onClick={saveAsset}>
              保存
            </Button>
          </Space>
        }
      >
        <Form<AssetFormValues> form={assetForm} layout="vertical">
          <Form.Item name="agentId" label="Agent ID" rules={[{ required: true, message: '请输入 Agent ID' }]}>
            <Input disabled={!!editingAsset} placeholder="例如 weather-assistant" />
          </Form.Item>
          <Form.Item name="canonicalName" label="规范名" rules={[{ required: true, message: '请输入规范名' }]}>
            <Input placeholder="例如 higress/weather-assistant" />
          </Form.Item>
          <Form.Item name="displayName" label="展示名" rules={[{ required: true, message: '请输入展示名' }]}>
            <Input placeholder="例如 天气助手" />
          </Form.Item>
          <Form.Item name="intro" label="简介">
            <Input placeholder="一句话描述该智能体做什么" />
          </Form.Item>
          <Form.Item name="description" label="详细说明">
            <TextArea rows={4} placeholder="补充适用场景、授权范围和工具说明。" />
          </Form.Item>
          <Form.Item name="iconUrl" label="图标 URL">
            <Input placeholder="可选，用于 Portal 展示" />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Select mode="tags" placeholder="例如 MCP、企业知识库、客服" />
          </Form.Item>
          <Form.Item
            name="mcpServerName"
            label="绑定 MCP Server"
            rules={[{ required: true, message: '请选择一个 MCP Server' }]}
          >
            <Select
              disabled={editingAsset?.status === 'published'}
              showSearch
              placeholder="请选择已存在的 MCP Server"
              options={(assetOptions.servers || []).map((item) => ({
                label: item.mcpServerName,
                value: item.mcpServerName,
              }))}
            />
          </Form.Item>
          {selectedServer ? (
            <Alert
              type={selectedServer.authEnabled && selectedServer.authType ? 'success' : 'warning'}
              showIcon
              message={`MCP Server：${selectedServer.mcpServerName}`}
              description={
                <>
                  <div>类型：{selectedServer.type || '-'}</div>
                  <div>域名：{selectedServer.domains?.join(', ') || '-'}</div>
                  <div>描述：{selectedServer.description || '-'}</div>
                  <div>
                    鉴权：{selectedServer.authEnabled ? '已开启' : '未开启'} / {selectedServer.authType || '未配置'}
                  </div>
                </>
              }
            />
          ) : null}
        </Form>
      </Drawer>

      <Drawer
        title={grantAsset ? `智能体可见性授权 · ${grantAsset.displayName || grantAsset.agentId}` : '智能体可见性授权'}
        width={560}
        open={grantDrawerOpen}
        destroyOnClose
        onClose={() => setGrantDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setGrantDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={grantSaving} onClick={saveGrantAssignments}>
              保存授权
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Alert
            type="info"
            showIcon
            message="未配置授权记录时，已发布智能体默认公开可见。配置任意 grant 后，只对命中的 consumer、department 或 user level 可见。"
          />
          <div>
            <div style={{ marginBottom: 8 }}>指定账号</div>
            <Select
              mode="multiple"
              loading={grantLoading}
              value={grantConsumers}
              onChange={setGrantConsumers}
              style={{ width: '100%' }}
              placeholder="选择允许查看该智能体的 consumer"
              options={accounts.map((item) => ({ label: item.consumerName, value: item.consumerName }))}
            />
          </div>
          <div>
            <div style={{ marginBottom: 8 }}>指定部门</div>
            <Select
              mode="multiple"
              loading={grantLoading}
              value={grantDepartments}
              onChange={setGrantDepartments}
              style={{ width: '100%' }}
              placeholder="选择允许查看该智能体的部门"
              options={departmentOptions}
            />
          </div>
          <div>
            <div style={{ marginBottom: 8 }}>指定用户等级</div>
            <Select
              mode="multiple"
              loading={grantLoading}
              value={grantUserLevels}
              onChange={setGrantUserLevels}
              style={{ width: '100%' }}
              placeholder="选择允许查看该智能体的用户等级"
              options={USER_LEVELS.map((item) => ({ label: item, value: item }))}
            />
          </div>
        </Space>
      </Drawer>
    </PageContainer>
  );
};

export default AgentCatalogPage;
