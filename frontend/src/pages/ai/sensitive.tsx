import {
  AiSensitiveBlockAudit,
  AiSensitiveDateTimeValue,
  AiSensitiveDetectRule,
  AiSensitiveMatchType,
  AiSensitiveReplacePreset,
  AiSensitiveReplaceRule,
  AiSensitiveReplaceType,
  AiSensitiveStatus,
  AiSensitiveSystemConfig,
} from '@/interfaces/ai-sensitive';
import {
  deleteAiSensitiveDetectRule,
  deleteAiSensitiveReplaceRule,
  getAiSensitiveAudits,
  getAiSensitiveDetectRules,
  getAiSensitiveReplaceRules,
  getAiSensitiveStatus,
  getAiSensitiveSystemConfig,
  reconcileAiSensitiveRules,
  saveAiSensitiveDetectRule,
  saveAiSensitiveReplaceRule,
  updateAiSensitiveSystemConfig,
} from '@/services/ai-sensitive';
import { APP_DATE_TIME_DISPLAY_FORMAT, formatDateTimeDisplay } from '@/utils/time';
import { RedoOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import {
  Alert,
  Button,
  Card,
  DatePicker,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import moment, { Moment } from 'moment';
import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

const { Paragraph, Text } = Typography;
const { TabPane } = Tabs;
const { TextArea } = Input;

const DEFAULT_AUDIT_LIMIT = 200;
const DEFAULT_AUDIT_LOOKBACK_HOURS = 1;
const DATE_TIME_DISPLAY_FORMAT = APP_DATE_TIME_DISPLAY_FORMAT;

const formatDateTime = (value?: AiSensitiveDateTimeValue) => {
  return formatDateTimeDisplay(value);
};

const buildDefaultAuditQuery = () => ({
  startTime: moment().subtract(DEFAULT_AUDIT_LOOKBACK_HOURS, 'hour'),
  endTime: moment(),
});

const renderEnabledTag = (enabled?: boolean) => (
  typeof enabled === 'undefined' ? (
    <Tag>-</Tag>
  ) : (
    <Tag color={enabled === false ? 'default' : 'green'}>
      {enabled === false ? 'OFF' : 'ON'}
    </Tag>
  )
);

const AiSensitivePage: React.FC = () => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<AiSensitiveStatus>();
  const [detectRules, setDetectRules] = useState<AiSensitiveDetectRule[]>([]);
  const [replaceRules, setReplaceRules] = useState<AiSensitiveReplaceRule[]>([]);
  const [audits, setAudits] = useState<AiSensitiveBlockAudit[]>([]);
  const [systemConfig, setSystemConfig] = useState<AiSensitiveSystemConfig>();
  const [systemDictionaryDraft, setSystemDictionaryDraft] = useState('');
  const [loading, setLoading] = useState(false);
  const [auditLoading, setAuditLoading] = useState(false);
  const [systemSaving, setSystemSaving] = useState(false);
  const [reconciling, setReconciling] = useState(false);
  const [detectModalOpen, setDetectModalOpen] = useState(false);
  const [replaceModalOpen, setReplaceModalOpen] = useState(false);
  const [systemModalOpen, setSystemModalOpen] = useState(false);
  const [savingRule, setSavingRule] = useState(false);
  const [editingDetectRule, setEditingDetectRule] = useState<AiSensitiveDetectRule | null>(null);
  const [editingReplaceRule, setEditingReplaceRule] = useState<AiSensitiveReplaceRule | null>(null);
  const [auditError, setAuditError] = useState<string>();

  const [detectForm] = Form.useForm();
  const [replaceForm] = Form.useForm();
  const [auditForm] = Form.useForm();

  const matchTypeOptions = useMemo(
    () => [
      { label: t('aiSensitive.matchTypes.contains'), value: 'contains' },
      { label: t('aiSensitive.matchTypes.exact'), value: 'exact' },
      { label: t('aiSensitive.matchTypes.regex'), value: 'regex' },
    ],
    [t],
  );

  const replaceTypeOptions = useMemo(
    () => [
      { label: t('aiSensitive.replaceTypes.replace'), value: 'replace' },
      { label: t('aiSensitive.replaceTypes.hash'), value: 'hash' },
    ],
    [t],
  );

  const replacePresets = useMemo<AiSensitiveReplacePreset[]>(
    () => [
      {
        key: 'mobilePhone',
        label: t('aiSensitive.replacePreset.mobilePhone.label'),
        description: t('aiSensitive.replacePreset.mobilePhone.description'),
        pattern: String.raw`\b1\d{10}\b`,
        replaceType: 'replace',
        replaceValue: '1',
        restore: false,
        priority: 100,
      },
      {
        key: 'email',
        label: t('aiSensitive.replacePreset.email.label'),
        description: t('aiSensitive.replacePreset.email.description'),
        pattern: String.raw`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
        replaceType: 'replace',
        replaceValue: '[EMAIL]',
        restore: false,
        priority: 90,
      },
      {
        key: 'bankCard',
        label: t('aiSensitive.replacePreset.bankCard.label'),
        description: t('aiSensitive.replacePreset.bankCard.description'),
        pattern: String.raw`\b\d{16,19}\b`,
        replaceType: 'replace',
        replaceValue: '1',
        restore: false,
        priority: 80,
      },
    ],
    [t],
  );

  const replacePresetOptions = useMemo(
    () =>
      replacePresets.map((preset) => ({
        label: preset.label,
        value: preset.key,
      })),
    [replacePresets],
  );

  const loadStatus = async () => {
    const result = await getAiSensitiveStatus();
    setStatus(result);
  };

  const loadSystemConfig = async () => {
    const result = await getAiSensitiveSystemConfig();
    setSystemConfig(result);
  };

  const loadDetectRules = async () => {
    const result = await getAiSensitiveDetectRules();
    setDetectRules(result || []);
  };

  const loadReplaceRules = async () => {
    const result = await getAiSensitiveReplaceRules();
    setReplaceRules(result || []);
  };

  const buildAuditQueryParams = (values?: Record<string, any>) => {
    const normalizedValues: Record<string, any> = {
      limit: DEFAULT_AUDIT_LIMIT,
    };
    if (values?.consumerName) {
      normalizedValues.consumerName = values.consumerName.trim();
    }
    if (values?.displayName) {
      normalizedValues.displayName = values.displayName.trim();
    }
    if (values?.routeName) {
      normalizedValues.routeName = values.routeName.trim();
    }
    if (values?.matchType) {
      normalizedValues.matchType = values.matchType;
    }
    if (values?.startTime && moment.isMoment(values.startTime)) {
      normalizedValues.startTime = (values.startTime as Moment).toDate().toISOString();
    }
    if (values?.endTime && moment.isMoment(values.endTime)) {
      normalizedValues.endTime = (values.endTime as Moment).toDate().toISOString();
    }
    return normalizedValues;
  };

  const loadAudits = async (params?: Record<string, any>) => {
    setAuditLoading(true);
    setAuditError(undefined);
    try {
      const values = params || auditForm.getFieldsValue();
      const result = await getAiSensitiveAudits(buildAuditQueryParams(values) as any);
      setAudits(result || []);
    } catch (error) {
      setAudits([]);
      setAuditError(t('aiSensitive.messages.auditQueryFailed'));
      message.error(t('aiSensitive.messages.auditQueryFailed'));
    } finally {
      setAuditLoading(false);
    }
  };

  const loadAll = async (auditParams?: Record<string, any>) => {
    setLoading(true);
    try {
      await Promise.all([loadStatus(), loadSystemConfig(), loadDetectRules(), loadReplaceRules()]);
      await loadAudits(auditParams);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const defaults = buildDefaultAuditQuery();
    auditForm.setFieldsValue(defaults);
    loadAll(defaults);
  }, []);

  const openCreateDetectModal = () => {
    setEditingDetectRule(null);
    detectForm.setFieldsValue({
      pattern: '',
      matchType: 'contains',
      priority: 0,
      enabled: true,
      description: '',
    });
    setDetectModalOpen(true);
  };

  const openEditDetectModal = (rule: AiSensitiveDetectRule) => {
    setEditingDetectRule(rule);
    detectForm.setFieldsValue({
      ...rule,
      enabled: rule.enabled !== false,
    });
    setDetectModalOpen(true);
  };

  const openCreateReplaceModal = () => {
    setEditingReplaceRule(null);
    replaceForm.setFieldsValue({
      presetKey: undefined,
      pattern: '',
      replaceType: 'replace',
      replaceValue: '',
      restore: false,
      priority: 0,
      enabled: true,
      description: '',
    });
    setReplaceModalOpen(true);
  };

  const openEditReplaceModal = (rule: AiSensitiveReplaceRule) => {
    setEditingReplaceRule(rule);
    replaceForm.setFieldsValue({
      presetKey: undefined,
      ...rule,
      restore: !!rule.restore,
      enabled: rule.enabled !== false,
    });
    setReplaceModalOpen(true);
  };

  const applyReplacePreset = (presetKey?: string) => {
    const preset = replacePresets.find((item) => item.key === presetKey);
    if (!preset) {
      return;
    }
    replaceForm.setFieldsValue({
      ...replaceForm.getFieldsValue(),
      presetKey,
      pattern: preset.pattern,
      replaceType: preset.replaceType,
      replaceValue: preset.replaceValue,
      restore: preset.restore,
      priority: preset.priority,
      description: preset.description,
    });
  };

  const openSystemDictionaryModal = () => {
    setSystemDictionaryDraft(systemConfig?.dictionaryText || '');
    setSystemModalOpen(true);
  };

  const saveSystemConfig = async (nextConfig: AiSensitiveSystemConfig) => {
    setSystemSaving(true);
    try {
      const result = await updateAiSensitiveSystemConfig(nextConfig);
      setSystemConfig(result);
      await loadStatus();
      message.success(t('aiSensitive.messages.systemConfigSaved'));
      return result;
    } finally {
      setSystemSaving(false);
    }
  };

  const handleToggleSystemDeny = async (checked: boolean) => {
    await saveSystemConfig({
      systemDenyEnabled: checked,
      dictionaryText: systemConfig?.dictionaryText || '',
    });
  };

  const submitSystemDictionary = async () => {
    const result = await saveSystemConfig({
      systemDenyEnabled: systemConfig?.systemDenyEnabled || false,
      dictionaryText: systemDictionaryDraft,
    });
    setSystemModalOpen(false);
    setSystemDictionaryDraft(result?.dictionaryText || '');
  };

  const submitDetectRule = async () => {
    const values = await detectForm.validateFields();
    setSavingRule(true);
    try {
      await saveAiSensitiveDetectRule({
        ...editingDetectRule,
        ...values,
      });
      message.success(t('aiSensitive.messages.detectSaved'));
      setDetectModalOpen(false);
      await Promise.all([loadStatus(), loadDetectRules()]);
    } finally {
      setSavingRule(false);
    }
  };

  const submitReplaceRule = async () => {
    const values = await replaceForm.validateFields();
    const { presetKey: _presetKey, ...payload } = values;
    setSavingRule(true);
    try {
      await saveAiSensitiveReplaceRule({
        ...editingReplaceRule,
        ...payload,
      });
      message.success(t('aiSensitive.messages.replaceSaved'));
      setReplaceModalOpen(false);
      await Promise.all([loadStatus(), loadReplaceRules()]);
    } finally {
      setSavingRule(false);
    }
  };

  const handleDeleteDetectRule = (rule: AiSensitiveDetectRule) => {
    Modal.confirm({
      title: t('aiSensitive.messages.deleteDetectConfirm'),
      content: rule.pattern,
      onOk: async () => {
        if (!rule.id) {
          return;
        }
        await deleteAiSensitiveDetectRule(rule.id);
        message.success(t('aiSensitive.messages.detectDeleted'));
        await Promise.all([loadStatus(), loadDetectRules()]);
      },
    });
  };

  const handleDeleteReplaceRule = (rule: AiSensitiveReplaceRule) => {
    Modal.confirm({
      title: t('aiSensitive.messages.deleteReplaceConfirm'),
      content: rule.pattern,
      onOk: async () => {
        if (!rule.id) {
          return;
        }
        await deleteAiSensitiveReplaceRule(rule.id);
        message.success(t('aiSensitive.messages.replaceDeleted'));
        await Promise.all([loadStatus(), loadReplaceRules()]);
      },
    });
  };

  const handleReconcile = async () => {
    setReconciling(true);
    try {
      const result = await reconcileAiSensitiveRules();
      setStatus(result);
      await loadSystemConfig();
      message.success(t('aiSensitive.messages.reconciled'));
    } finally {
      setReconciling(false);
    }
  };

  const handleResetAudits = () => {
    const defaults = buildDefaultAuditQuery();
    auditForm.resetFields();
    auditForm.setFieldsValue(defaults);
    loadAudits(defaults);
  };

  const detectColumns = [
    {
      title: t('aiSensitive.fields.pattern'),
      dataIndex: 'pattern',
      key: 'pattern',
      width: 280,
    },
    {
      title: t('aiSensitive.fields.matchType'),
      dataIndex: 'matchType',
      key: 'matchType',
      render: (value: AiSensitiveMatchType) => t(`aiSensitive.matchTypes.${value}`),
      width: 140,
    },
    {
      title: t('aiSensitive.fields.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
    },
    {
      title: t('aiSensitive.fields.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (value: boolean) => renderEnabledTag(value),
      width: 100,
    },
    {
      title: t('aiSensitive.fields.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('aiSensitive.fields.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (value: string) => formatDateTime(value),
      width: 180,
    },
    {
      title: t('misc.action'),
      key: 'action',
      width: 120,
      render: (_: any, record: AiSensitiveDetectRule) => (
        <Space>
          <a onClick={() => openEditDetectModal(record)}>{t('misc.edit')}</a>
          <a onClick={() => handleDeleteDetectRule(record)}>{t('misc.delete')}</a>
        </Space>
      ),
    },
  ];

  const replaceColumns = [
    {
      title: t('aiSensitive.fields.pattern'),
      dataIndex: 'pattern',
      key: 'pattern',
      width: 260,
    },
    {
      title: t('aiSensitive.fields.replaceType'),
      dataIndex: 'replaceType',
      key: 'replaceType',
      render: (value: AiSensitiveReplaceType) => t(`aiSensitive.replaceTypes.${value}`),
      width: 140,
    },
    {
      title: t('aiSensitive.fields.replaceValue'),
      dataIndex: 'replaceValue',
      key: 'replaceValue',
      ellipsis: true,
      width: 220,
    },
    {
      title: t('aiSensitive.fields.restore'),
      dataIndex: 'restore',
      key: 'restore',
      render: (value: boolean) => renderEnabledTag(value),
      width: 100,
    },
    {
      title: t('aiSensitive.fields.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
    },
    {
      title: t('aiSensitive.fields.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (value: boolean) => renderEnabledTag(value),
      width: 100,
    },
    {
      title: t('aiSensitive.fields.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (value: string) => formatDateTime(value),
      width: 180,
    },
    {
      title: t('misc.action'),
      key: 'action',
      width: 120,
      render: (_: any, record: AiSensitiveReplaceRule) => (
        <Space>
          <a onClick={() => openEditReplaceModal(record)}>{t('misc.edit')}</a>
          <a onClick={() => handleDeleteReplaceRule(record)}>{t('misc.delete')}</a>
        </Space>
      ),
    },
  ];

  const auditColumns = [
    {
      title: t('aiSensitive.fields.blockedAt'),
      dataIndex: 'blockedAt',
      key: 'blockedAt',
      render: (value: string) => formatDateTime(value),
      width: 180,
    },
    {
      title: t('aiSensitive.fields.consumerName'),
      dataIndex: 'consumerName',
      key: 'consumerName',
      width: 160,
    },
    {
      title: t('aiSensitive.fields.displayName'),
      dataIndex: 'displayName',
      key: 'displayName',
      width: 160,
    },
    {
      title: t('aiSensitive.fields.routeName'),
      dataIndex: 'routeName',
      key: 'routeName',
      width: 180,
    },
    {
      title: t('aiSensitive.fields.phase'),
      dataIndex: 'requestPhase',
      key: 'requestPhase',
      width: 120,
    },
    {
      title: t('aiSensitive.fields.matchType'),
      dataIndex: 'matchType',
      key: 'matchType',
      render: (value: AiSensitiveMatchType) => (value ? t(`aiSensitive.matchTypes.${value}`) : '-'),
      width: 120,
    },
    {
      title: t('aiSensitive.fields.matchedRule'),
      dataIndex: 'matchedRule',
      key: 'matchedRule',
      width: 220,
      ellipsis: true,
    },
    {
      title: t('aiSensitive.fields.matchedExcerpt'),
      dataIndex: 'matchedExcerpt',
      key: 'matchedExcerpt',
      render: (value: string) => (
        <Paragraph style={{ marginBottom: 0 }} ellipsis={{ rows: 2 }}>
          {value || '-'}
        </Paragraph>
      ),
    },
  ];

  return (
    <PageContainer
      extra={[
        <Button key="refresh" icon={<RedoOutlined />} onClick={() => loadAll()} loading={loading}>
          {t('aiSensitive.actions.refresh')}
        </Button>,
        <Button key="reconcile" type="primary" onClick={handleReconcile} loading={reconciling}>
          {t('aiSensitive.actions.reconcile')}
        </Button>,
      ]}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Descriptions bordered size="small" column={3}>
          <Descriptions.Item label={t('aiSensitive.status.detectRuleCount')}>
            {status?.detectRuleCount ?? 0}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.replaceRuleCount')}>
            {status?.replaceRuleCount ?? 0}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.auditRecordCount')}>
            {status?.auditRecordCount ?? 0}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.systemDenyEnabled')}>
            {renderEnabledTag(status?.systemDenyEnabled)}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.systemDictionaryWordCount')}>
            {status?.systemDictionaryWordCount ?? 0}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.systemDictionaryUpdatedAt')}>
            {formatDateTime(status?.systemDictionaryUpdatedAt)}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.projectedInstanceCount')}>
            {status?.projectedInstanceCount ?? 0}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.lastReconciledAt')}>
            {formatDateTime(status?.lastReconciledAt)}
          </Descriptions.Item>
          <Descriptions.Item label={t('aiSensitive.status.lastMigratedAt')}>
            {formatDateTime(status?.lastMigratedAt)}
          </Descriptions.Item>
        </Descriptions>

        {status?.lastError && (
          <Alert type="error" showIcon message={status.lastError} />
        )}

        <Card
          size="small"
          title={t('aiSensitive.systemConfig.title')}
          extra={(
            <Space>
              <Text type="secondary">
                {t('aiSensitive.systemConfig.wordCount', {
                  count: status?.systemDictionaryWordCount ?? 0,
                })}
              </Text>
              <Button onClick={openSystemDictionaryModal}>
                {t('aiSensitive.actions.editSystemDictionary')}
              </Button>
            </Space>
          )}
        >
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Space>
              <Text>{t('aiSensitive.systemConfig.enabled')}</Text>
              <Switch
                checked={systemConfig?.systemDenyEnabled}
                loading={systemSaving}
                onChange={handleToggleSystemDeny}
              />
            </Space>
            <Descriptions size="small" column={3}>
              <Descriptions.Item label={t('aiSensitive.systemConfig.currentStatus')}>
                {renderEnabledTag(systemConfig?.systemDenyEnabled)}
              </Descriptions.Item>
              <Descriptions.Item label={t('aiSensitive.systemConfig.updatedBy')}>
                {systemConfig?.updatedBy || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('aiSensitive.systemConfig.updatedAt')}>
                {formatDateTime(systemConfig?.updatedAt)}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        </Card>

        <Tabs defaultActiveKey="detect">
          <TabPane tab={t('aiSensitive.tabs.detect')} key="detect">
            <Space style={{ marginBottom: 16 }}>
              <Button type="primary" onClick={openCreateDetectModal}>
                {t('aiSensitive.actions.addDetectRule')}
              </Button>
            </Space>
            <Table
              rowKey="id"
              loading={loading}
              columns={detectColumns}
              dataSource={detectRules}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane tab={t('aiSensitive.tabs.replace')} key="replace">
            <Space style={{ marginBottom: 16 }}>
              <Button type="primary" onClick={openCreateReplaceModal}>
                {t('aiSensitive.actions.addReplaceRule')}
              </Button>
            </Space>
            <Table
              rowKey="id"
              loading={loading}
              columns={replaceColumns}
              dataSource={replaceRules}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane tab={t('aiSensitive.tabs.audit')} key="audit">
            <Form
              form={auditForm}
              layout="inline"
              onFinish={(values) => loadAudits(values)}
              style={{ marginBottom: 16, rowGap: 12 }}
            >
              <Form.Item name="consumerName" label={t('aiSensitive.fields.consumerName')}>
                <Input placeholder={t('aiSensitive.placeholders.consumerName')} allowClear />
              </Form.Item>
              <Form.Item name="displayName" label={t('aiSensitive.fields.displayName')}>
                <Input placeholder={t('aiSensitive.placeholders.displayName')} allowClear />
              </Form.Item>
              <Form.Item name="routeName" label={t('aiSensitive.fields.routeName')}>
                <Input placeholder={t('aiSensitive.placeholders.routeName')} allowClear />
              </Form.Item>
              <Form.Item name="matchType" label={t('aiSensitive.fields.matchType')}>
                <Select style={{ width: 140 }} allowClear options={matchTypeOptions} />
              </Form.Item>
              <Form.Item name="startTime" label={t('aiSensitive.fields.startTime')}>
                <DatePicker
                  showTime
                  allowClear={false}
                  format={DATE_TIME_DISPLAY_FORMAT}
                />
              </Form.Item>
              <Form.Item name="endTime" label={t('aiSensitive.fields.endTime')}>
                <DatePicker
                  showTime
                  allowClear={false}
                  format={DATE_TIME_DISPLAY_FORMAT}
                />
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit">
                    {t('aiSensitive.actions.search')}
                  </Button>
                  <Button onClick={handleResetAudits}>
                    {t('aiSensitive.actions.reset')}
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            {auditError && (
              <Alert
                style={{ marginBottom: 16 }}
                type="error"
                showIcon
                message={auditError}
              />
            )}

            <Table
              rowKey="id"
              loading={auditLoading}
              columns={auditColumns}
              dataSource={audits}
              pagination={{ pageSize: 10 }}
              expandable={{
                expandedRowRender: (record) => (
                  <Descriptions column={1} size="small" bordered>
                    <Descriptions.Item label={t('aiSensitive.fields.requestId')}>
                      {record.requestId || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('aiSensitive.fields.blockedBy')}>
                      {record.blockedBy || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('aiSensitive.fields.blockedReason')}>
                      <Paragraph style={{ marginBottom: 0 }} copyable>
                        {record.blockedReasonJson || '-'}
                      </Paragraph>
                    </Descriptions.Item>
                  </Descriptions>
                ),
              }}
            />
          </TabPane>
        </Tabs>
      </Space>

      <Modal
        title={editingDetectRule ? t('aiSensitive.modals.editDetect') : t('aiSensitive.modals.addDetect')}
        open={detectModalOpen}
        onCancel={() => setDetectModalOpen(false)}
        onOk={submitDetectRule}
        confirmLoading={savingRule}
        destroyOnClose
      >
        <Form form={detectForm} layout="vertical">
          <Form.Item
            name="pattern"
            label={t('aiSensitive.fields.pattern')}
            rules={[{ required: true, message: t('aiSensitive.rules.patternRequired') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="matchType"
            label={t('aiSensitive.fields.matchType')}
            rules={[{ required: true, message: t('aiSensitive.rules.matchTypeRequired') }]}
          >
            <Select options={matchTypeOptions} />
          </Form.Item>
          <Form.Item name="priority" label={t('aiSensitive.fields.priority')}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="description" label={t('aiSensitive.fields.description')}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="enabled" label={t('aiSensitive.fields.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingReplaceRule ? t('aiSensitive.modals.editReplace') : t('aiSensitive.modals.addReplace')}
        open={replaceModalOpen}
        onCancel={() => setReplaceModalOpen(false)}
        onOk={submitReplaceRule}
        confirmLoading={savingRule}
        destroyOnClose
      >
        <Form form={replaceForm} layout="vertical">
          <Form.Item
            name="presetKey"
            label={t('aiSensitive.fields.replacePreset')}
            extra={t('aiSensitive.replacePreset.help')}
          >
            <Select
              allowClear
              options={replacePresetOptions}
              placeholder={t('aiSensitive.placeholders.replacePreset')}
              onChange={applyReplacePreset}
            />
          </Form.Item>
          <Form.Item
            name="pattern"
            label={t('aiSensitive.fields.pattern')}
            rules={[{ required: true, message: t('aiSensitive.rules.patternRequired') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="replaceType"
            label={t('aiSensitive.fields.replaceType')}
            rules={[{ required: true, message: t('aiSensitive.rules.replaceTypeRequired') }]}
          >
            <Select options={replaceTypeOptions} />
          </Form.Item>
          <Form.Item name="replaceValue" label={t('aiSensitive.fields.replaceValue')}>
            <Input />
          </Form.Item>
          <Form.Item name="priority" label={t('aiSensitive.fields.priority')}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="description" label={t('aiSensitive.fields.description')}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="restore" label={t('aiSensitive.fields.restore')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="enabled" label={t('aiSensitive.fields.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('aiSensitive.modals.editSystemDictionary')}
        open={systemModalOpen}
        onCancel={() => setSystemModalOpen(false)}
        onOk={submitSystemDictionary}
        confirmLoading={systemSaving}
        width={760}
        destroyOnClose
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Text type="secondary">{t('aiSensitive.systemConfig.dictionaryHelp')}</Text>
          <TextArea
            rows={18}
            value={systemDictionaryDraft}
            onChange={(event) => setSystemDictionaryDraft(event.target.value)}
          />
        </Space>
      </Modal>
    </PageContainer>
  );
};

export default AiSensitivePage;
