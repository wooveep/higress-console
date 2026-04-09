/* eslint-disable */
// @ts-nocheck
import { InviteCodeRecord } from '@/interfaces/consumer';
import { OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import {
  createInviteCode,
  deleteConsumer,
  disableInviteCode,
  enableInviteCode,
  listInviteCodes,
  resetConsumerPassword,
} from '@/services/consumer';
import {
  createOrgAccount,
  createOrgDepartment,
  deleteOrgDepartment,
  downloadOrgTemplate,
  exportOrgWorkbook,
  importOrgWorkbook,
  listOrgAccounts,
  listOrgDepartmentsTree,
  moveOrgDepartment,
  updateOrgAccount,
  updateOrgAccountStatus,
  updateOrgDepartment,
} from '@/services/organization';
import { formatDateTimeDisplay } from '@/utils/time';
import { ApartmentOutlined, BranchesOutlined, PlusOutlined, RedoOutlined, UserOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-layout';
import { useRequest } from 'ahooks';
import {
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  message,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  TreeSelect,
  Tree,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import ConsumerForm, { ConsumerFormRef } from './components/ConsumerForm';

const { Text } = Typography;
const BUILTIN_ADMIN_CONSUMER = 'administrator';

type DepartmentModalMode = 'create' | 'rename' | 'move' | null;

const ConsumerList: React.FC = () => {
  const { t } = useTranslation();
  const formRef = useRef<ConsumerFormRef>(null);
  const importInputRef = useRef<HTMLInputElement>(null);
  const [departmentForm] = Form.useForm();

  const [departmentTree, setDepartmentTree] = useState<OrgDepartmentNode[]>([]);
  const [accounts, setAccounts] = useState<OrgAccountRecord[]>([]);
  const [keyword, setKeyword] = useState('');
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<string>();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [currentAccount, setCurrentAccount] = useState<OrgAccountRecord | null>(null);
  const [presetDepartmentId, setPresetDepartmentId] = useState<string>();
  const [departmentModalMode, setDepartmentModalMode] = useState<DepartmentModalMode>(null);
  const [departmentModalOpen, setDepartmentModalOpen] = useState(false);
  const [departmentConfirmLoading, setDepartmentConfirmLoading] = useState(false);
  const [currentDepartment, setCurrentDepartment] = useState<OrgDepartmentNode | null>(null);
  const [openInviteCodeModal, setOpenInviteCodeModal] = useState(false);
  const [inviteCodes, setInviteCodes] = useState<InviteCodeRecord[]>([]);
  const [inviteCodeLoading, setInviteCodeLoading] = useState(false);
  const [inviteStatusFilter, setInviteStatusFilter] = useState<string | undefined>(undefined);

  const formatDateTime = (value?: any) => formatDateTimeDisplay(value);

  const { loading, run: loadData } = useRequest(async () => {
    const [departments, organizationAccounts] = await Promise.all([
      listOrgDepartmentsTree(),
      listOrgAccounts(),
    ]);
    return { departments, organizationAccounts };
  }, {
    manual: true,
    onSuccess: (result) => {
      setDepartmentTree(result?.departments || []);
      setAccounts(result?.organizationAccounts || []);
    },
  });

  useEffect(() => {
    loadData();
  }, []);

  const departmentMap = useMemo(() => {
    const result: Record<string, OrgDepartmentNode> = {};
    const walk = (nodes: OrgDepartmentNode[]) => {
      nodes.forEach((node) => {
        result[node.departmentId] = node;
        walk(node.children || []);
      });
    };
    walk(departmentTree);
    return result;
  }, [departmentTree]);

  const collectSubtreeIds = (departmentId?: string): string[] => {
    if (!departmentId) {
      return [];
    }
    const result: string[] = [];
    const walk = (node?: OrgDepartmentNode) => {
      if (!node) {
        return;
      }
      result.push(node.departmentId);
      (node.children || []).forEach(walk);
    };
    walk(departmentMap[departmentId]);
    return result;
  };

  const selectedDepartment = selectedDepartmentId ? departmentMap[selectedDepartmentId] : undefined;
  const selectedDepartmentScope = useMemo(
    () => new Set(collectSubtreeIds(selectedDepartmentId)),
    [selectedDepartmentId, departmentMap],
  );

  const filteredAccounts = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    return accounts.filter((account) => {
      if (!account?.consumerName || account.consumerName === BUILTIN_ADMIN_CONSUMER) {
        return false;
      }
      if (selectedDepartmentScope.size && !selectedDepartmentScope.has(account.departmentId || '')) {
        return false;
      }
      if (!normalizedKeyword) {
        return true;
      }
      const searchable = [
        account.consumerName,
        account.displayName,
        account.email,
        account.departmentPath,
        account.parentConsumerName,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return searchable.includes(normalizedKeyword);
    });
  }, [accounts, keyword, selectedDepartmentScope]);

  const departmentTreeData = useMemo(() => {
    const build = (nodes: OrgDepartmentNode[] = []) =>
      nodes.map((node) => ({
        key: node.departmentId,
        title: (
          <Space size={8}>
            <ApartmentOutlined />
            <span>{node.name}</span>
            <Text type="secondary">({node.memberCount || 0})</Text>
          </Space>
        ),
        children: build(node.children || []),
      }));
    return build(departmentTree);
  }, [departmentTree]);

  const departmentSelectTree = useMemo(() => {
    const build = (nodes: OrgDepartmentNode[] = []) =>
      nodes.map((node) => ({
        title: node.name,
        value: node.departmentId,
        key: node.departmentId,
        children: build(node.children || []),
      }));
    return build(departmentTree);
  }, [departmentTree]);

  const availableMoveTree = useMemo(() => {
    if (!currentDepartment) {
      return departmentSelectTree;
    }
    const excluded = new Set(collectSubtreeIds(currentDepartment.departmentId));
    const filterTree = (nodes: any[] = []) =>
      nodes
        .filter((node) => !excluded.has(node.value))
        .map((node) => ({
          ...node,
          children: filterTree(node.children || []),
        }));
    return filterTree(departmentSelectTree);
  }, [currentDepartment, departmentSelectTree]);

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

  const renderStatus = (status?: string) => {
    const normalized = (status || 'pending').toLowerCase();
    if (normalized === 'active') {
      return <Tag color="success">{t('misc.enabled')}</Tag>;
    }
    if (normalized === 'disabled') {
      return <Tag color="error">{t('misc.disabled')}</Tag>;
    }
    return <Tag>{t('consumer.portalStatus.pending')}</Tag>;
  };

  const onRefresh = async () => {
    await loadData();
  };

  const saveWorkbook = (data: ArrayBuffer, filename: string) => {
    const blob = new Blob([data], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  };

  const onCreateAccount = (departmentId?: string) => {
    setCurrentAccount(null);
    setPresetDepartmentId(departmentId);
    setDrawerOpen(true);
  };

  const onEditAccount = (account: OrgAccountRecord) => {
    setCurrentAccount(account);
    setPresetDepartmentId(undefined);
    setDrawerOpen(true);
  };

  const onSubmitAccount = async () => {
    try {
      const payload = await formRef.current?.handleSubmit();
      if (!payload) {
        return;
      }
      if (currentAccount?.consumerName) {
        await updateOrgAccount(currentAccount.consumerName, payload);
        message.success(t('misc.edit'));
      } else {
        const result = await createOrgAccount(payload);
        message.success(t('consumer.create'));
        if (result?.tempPassword) {
          Modal.info({
            title: '账号创建成功',
            width: 520,
            content: (
              <Space direction="vertical" style={{ width: '100%' }}>
                <span>系统已为新账号生成默认密码，请妥善保存。</span>
                <Input.TextArea value={result.tempPassword} autoSize={{ minRows: 2, maxRows: 4 }} readOnly />
              </Space>
            ),
          });
        }
      }
      setDrawerOpen(false);
      setCurrentAccount(null);
      setPresetDepartmentId(undefined);
      await loadData();
    } catch (error) {}
  };

  const onToggleAccountStatus = async (account: OrgAccountRecord, status: 'active' | 'disabled') => {
    try {
      await updateOrgAccountStatus(account.consumerName, status);
      message.success(status === 'active' ? t('consumer.enableSuccess') : t('consumer.disableSuccess'));
      await loadData();
    } catch (error) {
      message.error(t('consumer.statusUpdateFailed'));
    }
  };

  const onResetConsumerPassword = async (account: OrgAccountRecord) => {
    try {
      const result = await resetConsumerPassword(account.consumerName);
      message.success(t('consumer.resetPasswordSuccess'));
      Modal.info({
        title: t('consumer.resetPasswordTitle'),
        width: 520,
        content: (
          <Space direction="vertical" style={{ width: '100%' }}>
            <span>{t('consumer.resetPasswordHint', { name: account.consumerName })}</span>
            <Input.TextArea value={result.tempPassword} autoSize={{ minRows: 2, maxRows: 4 }} readOnly />
          </Space>
        ),
      });
    } catch (error) {
      message.error(t('consumer.resetPasswordFailed'));
    }
  };

  const onDeleteAccount = (account: OrgAccountRecord) => {
    if ((account.status || '').toLowerCase() === 'active') {
      message.warning('请先禁用用户，再执行删除。');
      return;
    }
    Modal.confirm({
      title: t('misc.delete'),
      content: `是否确认删除账号 ${account.consumerName}？`,
      onOk: async () => {
        await deleteConsumer(account.consumerName);
        message.success(t('consumer.deleteSuccess'));
        await loadData();
      },
    });
  };

  const openDepartmentModal = (mode: DepartmentModalMode) => {
    setDepartmentModalMode(mode);
    setDepartmentModalOpen(true);
    if (mode === 'create') {
      departmentForm.setFieldsValue({
        name: '',
        parentDepartmentId: currentDepartment?.departmentId,
        adminUserLevel: 'normal',
      });
      return;
    }
    if (mode === 'rename') {
      departmentForm.setFieldsValue({
        name: currentDepartment?.name,
        adminConsumerName: currentDepartment?.adminConsumerName,
      });
      return;
    }
    if (mode === 'move') {
      departmentForm.setFieldsValue({
        parentDepartmentId: currentDepartment?.parentDepartmentId,
      });
    }
  };

  const onSubmitDepartment = async () => {
    try {
      const values = await departmentForm.validateFields();
      setDepartmentConfirmLoading(true);
      if (departmentModalMode === 'create') {
        await createOrgDepartment({
          name: values.name,
          parentDepartmentId: values.parentDepartmentId,
          admin: {
            consumerName: values.adminConsumerName,
            displayName: values.adminDisplayName,
            email: values.adminEmail,
            userLevel: values.adminUserLevel,
            password: values.adminPassword,
          },
        });
        message.success('部门已创建');
      } else if (departmentModalMode === 'rename' && currentDepartment?.departmentId) {
        await updateOrgDepartment(currentDepartment.departmentId, {
          name: values.name,
          adminConsumerName: values.adminConsumerName,
        });
        message.success('部门信息已更新');
      } else if (departmentModalMode === 'move' && currentDepartment?.departmentId) {
        await moveOrgDepartment(currentDepartment.departmentId, {
          parentDepartmentId: values.parentDepartmentId,
        });
        message.success('部门已移动');
      }
      setDepartmentModalOpen(false);
      departmentForm.resetFields();
      await loadData();
    } catch (error) {
    } finally {
      setDepartmentConfirmLoading(false);
    }
  };

  const onDownloadTemplate = async () => {
    const data = await downloadOrgTemplate();
    saveWorkbook(data, 'organization-template.xlsx');
  };

  const onExportOrganization = async () => {
    const data = await exportOrgWorkbook();
    saveWorkbook(data, 'organization-export.xlsx');
  };

  const onImportOrganization = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    const result = await importOrgWorkbook(file);
    message.success(
      `导入完成：部门新增 ${result.createdDepartments}，部门更新 ${result.updatedDepartments}，账号新增 ${result.createdAccounts}，账号更新 ${result.updatedAccounts}`,
    );
    await loadData();
  };

  const onDeleteDepartment = () => {
    if (!currentDepartment?.departmentId) {
      return;
    }
    Modal.confirm({
      title: '删除部门',
      content: `确认删除部门 ${currentDepartment.name} 吗？`,
      onOk: async () => {
        await deleteOrgDepartment(currentDepartment.departmentId);
        if (selectedDepartmentId === currentDepartment.departmentId) {
          setSelectedDepartmentId(undefined);
        }
        setCurrentDepartment(null);
        message.success('部门已删除');
        await loadData();
      },
    });
  };

  const loadInviteCodeList = async (status?: string) => {
    setInviteCodeLoading(true);
    try {
      const data = await listInviteCodes({ status });
      setInviteCodes(data || []);
    } finally {
      setInviteCodeLoading(false);
    }
  };

  const onOpenInviteCodeManager = async () => {
    setOpenInviteCodeModal(true);
    await loadInviteCodeList(inviteStatusFilter);
  };

  const onCreateInviteCode = async () => {
    try {
      const result = await createInviteCode();
      message.success(t('consumer.inviteCode.createSuccess'));
      Modal.info({
        title: t('consumer.inviteCode.codeGeneratedTitle'),
        width: 520,
        content: (
          <Space direction="vertical" style={{ width: '100%' }}>
            <span>{t('consumer.inviteCode.codeGeneratedHint')}</span>
            <Input.TextArea value={result.inviteCode} autoSize={{ minRows: 2, maxRows: 4 }} readOnly />
          </Space>
        ),
      });
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.createFailed'));
    }
  };

  const onDisableInviteCode = async (inviteCode: string) => {
    try {
      await disableInviteCode(inviteCode);
      message.success(t('consumer.inviteCode.disableSuccess'));
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.disableFailed'));
    }
  };

  const onEnableInviteCode = async (inviteCode: string) => {
    try {
      await enableInviteCode(inviteCode);
      message.success(t('consumer.inviteCode.enableSuccess'));
      await loadInviteCodeList(inviteStatusFilter);
    } catch (error) {
      message.error(t('consumer.inviteCode.enableFailed'));
    }
  };

  const columns = [
    {
      title: '账号',
      dataIndex: 'consumerName',
      key: 'consumerName',
          render: (_, record: OrgAccountRecord) => (
        <Space direction="vertical" size={0}>
          <Space>
            <UserOutlined />
            <Text strong>{record.consumerName}</Text>
            {record.isDepartmentAdmin ? <Tag color="gold">部门管理员</Tag> : null}
          </Space>
          {record.displayName ? <Text type="secondary">{record.displayName}</Text> : null}
        </Space>
      ),
    },
    {
      title: '所属部门',
      dataIndex: 'departmentPath',
      key: 'departmentPath',
      render: (value) => value || <Text type="secondary">未分配</Text>,
    },
    {
      title: '父账号',
      dataIndex: 'parentConsumerName',
      key: 'parentConsumerName',
      render: (value) => value || <Text type="secondary">无</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value) => renderStatus(value),
    },
    {
      title: t('consumer.columns.userLevel'),
      dataIndex: 'userLevel',
      key: 'userLevel',
      width: 140,
      render: (value) => renderUserLevel(value),
    },
    {
      title: '最近登录',
      dataIndex: 'lastLoginAt',
      key: 'lastLoginAt',
      width: 180,
      render: (value) => formatDateTime(value) || '-',
    },
    {
      title: t('misc.actions'),
      key: 'actions',
      width: 260,
      render: (_, record: OrgAccountRecord) => (
        <Space size="small" wrap>
          <a onClick={() => onEditAccount(record)}>{t('misc.edit')}</a>
          {(record.status || '').toLowerCase() === 'active'
            ? <a onClick={() => onToggleAccountStatus(record, 'disabled')}>{t('consumer.disable')}</a>
            : <a onClick={() => onToggleAccountStatus(record, 'active')}>{t('consumer.enable')}</a>}
          <a onClick={() => onResetConsumerPassword(record)}>{t('consumer.resetPassword')}</a>
          <a onClick={() => onDeleteAccount(record)}>{t('misc.delete')}</a>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer>
      <Space direction="vertical" style={{ width: '100%' }} size={16}>
        <Card>
          <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
            <Space wrap>
              <Input
                allowClear
                value={keyword}
                placeholder="搜索账号、邮箱、部门、父账号"
                style={{ width: 280 }}
                onChange={(e) => setKeyword(e.target.value)}
              />
              <Button icon={<RedoOutlined />} onClick={onRefresh}>{t('misc.refresh')}</Button>
            </Space>
            <Space wrap>
              <Button onClick={onOpenInviteCodeManager}>{t('consumer.inviteCode.manage')}</Button>
              <Button onClick={onDownloadTemplate}>下载模板</Button>
              <Button onClick={onExportOrganization}>导出组织</Button>
              <Button onClick={() => importInputRef.current?.click()}>导入组织</Button>
              <Button icon={<ApartmentOutlined />} onClick={() => openDepartmentModal('create')}>
                新建部门
              </Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => onCreateAccount(selectedDepartmentId)}>
                新建账号
              </Button>
            </Space>
          </Space>
        </Card>

        <div style={{ display: 'grid', gridTemplateColumns: '320px 1fr', gap: 16 }}>
          <Card
            title="组织树"
            extra={
              <Button type="link" onClick={() => {
                setSelectedDepartmentId(undefined);
                setCurrentDepartment(null);
              }}>
                全部账号
              </Button>
            }
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              {departmentTreeData.length ? (
                <Tree
                  selectedKeys={selectedDepartmentId ? [selectedDepartmentId] : []}
                  treeData={departmentTreeData}
                  onSelect={(keys, info) => {
                    const nextDepartmentId = keys?.[0] as string | undefined;
                    setSelectedDepartmentId(nextDepartmentId);
                    setCurrentDepartment(nextDepartmentId ? departmentMap[nextDepartmentId] : null);
                  }}
                />
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无部门" />
              )}
              {selectedDepartment ? (
                <Card size="small" title={selectedDepartment.name}>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Text type="secondary">
                      部门管理员：{selectedDepartment.adminDisplayName || selectedDepartment.adminConsumerName || '未设置'}
                    </Text>
                    <Space wrap>
                    <Button size="small" onClick={() => onCreateAccount(selectedDepartment.departmentId)}>
                      新建账号
                    </Button>
                    <Button size="small" onClick={() => openDepartmentModal('create')}>
                      新建子部门
                    </Button>
                    <Button size="small" onClick={() => openDepartmentModal('rename')}>
                      编辑部门
                    </Button>
                    <Button size="small" icon={<BranchesOutlined />} onClick={() => openDepartmentModal('move')}>
                      移动
                    </Button>
                    <Button size="small" danger onClick={onDeleteDepartment}>
                      删除
                    </Button>
                    </Space>
                  </Space>
                </Card>
              ) : null}
            </Space>
          </Card>

          <Card title={`账号列表 (${filteredAccounts.length})`}>
            <Table
              rowKey="consumerName"
              loading={loading}
              dataSource={filteredAccounts}
              columns={columns}
              pagination={{ pageSize: 10, showSizeChanger: true }}
            />
          </Card>
        </div>
      </Space>

      <Drawer
        title={currentAccount ? t('misc.edit') : t('consumer.create')}
        width={520}
        open={drawerOpen}
        footer={(
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => {
              setDrawerOpen(false);
              setCurrentAccount(null);
              setPresetDepartmentId(undefined);
            }}>
              {t('misc.cancel')}
            </Button>
            <Button type="primary" onClick={onSubmitAccount}>
              {t('misc.save')}
            </Button>
          </Space>
        )}
        onClose={() => {
          setDrawerOpen(false);
          setCurrentAccount(null);
          setPresetDepartmentId(undefined);
        }}
        destroyOnClose
      >
        <ConsumerForm
          ref={formRef}
          value={currentAccount}
          departments={departmentTree}
          accounts={accounts}
          presetDepartmentId={presetDepartmentId}
        />
      </Drawer>

      <Modal
        title={
          departmentModalMode === 'create'
            ? '新建部门'
            : departmentModalMode === 'rename'
              ? '重命名部门'
              : '移动部门'
        }
        open={departmentModalOpen}
        confirmLoading={departmentConfirmLoading}
        destroyOnClose
        onCancel={() => {
          setDepartmentModalOpen(false);
          departmentForm.resetFields();
        }}
        onOk={onSubmitDepartment}
      >
        <Form form={departmentForm} layout="vertical">
          {departmentModalMode !== 'move' ? (
            <Form.Item
              label="部门名称"
              name="name"
              rules={departmentModalMode === 'rename' ? [] : [{ required: true, message: '请输入部门名称' }]}
            >
              <Input allowClear maxLength={128} placeholder="例如：华东销售部" />
            </Form.Item>
          ) : null}
          {departmentModalMode === 'create' ? (
            <>
              <Typography.Title level={5} style={{ marginBottom: 12 }}>部门管理员</Typography.Title>
              <Form.Item
                label="管理员账号名"
                name="adminConsumerName"
                rules={[{ required: true, message: '请输入管理员账号名' }]}
              >
                <Input allowClear maxLength={63} placeholder="例如：sales-east-admin" />
              </Form.Item>
              <Form.Item label="显示名" name="adminDisplayName">
                <Input allowClear maxLength={63} placeholder="可选，默认与账号名一致" />
              </Form.Item>
              <Form.Item label="邮箱" name="adminEmail">
                <Input allowClear maxLength={128} placeholder="可选" />
              </Form.Item>
              <Form.Item label="用户等级" name="adminUserLevel" initialValue="normal">
                <Select>
                  <Select.Option value="normal">{t('consumer.userLevel.normal')}</Select.Option>
                  <Select.Option value="plus">{t('consumer.userLevel.plus')}</Select.Option>
                  <Select.Option value="pro">{t('consumer.userLevel.pro')}</Select.Option>
                  <Select.Option value="ultra">{t('consumer.userLevel.ultra')}</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item label="初始密码" name="adminPassword">
                <Input.Password placeholder="留空时使用系统默认密码" />
              </Form.Item>
            </>
          ) : null}
          {departmentModalMode === 'rename' ? (
            <Form.Item label="部门管理员" name="adminConsumerName">
              <Select
                allowClear
                showSearch
                options={accounts.map((account) => ({
                  label: `${account.consumerName}${account.displayName ? ` / ${account.displayName}` : ''}`,
                  value: account.consumerName,
                }))}
                placeholder="留空则保持当前管理员"
                optionFilterProp="label"
              />
            </Form.Item>
          ) : null}
          {departmentModalMode !== 'rename' ? (
            <Form.Item label="父部门" name="parentDepartmentId">
              <TreeSelect
                allowClear
                treeDefaultExpandAll
                placeholder={departmentModalMode === 'create' ? '默认创建在当前部门下' : '移动到顶级时留空'}
                treeData={availableMoveTree}
              />
            </Form.Item>
          ) : null}
        </Form>
      </Modal>
      <input
        ref={importInputRef}
        type="file"
        accept=".xlsx"
        style={{ display: 'none' }}
        onChange={onImportOrganization}
      />

      <Modal
        title={t('consumer.inviteCode.manage')}
        open={openInviteCodeModal}
        footer={null}
        width={920}
        onCancel={() => setOpenInviteCodeModal(false)}
      >
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
            <Select
              allowClear
              style={{ width: 220 }}
              value={inviteStatusFilter}
              placeholder={t('consumer.inviteCode.statusFilterPlaceholder') || ''}
              onChange={async (value) => {
                setInviteStatusFilter(value);
                await loadInviteCodeList(value);
              }}
            >
              <Select.Option value="active">{t('misc.enabled')}</Select.Option>
              <Select.Option value="disabled">{t('misc.disabled')}</Select.Option>
              <Select.Option value="used">{t('consumer.inviteCode.status.used')}</Select.Option>
            </Select>
            <Button type="primary" onClick={onCreateInviteCode}>
              {t('consumer.inviteCode.create')}
            </Button>
          </Space>
          <Table
            rowKey="inviteCode"
            loading={inviteCodeLoading}
            dataSource={inviteCodes}
            pagination={{ pageSize: 8, showSizeChanger: false }}
            columns={[
              { title: t('consumer.inviteCode.columns.code'), dataIndex: 'inviteCode', key: 'inviteCode' },
              {
                title: t('consumer.inviteCode.columns.status'),
                dataIndex: 'status',
                key: 'status',
                render: (value) => renderInviteStatus(value),
              },
              {
                title: t('consumer.inviteCode.columns.expiresAt'),
                dataIndex: 'expiresAt',
                key: 'expiresAt',
                render: (value) => formatDateTime(value) || '-',
              },
              {
                title: t('consumer.inviteCode.columns.usedBy'),
                dataIndex: 'usedByConsumer',
                key: 'usedByConsumer',
                render: (value) => value || '-',
              },
              {
                title: t('consumer.inviteCode.columns.usedAt'),
                dataIndex: 'usedAt',
                key: 'usedAt',
                render: (value) => formatDateTime(value) || '-',
              },
              {
                title: t('consumer.inviteCode.columns.createdAt'),
                dataIndex: 'createdAt',
                key: 'createdAt',
                render: (value) => formatDateTime(value) || '-',
              },
              {
                title: t('misc.actions'),
                key: 'actions',
                render: (_, record: InviteCodeRecord) => {
                  if ((record.status || '').toLowerCase() === 'active') {
                    return <a onClick={() => onDisableInviteCode(record.inviteCode)}>{t('consumer.inviteCode.disable')}</a>;
                  }
                  if ((record.status || '').toLowerCase() === 'disabled') {
                    return <a onClick={() => onEnableInviteCode(record.inviteCode)}>{t('consumer.inviteCode.enable')}</a>;
                  }
                  return '-';
                },
              },
            ]}
          />
        </Space>
      </Modal>
    </PageContainer>
  );
};

export default ConsumerList;
