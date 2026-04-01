import {
  AiQuotaConsumerQuota,
  AiQuotaRouteSummary,
  AiQuotaScheduleAction,
  AiQuotaScheduleRule,
  AiQuotaScheduleRuleRequest,
  AiQuotaUserPolicy,
  AiQuotaUserPolicyRequest,
} from '@/interfaces/ai-quota';
import {
  deleteAiQuotaScheduleRule,
  deltaAiQuota,
  getAiQuotaConsumers,
  getAiQuotaRoutes,
  getAiQuotaScheduleRules,
  getAiQuotaUserPolicy,
  refreshAiQuota,
  saveAiQuotaUserPolicy,
  saveAiQuotaScheduleRule,
} from '@/services/ai-quota';
import {
  dateTimeLocalInputToISOString,
  formatDateTimeDisplay,
  formatDateTimeLocalInputValue,
  getNowDateTimeLocalInputValue,
} from '@/utils/time';
import { RedoOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { useRequest } from 'ahooks';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;
const BUILTIN_ADMIN_CONSUMER = 'administrator';
const MICRO_YUAN_PER_RMB = 1_000_000;

type QuotaModalType = 'refresh' | 'delta' | null;

const isAmountQuotaUnit = (quotaUnit?: string) => quotaUnit === 'amount';

const toDisplayQuota = (value: number, quotaUnit?: string) => {
  if (!isAmountQuotaUnit(quotaUnit)) {
    return `${value ?? 0}`;
  }
  return `${(value / MICRO_YUAN_PER_RMB).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  })} RMB`;
};

const toFormQuotaValue = (value: number, quotaUnit?: string) => {
  if (!isAmountQuotaUnit(quotaUnit)) {
    return value ?? 0;
  }
  return (value ?? 0) / MICRO_YUAN_PER_RMB;
};

const toStoredQuotaValue = (value: number, quotaUnit?: string) => {
  if (!isAmountQuotaUnit(quotaUnit)) {
    return Math.round(value ?? 0);
  }
  return Math.round((value ?? 0) * MICRO_YUAN_PER_RMB);
};

const AiQuotaPage: React.FC = () => {
  const { t } = useTranslation();
  const [routes, setRoutes] = useState<AiQuotaRouteSummary[]>([]);
  const [selectedRouteName, setSelectedRouteName] = useState<string>();
  const [quotaList, setQuotaList] = useState<AiQuotaConsumerQuota[]>([]);
  const [keyword, setKeyword] = useState('');
  const [quotaModalType, setQuotaModalType] = useState<QuotaModalType>(null);
  const [currentConsumer, setCurrentConsumer] = useState<AiQuotaConsumerQuota | null>(null);
  const [scheduleDrawerOpen, setScheduleDrawerOpen] = useState(false);
  const [scheduleRules, setScheduleRules] = useState<AiQuotaScheduleRule[]>([]);
  const [scheduleConsumer, setScheduleConsumer] = useState<AiQuotaConsumerQuota | null>(null);
  const [editingScheduleRule, setEditingScheduleRule] = useState<AiQuotaScheduleRule | null>(null);
  const [policyDrawerOpen, setPolicyDrawerOpen] = useState(false);
  const [policyConsumer, setPolicyConsumer] = useState<AiQuotaConsumerQuota | null>(null);
  const [policyLoading, setPolicyLoading] = useState(false);
  const [policySubmitting, setPolicySubmitting] = useState(false);

  const [quotaForm] = Form.useForm();
  const [scheduleForm] = Form.useForm();
  const [policyForm] = Form.useForm();

  const selectedRoute = useMemo(
    () => routes.find((route) => route.routeName === selectedRouteName),
    [routes, selectedRouteName],
  );
  const amountMode = isAmountQuotaUnit(selectedRoute?.quotaUnit);
  const quotaInputPrecision = isAmountQuotaUnit(selectedRoute?.quotaUnit) ? 6 : 0;
  const quotaInputStep = isAmountQuotaUnit(selectedRoute?.quotaUnit) ? 0.01 : 1;
  const quotaUnitLabel = amountMode
    ? t('aiQuota.units.amount')
    : t('aiQuota.units.token');
  const quotaColumnTitle = amountMode ? t('aiQuota.columns.balance') : t('aiQuota.columns.quota');
  const refreshActionLabel = amountMode ? t('aiQuota.actions.refreshBalance') : t('aiQuota.actions.refresh');
  const deltaActionLabel = amountMode ? t('aiQuota.actions.deltaBalance') : t('aiQuota.actions.delta');
  const summaryKeyPrefixLabel = amountMode
    ? t('aiQuota.summary.balanceKeyPrefix')
    : t('aiQuota.summary.redisKeyPrefix');
  const refreshModalTitle = amountMode
    ? t('aiQuota.modals.refreshBalanceTitle')
    : t('aiQuota.modals.refreshTitle');
  const deltaModalTitle = amountMode
    ? t('aiQuota.modals.deltaBalanceTitle')
    : t('aiQuota.modals.deltaTitle');
  const refreshValueLabel = amountMode
    ? t('aiQuota.modals.refreshBalanceValue')
    : t('aiQuota.modals.refreshValue');
  const deltaValueLabel = amountMode
    ? t('aiQuota.modals.deltaBalanceValue')
    : t('aiQuota.modals.deltaValue');
  const refreshSuccessMessage = amountMode
    ? t('aiQuota.messages.refreshBalanceSuccess')
    : t('aiQuota.messages.refreshSuccess');
  const deltaSuccessMessage = amountMode
    ? t('aiQuota.messages.deltaBalanceSuccess')
    : t('aiQuota.messages.deltaSuccess');
  const scheduleTitle = amountMode ? t('aiQuota.schedule.balanceTitle') : t('aiQuota.schedule.title');
  const scheduleRefreshLabel = amountMode
    ? t('aiQuota.schedule.actions.refreshBalance')
    : t('aiQuota.schedule.actions.refresh');
  const scheduleDeltaLabel = amountMode
    ? t('aiQuota.schedule.actions.deltaBalance')
    : t('aiQuota.schedule.actions.delta');

  const createDefaultPolicyValues = (policy?: Partial<AiQuotaUserPolicy>) => ({
    limitTotal: toFormQuotaValue(policy?.limitTotal ?? 0, selectedRoute?.quotaUnit),
    limit5h: toFormQuotaValue(policy?.limit5h ?? 0, selectedRoute?.quotaUnit),
    limitDaily: toFormQuotaValue(policy?.limitDaily ?? 0, selectedRoute?.quotaUnit),
    dailyResetTime: policy?.dailyResetTime || '00:00',
    limitWeekly: toFormQuotaValue(policy?.limitWeekly ?? 0, selectedRoute?.quotaUnit),
    limitMonthly: toFormQuotaValue(policy?.limitMonthly ?? 0, selectedRoute?.quotaUnit),
    costResetAt: formatDateTimeLocalInputValue(policy?.costResetAt || '', ''),
  });

  const { loading: routesLoading, run: loadRoutes } = useRequest(getAiQuotaRoutes, {
    manual: true,
    onSuccess: (result = []) => {
      setRoutes(result);
      if (!result.length) {
        setSelectedRouteName(undefined);
        setQuotaList([]);
        return;
      }
      if (!selectedRouteName || !result.some((route) => route.routeName === selectedRouteName)) {
        setSelectedRouteName(result[0].routeName);
      }
    },
  });

  const { loading: quotaLoading, run: loadConsumers } = useRequest(getAiQuotaConsumers, {
    manual: true,
    onSuccess: (result = []) => {
      setQuotaList(result);
    },
  });

  const { loading: scheduleLoading, run: loadSchedules } = useRequest(getAiQuotaScheduleRules, {
    manual: true,
    onSuccess: (result = []) => {
      setScheduleRules(result);
    },
  });

  useEffect(() => {
    loadRoutes();
  }, []);

  useEffect(() => {
    if (selectedRouteName) {
      loadConsumers(selectedRouteName);
    }
  }, [selectedRouteName]);

  useEffect(() => {
    setPolicyDrawerOpen(false);
    setPolicyConsumer(null);
    policyForm.resetFields();
  }, [selectedRouteName]);

  const filteredQuotaList = useMemo(() => {
    return quotaList.filter((item) => {
      if (item.consumerName === BUILTIN_ADMIN_CONSUMER) {
        return false;
      }
      if (!keyword) {
        return true;
      }
      return item.consumerName.toLowerCase().includes(keyword.toLowerCase());
    });
  }, [keyword, quotaList]);

  const refreshAll = async () => {
    await loadRoutes();
    if (selectedRouteName) {
      await loadConsumers(selectedRouteName);
    }
  };

  const openQuotaModal = (type: QuotaModalType, consumer: AiQuotaConsumerQuota) => {
    if (consumer.consumerName === BUILTIN_ADMIN_CONSUMER) {
      return;
    }
    setQuotaModalType(type);
    setCurrentConsumer(consumer);
    quotaForm.setFieldsValue({
      value: type === 'refresh' ? toFormQuotaValue(consumer.quota, selectedRoute?.quotaUnit) : 0,
    });
  };

  const closeQuotaModal = () => {
    setQuotaModalType(null);
    setCurrentConsumer(null);
    quotaForm.resetFields();
  };

  const submitQuotaModal = async () => {
    if (!selectedRouteName || !currentConsumer || !quotaModalType) {
      return;
    }
    const values = await quotaForm.validateFields();
    const storedValue = toStoredQuotaValue(values.value, selectedRoute?.quotaUnit);
    if (quotaModalType === 'refresh') {
      await refreshAiQuota(selectedRouteName, currentConsumer.consumerName, storedValue);
      message.success(refreshSuccessMessage);
    } else {
      await deltaAiQuota(selectedRouteName, currentConsumer.consumerName, storedValue);
      message.success(deltaSuccessMessage);
    }
    closeQuotaModal();
    await loadConsumers(selectedRouteName);
  };

  const openScheduleDrawer = async (consumer: AiQuotaConsumerQuota) => {
    if (!selectedRouteName || consumer.consumerName === BUILTIN_ADMIN_CONSUMER) {
      return;
    }
    setScheduleConsumer(consumer);
    setEditingScheduleRule(null);
    scheduleForm.setFieldsValue({
      action: 'REFRESH',
      cron: '0 0 0 * * *',
      value: toFormQuotaValue(consumer.quota, selectedRoute?.quotaUnit),
      enabled: true,
    });
    setScheduleDrawerOpen(true);
    await loadSchedules(selectedRouteName, consumer.consumerName);
  };

  const closeScheduleDrawer = () => {
    setScheduleDrawerOpen(false);
    setScheduleConsumer(null);
    setEditingScheduleRule(null);
    setScheduleRules([]);
    scheduleForm.resetFields();
  };

  const openPolicyDrawer = async (consumer: AiQuotaConsumerQuota) => {
    if (!selectedRouteName || !amountMode || consumer.consumerName === BUILTIN_ADMIN_CONSUMER) {
      return;
    }
    setPolicyConsumer(consumer);
    setPolicyDrawerOpen(true);
    setPolicyLoading(true);
    try {
      const policy = await getAiQuotaUserPolicy(selectedRouteName, consumer.consumerName);
      policyForm.setFieldsValue(createDefaultPolicyValues(policy));
    } finally {
      setPolicyLoading(false);
    }
  };

  const closePolicyDrawer = () => {
    setPolicyDrawerOpen(false);
    setPolicyConsumer(null);
    setPolicyLoading(false);
    setPolicySubmitting(false);
    policyForm.resetFields();
  };

  const resetPolicyForm = async () => {
    if (!selectedRouteName || !policyConsumer) {
      return;
    }
    setPolicyLoading(true);
    try {
      const policy = await getAiQuotaUserPolicy(selectedRouteName, policyConsumer.consumerName);
      policyForm.setFieldsValue(createDefaultPolicyValues(policy));
    } finally {
      setPolicyLoading(false);
    }
  };

  const fillPolicyResetNow = () => {
    policyForm.setFieldsValue({ costResetAt: getNowDateTimeLocalInputValue() });
  };

  const clearPolicyResetAt = () => {
    policyForm.setFieldsValue({ costResetAt: '' });
  };

  const submitPolicy = async () => {
    if (!selectedRouteName || !policyConsumer) {
      return;
    }
    const values = await policyForm.validateFields();
    const payload: AiQuotaUserPolicyRequest = {
      limitTotal: toStoredQuotaValue(values.limitTotal, selectedRoute?.quotaUnit),
      limit5h: toStoredQuotaValue(values.limit5h, selectedRoute?.quotaUnit),
      limitDaily: toStoredQuotaValue(values.limitDaily, selectedRoute?.quotaUnit),
      dailyResetMode: 'fixed',
      dailyResetTime: values.dailyResetTime,
      limitWeekly: toStoredQuotaValue(values.limitWeekly, selectedRoute?.quotaUnit),
      limitMonthly: toStoredQuotaValue(values.limitMonthly, selectedRoute?.quotaUnit),
      costResetAt: dateTimeLocalInputToISOString(values.costResetAt?.trim()),
    };
    setPolicySubmitting(true);
    try {
      const saved = await saveAiQuotaUserPolicy(selectedRouteName, policyConsumer.consumerName, payload);
      message.success(t('aiQuota.messages.policySaved'));
      policyForm.setFieldsValue(createDefaultPolicyValues(saved));
    } finally {
      setPolicySubmitting(false);
    }
  };

  const submitScheduleRule = async () => {
    if (!selectedRouteName || !scheduleConsumer) {
      return;
    }
    const values = await scheduleForm.validateFields();
    const payload: AiQuotaScheduleRuleRequest = {
      id: editingScheduleRule?.id,
      consumerName: scheduleConsumer.consumerName,
      action: values.action as AiQuotaScheduleAction,
      cron: values.cron,
      value: toStoredQuotaValue(values.value, selectedRoute?.quotaUnit),
      enabled: values.enabled,
    };
    await saveAiQuotaScheduleRule(selectedRouteName, payload);
    message.success(t('aiQuota.messages.scheduleSaved'));
    setEditingScheduleRule(null);
    scheduleForm.setFieldsValue({
      action: 'REFRESH',
      cron: '0 0 0 * * *',
      value: toFormQuotaValue(scheduleConsumer.quota, selectedRoute?.quotaUnit),
      enabled: true,
    });
    await loadSchedules(selectedRouteName, scheduleConsumer.consumerName);
    await loadRoutes();
  };

  const editScheduleRule = (rule: AiQuotaScheduleRule) => {
    setEditingScheduleRule(rule);
    scheduleForm.setFieldsValue({
      action: rule.action,
      cron: rule.cron,
      value: toFormQuotaValue(rule.value, selectedRoute?.quotaUnit),
      enabled: rule.enabled,
    });
  };

  const removeScheduleRule = async (rule: AiQuotaScheduleRule) => {
    if (!selectedRouteName || !scheduleConsumer) {
      return;
    }
    await deleteAiQuotaScheduleRule(selectedRouteName, rule.id);
    message.success(t('aiQuota.messages.scheduleDeleted'));
    if (editingScheduleRule?.id === rule.id) {
      setEditingScheduleRule(null);
      scheduleForm.setFieldsValue({
        action: 'REFRESH',
        cron: '0 0 0 * * *',
        value: toFormQuotaValue(scheduleConsumer.quota, selectedRoute?.quotaUnit),
        enabled: true,
      });
    }
    await loadSchedules(selectedRouteName, scheduleConsumer.consumerName);
    await loadRoutes();
  };

  const resetScheduleForm = () => {
    setEditingScheduleRule(null);
    scheduleForm.setFieldsValue({
      action: 'REFRESH',
      cron: '0 0 0 * * *',
      value: toFormQuotaValue(scheduleConsumer?.quota ?? 0, selectedRoute?.quotaUnit),
      enabled: true,
    });
  };

  const quotaColumns = [
    {
      title: t('aiQuota.columns.consumer'),
      dataIndex: 'consumerName',
      key: 'consumerName',
    },
    {
      title: quotaColumnTitle,
      dataIndex: 'quota',
      key: 'quota',
      render: (value: number) => toDisplayQuota(value, selectedRoute?.quotaUnit),
    },
    {
      title: t('aiQuota.columns.actions'),
      key: 'actions',
      width: amountMode ? 280 : 220,
      render: (_: unknown, record: AiQuotaConsumerQuota) => (
        <Space size="small">
          <a onClick={() => openQuotaModal('refresh', record)}>{refreshActionLabel}</a>
          <a onClick={() => openQuotaModal('delta', record)}>{deltaActionLabel}</a>
          {amountMode && <a onClick={() => openPolicyDrawer(record)}>{t('aiQuota.actions.policy')}</a>}
          <a onClick={() => openScheduleDrawer(record)}>{t('aiQuota.actions.schedule')}</a>
        </Space>
      ),
    },
  ];

  const scheduleColumns = [
    {
      title: t('aiQuota.schedule.columns.action'),
      dataIndex: 'action',
      key: 'action',
      render: (value: AiQuotaScheduleAction) => (
        <Tag color={value === 'REFRESH' ? 'blue' : 'green'}>
          {value === 'REFRESH' ? scheduleRefreshLabel : scheduleDeltaLabel}
        </Tag>
      ),
    },
    {
      title: t('aiQuota.schedule.columns.cron'),
      dataIndex: 'cron',
      key: 'cron',
    },
    {
      title: t('aiQuota.schedule.columns.value'),
      dataIndex: 'value',
      key: 'value',
      render: (value: number) => toDisplayQuota(value, selectedRoute?.quotaUnit),
    },
    {
      title: t('aiQuota.schedule.columns.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (value: boolean) => (value ? t('misc.enabled') : t('misc.disabled')),
    },
    {
      title: t('aiQuota.schedule.columns.lastAppliedAt'),
      dataIndex: 'lastAppliedAt',
      key: 'lastAppliedAt',
      render: (value?: number) => formatDateTimeDisplay(value),
    },
    {
      title: t('aiQuota.schedule.columns.lastError'),
      dataIndex: 'lastError',
      key: 'lastError',
      ellipsis: true,
      render: (value?: string) => value || '-',
    },
    {
      title: t('aiQuota.columns.actions'),
      key: 'actions',
      width: 140,
      render: (_: unknown, record: AiQuotaScheduleRule) => (
        <Space size="small">
          <a onClick={() => editScheduleRule(record)}>{t('misc.edit')}</a>
          <a onClick={() => removeScheduleRule(record)}>{t('misc.delete')}</a>
        </Space>
      ),
    },
  ];

  if (!routesLoading && routes.length === 0) {
    return (
      <PageContainer>
        <div style={{ background: '#fff', padding: 24 }}>
          <Empty description={t('aiQuota.empty')}>
            <Button icon={<RedoOutlined />} onClick={() => loadRoutes()}>
              {t('misc.refresh')}
            </Button>
          </Empty>
        </div>
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <div style={{ background: '#fff', padding: 24, marginBottom: 16 }}>
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space wrap size={16}>
            <div>
              <div style={{ marginBottom: 8 }}>{t('aiQuota.route')}</div>
              <Select
                style={{ width: 320 }}
                value={selectedRouteName}
                onChange={setSelectedRouteName}
                options={routes.map((route) => ({
                  label: route.routeName,
                  value: route.routeName,
                }))}
              />
            </div>
            <div>
              <div style={{ marginBottom: 8 }}>{t('aiQuota.search')}</div>
              <Input
                style={{ width: 260 }}
                allowClear
                value={keyword}
                placeholder={t('aiQuota.searchPlaceholder') as string}
                onChange={(event) => setKeyword(event.target.value)}
              />
            </div>
          </Space>
          <Button icon={<RedoOutlined />} onClick={refreshAll}>
            {t('misc.refresh')}
          </Button>
        </Space>
      </div>

      {selectedRoute && (
        <div style={{ background: '#fff', padding: 24, marginBottom: 16 }}>
          <Descriptions column={2} size="small">
            <Descriptions.Item label={t('aiQuota.summary.route')}>
              {selectedRoute.routeName}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.path')}>
              {selectedRoute.path || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.domains')}>
              {selectedRoute.domains?.length ? selectedRoute.domains.join(', ') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label={summaryKeyPrefixLabel}>
              {selectedRoute.redisKeyPrefix}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.adminConsumer')}>
              {selectedRoute.adminConsumer}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.adminPath')}>
              {selectedRoute.adminPath}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.quotaUnit')}>
              {quotaUnitLabel}
            </Descriptions.Item>
            <Descriptions.Item label={t('aiQuota.summary.scheduleRuleCount')}>
              {selectedRoute.scheduleRuleCount}
            </Descriptions.Item>
          </Descriptions>
        </div>
      )}

      <div style={{ background: '#fff', padding: 24 }}>
        <Table
          rowKey="consumerName"
          loading={quotaLoading}
          dataSource={filteredQuotaList}
          columns={quotaColumns}
          pagination={{
            showSizeChanger: true,
            showTotal: (total) => `${t('misc.total')} ${total}`,
          }}
        />
      </div>

      <Modal
        title={quotaModalType === 'refresh' ? refreshModalTitle : deltaModalTitle}
        open={!!quotaModalType}
        onCancel={closeQuotaModal}
        onOk={submitQuotaModal}
        destroyOnClose
      >
        <Form form={quotaForm} layout="vertical">
          <Form.Item label={t('aiQuota.columns.consumer')}>
            <Text>{currentConsumer?.consumerName}</Text>
          </Form.Item>
          <Form.Item
            name="value"
            label={`${quotaModalType === 'refresh' ? refreshValueLabel : deltaValueLabel} (${quotaUnitLabel})`}
            extra={amountMode ? t('aiQuota.form.amountValueHelp') : undefined}
            rules={[
              {
                required: true,
                message:
                  (quotaModalType === 'refresh'
                    ? t('aiQuota.validation.refreshValueRequired')
                    : t('aiQuota.validation.deltaValueRequired')) || '',
              },
            ]}
          >
            <InputNumber style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        width={640}
        title={t('aiQuota.policy.title')}
        open={policyDrawerOpen}
        onClose={closePolicyDrawer}
        destroyOnClose
      >
        <Alert
          style={{ marginBottom: 16 }}
          type="info"
          showIcon
          message={t('aiQuota.policy.description')}
        />
        <div style={{ marginBottom: 16 }}>
          <Text strong>{t('aiQuota.columns.consumer')}:</Text>{' '}
          <Text>{policyConsumer?.consumerName || '-'}</Text>
        </div>
        <Form
          form={policyForm}
          layout="vertical"
          initialValues={createDefaultPolicyValues()}
        >
          <Form.Item label={t('aiQuota.policy.form.dailyResetMode')}>
            <Text>{t('aiQuota.policy.form.fixedResetMode')}</Text>
          </Form.Item>
          <Form.Item
            name="limitTotal"
            label={`${t('aiQuota.policy.form.limitTotal')} (${quotaUnitLabel})`}
            extra={t('aiQuota.policy.form.amountHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.policyLimitRequired') || '' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item
            name="limit5h"
            label={`${t('aiQuota.policy.form.limit5h')} (${quotaUnitLabel})`}
            extra={t('aiQuota.policy.form.amountHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.policyLimitRequired') || '' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item
            name="limitDaily"
            label={`${t('aiQuota.policy.form.limitDaily')} (${quotaUnitLabel})`}
            extra={t('aiQuota.policy.form.amountHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.policyLimitRequired') || '' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item
            name="dailyResetTime"
            label={t('aiQuota.policy.form.dailyResetTime')}
            extra={t('aiQuota.policy.form.dailyResetTimeHelp')}
            rules={[
              { required: true, message: t('aiQuota.validation.policyDailyResetTimeRequired') || '' },
              {
                pattern: /^([01][0-9]|2[0-3]):[0-5][0-9]$/,
                message: t('aiQuota.validation.policyDailyResetTimeInvalid') || '',
              },
            ]}
          >
            <Input placeholder="00:00" />
          </Form.Item>
          <Form.Item
            name="limitWeekly"
            label={`${t('aiQuota.policy.form.limitWeekly')} (${quotaUnitLabel})`}
            extra={t('aiQuota.policy.form.amountHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.policyLimitRequired') || '' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item
            name="limitMonthly"
            label={`${t('aiQuota.policy.form.limitMonthly')} (${quotaUnitLabel})`}
            extra={t('aiQuota.policy.form.amountHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.policyLimitRequired') || '' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item
            name="costResetAt"
            label={t('aiQuota.policy.form.costResetAt')}
            extra={t('aiQuota.policy.form.costResetAtHelp')}
          >
            <Input placeholder="2026-03-27T10:30" />
          </Form.Item>
          <Space style={{ marginBottom: 24 }}>
            <Button onClick={fillPolicyResetNow}>{t('aiQuota.policy.actions.setResetNow')}</Button>
            <Button onClick={clearPolicyResetAt}>{t('aiQuota.policy.actions.clearResetAt')}</Button>
          </Space>
          <Space>
            <Button type="primary" loading={policySubmitting} onClick={submitPolicy}>
              {t('misc.save')}
            </Button>
            <Button loading={policyLoading} onClick={resetPolicyForm}>
              {t('misc.reset')}
            </Button>
          </Space>
        </Form>
      </Drawer>

      <Drawer
        width={760}
        title={scheduleTitle}
        open={scheduleDrawerOpen}
        onClose={closeScheduleDrawer}
        destroyOnClose
      >
        <div style={{ marginBottom: 16 }}>
          <Text strong>{t('aiQuota.columns.consumer')}:</Text>{' '}
          <Text>{scheduleConsumer?.consumerName || '-'}</Text>
        </div>
        <Form form={scheduleForm} layout="vertical">
          <Form.Item
            name="action"
            label={t('aiQuota.schedule.form.action')}
            rules={[{ required: true, message: t('aiQuota.validation.scheduleActionRequired') || '' }]}
          >
            <Select
              options={[
                { label: scheduleRefreshLabel, value: 'REFRESH' },
                { label: scheduleDeltaLabel, value: 'DELTA' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="cron"
            label={t('aiQuota.schedule.form.cron')}
            extra={t('aiQuota.schedule.form.cronHelp')}
            rules={[{ required: true, message: t('aiQuota.validation.scheduleCronRequired') || '' }]}
          >
            <Input placeholder="0 0 0 * * *" />
          </Form.Item>
          <Form.Item
            name="value"
            label={`${t('aiQuota.schedule.form.value')} (${quotaUnitLabel})`}
            extra={amountMode ? t('aiQuota.schedule.form.amountValueHelp') : undefined}
            rules={[{ required: true, message: t('aiQuota.validation.scheduleValueRequired') || '' }]}
          >
            <InputNumber style={{ width: '100%' }} precision={quotaInputPrecision} step={quotaInputStep} />
          </Form.Item>
          <Form.Item name="enabled" label={t('aiQuota.schedule.form.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Space style={{ marginBottom: 24 }}>
            <Button type="primary" onClick={submitScheduleRule}>
              {editingScheduleRule ? t('misc.save') : t('misc.create')}
            </Button>
            <Button onClick={resetScheduleForm}>{t('misc.reset')}</Button>
          </Space>
        </Form>

        <Table
          rowKey="id"
          loading={scheduleLoading}
          dataSource={scheduleRules}
          columns={scheduleColumns}
          pagination={false}
        />
      </Drawer>
    </PageContainer>
  );
};

export default AiQuotaPage;
