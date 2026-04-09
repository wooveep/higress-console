import { OrgAccountMutation, OrgAccountRecord, OrgDepartmentNode } from '@/interfaces/org';
import { Form, Input, Select, TreeSelect } from 'antd';
import React, { forwardRef, useEffect, useImperativeHandle, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  value?: OrgAccountRecord | null;
  departments?: OrgDepartmentNode[];
  accounts?: OrgAccountRecord[];
  presetDepartmentId?: string;
}

export interface ConsumerFormRef {
  reset: () => void;
  handleSubmit: () => Promise<OrgAccountMutation>;
}

const buildDepartmentTree = (departments: OrgDepartmentNode[] = []) => {
  return departments.map((department) => ({
    title: department.name,
    value: department.departmentId,
    key: department.departmentId,
    children: buildDepartmentTree(department.children || []),
  }));
};

const ConsumerForm = forwardRef<ConsumerFormRef, Props>((props, ref) => {
  const { t } = useTranslation();
  const { value, departments = [], accounts = [], presetDepartmentId } = props;
  const [form] = Form.useForm();

  const departmentTree = useMemo(() => buildDepartmentTree(departments), [departments]);
  const parentAccountOptions = useMemo(() => {
    return accounts
      .filter((account) => account.consumerName && account.consumerName !== value?.consumerName)
      .map((account) => ({
        label: `${account.consumerName}${account.displayName ? ` / ${account.displayName}` : ''}`,
        value: account.consumerName,
      }));
  }, [accounts, value?.consumerName]);

  useEffect(() => {
    if (value) {
      form.setFieldsValue({
        consumerName: value.consumerName,
        displayName: value.displayName,
        email: value.email,
        userLevel: value.userLevel || 'normal',
        departmentId: value.departmentId,
        parentConsumerName: value.isDepartmentAdmin ? undefined : value.parentConsumerName,
        password: undefined,
      });
      return;
    }
    form.resetFields();
    form.setFieldsValue({
      userLevel: 'normal',
      departmentId: presetDepartmentId || undefined,
    });
  }, [form, presetDepartmentId, value]);

  useImperativeHandle(ref, () => ({
    reset: () => {
      form.resetFields();
    },
    handleSubmit: async () => {
      const values = await form.validateFields();
      return {
        consumerName: values.consumerName,
        displayName: values.displayName,
        email: values.email,
        userLevel: values.userLevel,
        password: values.password,
        departmentId: values.departmentId,
        parentConsumerName: values.parentConsumerName,
      };
    },
  }));

  return (
    <Form form={form} layout="vertical">
      <Form.Item
        label="账号名"
        required
        name="consumerName"
        rules={[{ required: true, message: t('consumer.consumerForm.nameRequired') || '' }]}
      >
        <Input
          showCount
          allowClear
          maxLength={63}
          placeholder={t('consumer.consumerForm.namePlaceholder') || ''}
          disabled={!!value}
        />
      </Form.Item>
      <Form.Item label="显示名" name="displayName">
        <Input showCount allowClear maxLength={63} placeholder="可选，默认与账号名一致" />
      </Form.Item>
      <Form.Item label="邮箱" name="email">
        <Input showCount allowClear maxLength={128} placeholder="可选" />
      </Form.Item>
      <Form.Item label="所属部门" name="departmentId">
        <TreeSelect
          allowClear
          treeDefaultExpandAll
          treeData={departmentTree}
          placeholder="未分配"
        />
      </Form.Item>
      <Form.Item label="父账号" name="parentConsumerName">
        <Select
          allowClear
          showSearch
          options={parentAccountOptions}
          placeholder={value?.isDepartmentAdmin ? '部门管理员默认无父账号' : '留空则默认归属部门管理员'}
          optionFilterProp="label"
          disabled={!!value?.isDepartmentAdmin}
        />
      </Form.Item>
      <Form.Item
        label={t('consumer.consumerForm.portalUserLevel')}
        name="userLevel"
        rules={[{ required: true, message: t('consumer.consumerForm.portalUserLevelRequired') || '' }]}
      >
        <Select placeholder={t('consumer.consumerForm.portalUserLevelPlaceholder') || ''}>
          <Select.Option value="normal">{t('consumer.userLevel.normal')}</Select.Option>
          <Select.Option value="plus">{t('consumer.userLevel.plus')}</Select.Option>
          <Select.Option value="pro">{t('consumer.userLevel.pro')}</Select.Option>
          <Select.Option value="ultra">{t('consumer.userLevel.ultra')}</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label="Portal密码" name="password">
        <Input.Password placeholder={value ? '留空则不修改密码' : '留空将由系统生成临时密码'} />
      </Form.Item>
    </Form>
  );
});

export default ConsumerForm;
