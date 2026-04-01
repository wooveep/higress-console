/* eslint-disable */
// @ts-nocheck
import { Consumer, CredentialType, InviteCodeRecord } from '@/interfaces/consumer';
import {
  addConsumer,
  addConsumerDepartment,
  createInviteCode,
  disableInviteCode,
  enableInviteCode,
  deleteConsumer,
  getConsumerDepartments,
  getConsumers,
  listInviteCodes,
  resetConsumerPassword,
  updateConsumer,
  updateConsumerStatus,
} from '@/services/consumer';
import { formatDateTimeDisplay } from '@/utils/time';
import { ApartmentOutlined, ExclamationCircleOutlined, RedoOutlined, UserOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { useRequest } from 'ahooks';
import { Button, Drawer, Form, Input, message, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import ConsumerForm from './components/ConsumerForm';

const { Text } = Typography;

interface FormRef {
  reset: () => void;
  handleSubmit: () => Promise<Consumer>;
}

interface OrganizationRow {
  key: string;
  rowType: 'department' | 'user';
  name: string;
  department?: string;
  credentials?: any[];
  memberCount?: number;
  consumer?: Consumer;
  children?: OrganizationRow[];
}

const BUILTIN_ADMIN_CONSUMER = 'administrator';
const USER_LEVEL_ORDER: Record<string, number> = {
  normal: 1,
  plus: 2,
  pro: 3,
  ultra: 4,
};

const ConsumerList: React.FC = () => {
  const { t } = useTranslation();
  const [inviteCodes, setInviteCodes] = useState<InviteCodeRecord[]>([]);
  const [inviteCodeLoading, setInviteCodeLoading] = useState(false);
  const [openInviteCodeModal, setOpenInviteCodeModal] = useState(false);
  const [inviteStatusFilter, setInviteStatusFilter] = useState<string | undefined>(undefined);

  const formatDateTime = (value?: any) => {
    return formatDateTimeDisplay(value);
  };

  const renderInviteStatus = (status?: string) => {
    const normalizedStatus = (status || '').toLowerCase();
    if (normalizedStatus === 'active') {
      return <Tag color="success">{t('misc.enabled')}</Tag>;
    }
    if (normalizedStatus === 'disabled') {
      return <Tag color="error">{t('misc.disabled')}</Tag>;
    }
    if (normalizedStatus === 'used') {
      return <Tag color="processing">{t('consumer.inviteCode.status.used')}</Tag>;
    }
    return <Tag>{status || '-'}</Tag>;
  };

  const renderUserLevel = (level?: string) => {
    const normalized = (level || 'normal').toLowerCase();
    if (normalized === 'ultra') {
      return <Tag color="gold">{t('consumer.userLevel.ultra')}</Tag>;
    }
    if (normalized === 'pro') {
      return <Tag color="purple">{t('consumer.userLevel.pro')}</Tag>;
    }
    if (normalized === 'plus') {
      return <Tag color="blue">{t('consumer.userLevel.plus')}</Tag>;
    }
    return <Tag>{t('consumer.userLevel.normal')}</Tag>;
  };

  const extractMaskedKeys = (credentials?: any[]): string[] => {
    if (!Array.isArray(credentials)) {
      return [];
    }
    const result: string[] = [];
    credentials.forEach((credential) => {
      if (credential?.type !== 'key-auth' || !Array.isArray(credential?.values)) {
        return;
      }
      credential.values.forEach((value: string) => {
        if (typeof value === 'string' && value.trim()) {
          result.push(value.trim());
        }
      });
    });
    return result;
  };

  const isBuiltinAdministrator = (consumer?: Consumer | null): boolean => {
    if (!consumer?.name) {
      return false;
    }
    const normalizedName = consumer.name.trim().toLowerCase();
    const normalizedSource = (consumer.portalUserSource || '').trim().toLowerCase();
    if (normalizedName !== BUILTIN_ADMIN_CONSUMER) {
      return false;
    }
    return normalizedSource === 'system' || normalizedSource === '';
  };

  const columns = [
    {
      title: t('consumer.columns.organization'),
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      render: (_, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return (
            <Space>
              <ApartmentOutlined />
              <Text strong>{record.name}</Text>
            </Space>
          );
        }
        return (
          <Space>
            <UserOutlined />
            <span>{record.name}</span>
            {isBuiltinAdministrator(record.consumer) ? <Tag color="gold">{t('consumer.systemBuiltin')}</Tag> : null}
          </Space>
        );
      },
    },
    {
      title: t('consumer.columns.type'),
      dataIndex: 'rowType',
      key: 'rowType',
      width: 120,
      render: (_, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return <Tag color="processing">{t('consumer.type.department')}</Tag>;
        }
        return <Tag color="success">{t('consumer.type.user')}</Tag>;
      },
    },
    {
      title: t('consumer.columns.authMethods'),
      dataIndex: 'credentials',
      key: 'credentials',
      render: (value, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return <Text type="secondary">{t('consumer.departmentSummary', { count: record.memberCount || 0 })}</Text>;
        }
        const maskedKeys = extractMaskedKeys(value);
        if (!maskedKeys.length) {
          return <Text type="secondary">{t('consumer.noActiveKeys')}</Text>;
        }
        const summaryText = `${maskedKeys.slice(0, 2).join(', ')}${maskedKeys.length > 2 ? ` +${maskedKeys.length - 2}` : ''}`;
        return (
          <Space size={8}>
            <Tag color={CredentialType.KEY_AUTH.displayColor}>
              {t('consumer.keyCount', { count: maskedKeys.length })}
            </Tag>
            <Text type="secondary">{summaryText}</Text>
          </Space>
        );
      },
    },
    {
      title: t('consumer.columns.portalStatus'),
      dataIndex: 'portalStatus',
      key: 'portalStatus',
      width: 120,
      render: (_, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return '-';
        }
        const status = (record.consumer?.portalStatus || 'pending').toLowerCase();
        if (status === 'active') {
          return <Tag color="success">{t('misc.enabled')}</Tag>;
        }
        if (status === 'disabled') {
          return <Tag color="error">{t('misc.disabled')}</Tag>;
        }
        return <Tag color="default">{t('consumer.portalStatus.pending')}</Tag>;
      },
    },
    {
      title: t('consumer.columns.userLevel'),
      dataIndex: 'portalUserLevel',
      key: 'portalUserLevel',
      width: 130,
      render: (_, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return '-';
        }
        return renderUserLevel(record.consumer?.portalUserLevel);
      },
      sorter: (a: OrganizationRow, b: OrganizationRow) => {
        const left = USER_LEVEL_ORDER[(a.consumer?.portalUserLevel || 'normal').toLowerCase()] || 1;
        const right = USER_LEVEL_ORDER[(b.consumer?.portalUserLevel || 'normal').toLowerCase()] || 1;
        return left - right;
      },
    },
    {
      title: t('misc.actions'),
      dataIndex: 'action',
      key: 'action',
      width: 220,
      align: 'center',
      render: (_, record: OrganizationRow) => {
        if (record.rowType === 'department') {
          return (
            <Space size="small">
              <a onClick={() => onShowDrawer(record.department)}>{t('consumer.create')}</a>
            </Space>
          );
        }
        return (
          <Space size="small">
            {
              isBuiltinAdministrator(record.consumer)
                ? <Tag color="gold">{t('consumer.systemBuiltin')}</Tag>
                : null
            }
            {
              isBuiltinAdministrator(record.consumer)
                ? null
                : (
                  <>
            <a onClick={() => onEditDrawer(record.consumer)}>{t('misc.edit')}</a>
            {
              (record.consumer?.portalStatus || '').toLowerCase() === 'active'
                ? <a onClick={() => onToggleConsumerStatus(record.consumer, 'disabled')}>{t('consumer.disable')}</a>
                : <a onClick={() => onToggleConsumerStatus(record.consumer, 'active')}>{t('consumer.enable')}</a>
            }
            <a onClick={() => onResetConsumerPassword(record.consumer)}>{t('consumer.resetPassword')}</a>
            <a onClick={() => onShowModal(record.consumer)}>{t('misc.delete')}</a>
                  </>
                )
            }
          </Space>
        );
      },
    },
  ];

  const [form] = Form.useForm();
  const [departmentForm] = Form.useForm();
  const formRef = useRef<FormRef>(null);
  const [allConsumers, setAllConsumers] = useState<Consumer[]>([]);
  const [allDepartments, setAllDepartments] = useState<string[]>([]);
  const [keyword, setKeyword] = useState('');
  const [departmentKeyword, setDepartmentKeyword] = useState('');
  const [keySearch, setKeySearch] = useState('');
  const [currentConsumer, setCurrentConsumer] = useState<Consumer | null>(null);
  const [openDrawer, setOpenDrawer] = useState(false);
  const [openModal, setOpenModal] = useState(false);
  const [openDepartmentModal, setOpenDepartmentModal] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [departmentConfirmLoading, setDepartmentConfirmLoading] = useState(false);
  const [presetDepartment, setPresetDepartment] = useState<string>('');

  const { loading, run, refresh } = useRequest(getConsumers, {
    manual: true,
    onSuccess: (result) => {
      const consumers = (result || []) as Consumer[];
      consumers.sort((i1, i2) => {
        return i1.name.localeCompare(i2.name);
      })
      consumers.forEach(c => c.key = c.key || c.name);
      setAllConsumers(consumers);
    },
  });

  const { loading: departmentsLoading, run: loadDepartments } = useRequest(getConsumerDepartments, {
    manual: true,
    onSuccess: (result) => {
      const departments = (result || []).filter(Boolean);
      departments.sort((i1, i2) => i1.localeCompare(i2));
      setAllDepartments(departments);
    },
  });

  useEffect(() => {
    run({});
    loadDepartments();
  }, []);

  const onEditDrawer = (consumer: Consumer) => {
    setCurrentConsumer(consumer);
    setPresetDepartment('');
    setOpenDrawer(true);
  };

  const onShowDrawer = (department?: string) => {
    setOpenDrawer(true);
    setCurrentConsumer(null);
    setPresetDepartment(department || '');
  };

  const handleDrawerOK = async () => {
    const values: Consumer = formRef.current ? await formRef.current.handleSubmit() : {} as Consumer;
    if (!values) {
      return;
    };

    try {
      if (currentConsumer) {
        await updateConsumer({ version: currentConsumer.version, ...values } as Consumer);
        message.success('用户更新成功');
      } else {
        const created = await addConsumer({ ...values, version: 0 } as Consumer);
        message.success('用户创建成功');
        if (created?.portalTempPassword) {
          Modal.info({
            title: 'Portal 临时密码',
            content: `用户 ${created.name} 的临时密码：${created.portalTempPassword}`,
          });
        }
      }
      setOpenDrawer(false);
      formRef.current && formRef.current.reset();
      setPresetDepartment('');
      refresh();
      loadDepartments();
    } catch (errInfo) {
      console.log('Save failed: ', errInfo);
    }
  };

  const handleDrawerCancel = () => {
    setOpenDrawer(false);
    formRef.current && formRef.current.reset();
    setCurrentConsumer(null);
    setPresetDepartment('');
  };

  const onShowModal = (consumer: Consumer) => {
    setCurrentConsumer(consumer);
    setOpenModal(true);
  };

  const onToggleConsumerStatus = async (consumer: Consumer, status: 'active' | 'disabled') => {
    if (!consumer?.name) {
      return;
    }
    try {
      await updateConsumerStatus(consumer.name, status);
      message.success(status === 'active' ? t('consumer.enableSuccess') : t('consumer.disableSuccess'));
      refresh();
      loadDepartments();
    } catch (error) {
      message.error(t('consumer.statusUpdateFailed'));
    }
  };

  const onResetConsumerPassword = async (consumer: Consumer) => {
    if (!consumer?.name) {
      return;
    }
    try {
      const result = await resetConsumerPassword(consumer.name);
      message.success(t('consumer.resetPasswordSuccess'));
      Modal.info({
        title: t('consumer.resetPasswordTitle'),
        content: (
          <Space direction="vertical" size={8}>
            <span>{t('consumer.resetPasswordHint', { name: consumer.name })}</span>
            <Text copyable>{result?.tempPassword || '-'}</Text>
          </Space>
        ),
      });
    } catch (error) {
      message.error(t('consumer.resetPasswordFailed'));
    }
  };

  const loadInviteCodeList = async (status?: string) => {
    try {
      setInviteCodeLoading(true);
      const list = await listInviteCodes({ pageNum: 1, pageSize: 200, status });
      setInviteCodes(list || []);
    } finally {
      setInviteCodeLoading(false);
    }
  };

  const onOpenInviteCodeManager = () => {
    setOpenInviteCodeModal(true);
    loadInviteCodeList(inviteStatusFilter);
  };

  const onCreateInviteCode = async () => {
    try {
      const created = await createInviteCode(7);
      message.success(t('consumer.inviteCode.createSuccess'));
      Modal.info({
        title: t('consumer.inviteCode.codeGeneratedTitle'),
        content: (
          <Space direction="vertical" size={8}>
            <span>{t('consumer.inviteCode.codeGeneratedHint')}</span>
            <Text copyable>{created?.inviteCode || '-'}</Text>
          </Space>
        ),
      });
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.createFailed'));
    }
  };

  const onDisableInviteCode = async (code?: string) => {
    if (!code) {
      return;
    }
    try {
      await disableInviteCode(code);
      message.success(t('consumer.inviteCode.disableSuccess'));
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.disableFailed'));
    }
  };

  const onEnableInviteCode = async (code?: string) => {
    if (!code) {
      return;
    }
    try {
      await enableInviteCode(code);
      message.success(t('consumer.inviteCode.enableSuccess'));
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.enableFailed'));
    }
  };

  const handleModalOk = async () => {
    setConfirmLoading(true);
    try {
      await deleteConsumer(currentConsumer.name);
      message.success(t("consumer.deleteSuccess"));
    } catch (error) { }
    setConfirmLoading(false);
    setOpenModal(false);
    refresh();
    loadDepartments();
  };

  const handleModalCancel = () => {
    setOpenModal(false);
    setCurrentConsumer(null);
  };

  const handleDepartmentModalOk = async () => {
    try {
      const values = await departmentForm.validateFields();
      setDepartmentConfirmLoading(true);
      await addConsumerDepartment(values.name);
      message.success(t('consumer.departmentCreateSuccess'));
      setOpenDepartmentModal(false);
      departmentForm.resetFields();
      loadDepartments();
    } catch (error) {
    } finally {
      setDepartmentConfirmLoading(false);
    }
  };

  const handleDepartmentModalCancel = () => {
    setOpenDepartmentModal(false);
    departmentForm.resetFields();
  };

  const handleReset = () => {
    setKeyword('');
    setDepartmentKeyword('');
    setKeySearch('');
    form.resetFields();
  };

  const dataSource = React.useMemo(() => {
    const ungroupedKey = '__ungrouped__';
    const groupedConsumers = {};
    allConsumers.forEach((consumer) => {
      const department = consumer.department || '';
      groupedConsumers[department] = groupedConsumers[department] || [];
      groupedConsumers[department].push(consumer);
    });

    const departmentSet = new Set(allDepartments);
    allConsumers.forEach((consumer) => {
      if (consumer.department) {
        departmentSet.add(consumer.department);
      }
    });
    if (groupedConsumers['']?.length) {
      departmentSet.add(ungroupedKey);
    }

    const normalizedKeyword = keyword.trim().toLowerCase();
    const normalizedDepartmentKeyword = departmentKeyword.trim().toLowerCase();
    const normalizedKeySearch = keySearch.trim().toLowerCase();

    return Array.from(departmentSet)
      .sort((i1, i2) => i1.localeCompare(i2))
      .map((department): OrganizationRow | null => {
        const rawDepartment = department === ungroupedKey ? '' : department;
        const departmentLabel = rawDepartment || t('consumer.ungrouped');
        const users = (groupedConsumers[rawDepartment] || [])
          .filter((item) => {
            if (normalizedDepartmentKeyword && !departmentLabel.toLowerCase().includes(normalizedDepartmentKeyword)) {
              return false;
            }
            if (normalizedKeyword && !item.name.toLowerCase().includes(normalizedKeyword)) {
              return false;
            }
            const keySearchText = extractMaskedKeys(item.credentials).join(' ').toLowerCase();
            if (normalizedKeySearch && !keySearchText.includes(normalizedKeySearch)) {
              return false;
            }
            return true;
          })
          .sort((i1, i2) => i1.name.localeCompare(i2.name));

        if (!users.length && (normalizedKeyword || normalizedKeySearch)) {
          return null;
        }
        if (!users.length && normalizedDepartmentKeyword && !departmentLabel.toLowerCase().includes(normalizedDepartmentKeyword)) {
          return null;
        }

        return {
          key: `department-${rawDepartment || ungroupedKey}`,
          rowType: 'department',
          name: departmentLabel,
          department: rawDepartment,
          memberCount: users.length,
          children: users.map((consumer) => ({
            key: `user-${consumer.name}`,
            rowType: 'user',
            name: consumer.name,
            department: consumer.department,
            credentials: consumer.credentials,
            consumer,
          })),
        };
      })
      .filter(Boolean);
  }, [allConsumers, allDepartments, departmentKeyword, keySearch, keyword]);

  return (
    <PageContainer>
      <Form
        form={form}
        style={{
          background: '#fff',
          padding: '24px',
          marginBottom: 16,
        }}
        layout="inline"
      >
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space wrap size={24}>
            <Form.Item name="departmentKeyword" label={t('consumer.columns.department')} style={{ marginBottom: 0 }}>
              <Input
                placeholder={t('consumer.consumerForm.departmentPlaceholder')}
                value={departmentKeyword}
                onChange={(e) => setDepartmentKeyword(e.target.value)}
                allowClear
              />
            </Form.Item>
            <Form.Item name="keyword" label={t('consumer.columns.name')} style={{ marginBottom: 0 }}>
              <Input
                placeholder={t('consumer.columns.name')}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                allowClear
              />
            </Form.Item>
            <Form.Item name="keySearch" label={t('consumer.key')} style={{ marginBottom: 0 }}>
              <Input
                placeholder={t('consumer.key')}
                value={keySearch}
                onChange={(e) => setKeySearch(e.target.value)}
                allowClear
              />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0 }}>
              <Space>
                <Button onClick={handleReset}>{t('misc.reset')}</Button>
              </Space>
            </Form.Item>
          </Space>
          <Space>
            <Button onClick={onOpenInviteCodeManager}>{t('consumer.inviteCode.manage')}</Button>
            <Button
              onClick={() => setOpenDepartmentModal(true)}
            >
              {t('consumer.createDepartment')}
            </Button>
            <Button
              type="primary"
              onClick={() => onShowDrawer()}
            >
              {t('consumer.create')}
            </Button>
            <Button
              icon={<RedoOutlined />}
              onClick={() => {
                refresh();
                loadDepartments();
              }}
            />
          </Space>
        </Space>
      </Form>
      <Table
        loading={loading || departmentsLoading}
        dataSource={dataSource}
        columns={columns}
        pagination={false}
        defaultExpandAllRows
        expandable={{ defaultExpandAllRows: true }}
        locale={{ emptyText: t('mcp.detail.noData') }}
      />
      <Drawer
        title={t(currentConsumer ? "consumer.edit" : "consumer.create")}
        placement="right"
        width={660}
        onClose={handleDrawerCancel}
        open={openDrawer}
        extra={
          <Space>
            <Button onClick={handleDrawerCancel}>{t('misc.cancel')}</Button>
            <Button type="primary" onClick={handleDrawerOK}>
              {t('misc.confirm')}
            </Button>
          </Space>
        }
      >
        <ConsumerForm
          ref={formRef}
          value={currentConsumer}
          departments={allDepartments}
          presetDepartment={presetDepartment}
        />
      </Drawer>
      <Modal
        title={<div><ExclamationCircleOutlined style={{ color: '#ffde5c', marginRight: 8 }} />{t('misc.delete')}</div>}
        open={openModal}
        onOk={handleModalOk}
        confirmLoading={confirmLoading}
        onCancel={handleModalCancel}
        cancelText={t('misc.cancel')}
        okText={t('misc.confirm')}
      >
        <p>
          <Trans t={t} i18nKey="consumer.deleteConfirmation">
            确定删除 <span style={{ color: '#0070cc' }}>{{ currentConsumerName: (currentConsumer && currentConsumer.name) || '' }}</span> 吗？
          </Trans>
        </p>
      </Modal>
      <Modal
        title={t('consumer.createDepartment')}
        open={openDepartmentModal}
        onOk={handleDepartmentModalOk}
        confirmLoading={departmentConfirmLoading}
        onCancel={handleDepartmentModalCancel}
        cancelText={t('misc.cancel')}
        okText={t('misc.confirm')}
      >
        <Form form={departmentForm} layout="vertical">
          <Form.Item
            label={t('consumer.departmentForm.name')}
            name="name"
            rules={[{ required: true, message: t('consumer.departmentForm.nameRequired') || '' }]}
          >
            <Input
              showCount
              allowClear
              maxLength={63}
              placeholder={t('consumer.departmentForm.namePlaceholder') || ''}
            />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={t('consumer.inviteCode.manage')}
        open={openInviteCodeModal}
        onCancel={() => setOpenInviteCodeModal(false)}
        footer={null}
        width={920}
      >
        <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }} wrap>
          <Space>
            <Select
              allowClear
              style={{ width: 220 }}
              placeholder={t('consumer.inviteCode.statusFilterPlaceholder') || ''}
              value={inviteStatusFilter}
              onChange={(value) => {
                setInviteStatusFilter(value);
                loadInviteCodeList(value);
              }}
              options={[
                { value: 'active', label: t('misc.enabled') },
                { value: 'disabled', label: t('misc.disabled') },
              ]}
            />
            <Button icon={<RedoOutlined />} onClick={() => loadInviteCodeList(inviteStatusFilter)} />
          </Space>
          <Button type="primary" onClick={onCreateInviteCode}>
            {t('consumer.inviteCode.create')}
          </Button>
        </Space>
        <Table
          rowKey="inviteCode"
          loading={inviteCodeLoading}
          dataSource={inviteCodes}
          pagination={false}
          locale={{ emptyText: t('mcp.detail.noData') }}
          columns={[
            {
              title: t('consumer.inviteCode.columns.code'),
              dataIndex: 'inviteCode',
              key: 'inviteCode',
              render: (value: string) => <Text copyable>{value}</Text>,
            },
            {
              title: t('consumer.inviteCode.columns.status'),
              dataIndex: 'status',
              key: 'status',
              width: 120,
              render: (value: string) => renderInviteStatus(value),
            },
            {
              title: t('consumer.inviteCode.columns.expiresAt'),
              dataIndex: 'expiresAt',
              key: 'expiresAt',
              width: 180,
              render: (value: string) => formatDateTime(value),
            },
            {
              title: t('consumer.inviteCode.columns.usedBy'),
              dataIndex: 'usedByConsumer',
              key: 'usedByConsumer',
              width: 140,
              render: (value: string) => value || '-',
            },
            {
              title: t('consumer.inviteCode.columns.usedAt'),
              dataIndex: 'usedAt',
              key: 'usedAt',
              width: 180,
              render: (value: string) => formatDateTime(value),
            },
            {
              title: t('consumer.inviteCode.columns.createdAt'),
              dataIndex: 'createdAt',
              key: 'createdAt',
              width: 180,
              render: (value: string) => formatDateTime(value),
            },
            {
              title: t('misc.actions'),
              dataIndex: 'action',
              key: 'action',
              width: 100,
              render: (_, record: InviteCodeRecord) => {
                const status = (record.status || '').toLowerCase();
                if (status === 'active') {
                  return <a onClick={() => onDisableInviteCode(record.inviteCode)}>{t('consumer.inviteCode.disable')}</a>;
                }
                if (status === 'disabled') {
                  return <a onClick={() => onEnableInviteCode(record.inviteCode)}>{t('consumer.inviteCode.enable')}</a>;
                }
                return '-';
              },
            },
          ]}
        />
      </Modal>
    </PageContainer>
  );
};

export default ConsumerList;
