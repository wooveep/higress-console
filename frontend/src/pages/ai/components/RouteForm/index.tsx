/* eslint-disable max-lines */
import { AiRoute } from '@/interfaces/ai-route';
import { Consumer, CredentialType } from '@/interfaces/consumer';
import { DEFAULT_DOMAIN, Domain } from '@/interfaces/domain';
import { ModelAsset, ModelAssetBinding } from '@/interfaces/model-asset';
import FactorGroup from '@/pages/route/components/FactorGroup';
import { getGatewayDomains } from '@/services';
import { getConsumers } from '@/services/consumer';
import { getModelAssets } from '@/services/model-asset';
import { MinusCircleOutlined, PlusOutlined, RedoOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import { Alert, Button, Checkbox, Empty, Form, Input, InputNumber, Select, Space, Switch } from 'antd';
import { uniqueId } from 'lodash';
import React, { forwardRef, useEffect, useImperativeHandle, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  expandConsumersByAllowedLevels,
  normalizeUserLevels,
  USER_LEVELS,
} from '@/utils/consumer-level';
import { HistoryButton, ModelMappingEditor, RedoOutlinedBtn } from './Components';
import { modelMapping2String, string2ModelMapping } from './util';

const { Option } = Select;

const AI_ROUTE_BINDING_REFS_CONFIG_KEY = 'higress.io/console-ai-binding-refs';

interface PublishedBindingOption {
  displayName: string;
  modelId: string;
  providerName: string;
  targetModel: string;
  label: string;
}

interface AiRouteFormProps {
  value?: AiRoute | null;
}

export interface AiRouteFormHandle {
  reset: () => void;
  handleSubmit: () => Promise<AiRoute | false>;
}

const toBindingOption = (asset: ModelAsset, binding: ModelAssetBinding): PublishedBindingOption => {
  const displayName = asset.displayName || asset.canonicalName || asset.assetId;
  const modelId = binding.modelId || binding.bindingId;
  const providerName = binding.providerName || '';
  const targetModel = binding.targetModel || '';
  return {
    displayName,
    modelId,
    providerName,
    targetModel,
    label: `${displayName} / ${modelId} / ${providerName} / ${targetModel}`,
  };
};

const AiRouteForm = forwardRef<AiRouteFormHandle, AiRouteFormProps>((props, ref) => {
  const { t } = useTranslation();
  const tr = (key: string) => String(t(key));
  const { value } = props;
  const [form] = Form.useForm();
  const [fallbackConfigEnabled, setFallbackConfigEnabled] = useState(false);
  const [authConfigEnabled, setAuthConfigEnabled] = useState(false);
  const [upstreamsError, setUpstreamsError] = useState<any>(false);
  const [bindingOptions, setBindingOptions] = useState<PublishedBindingOption[]>([]);
  const [bindingsLoaded, setBindingsLoaded] = useState(false);
  const [legacyProviderWarning, setLegacyProviderWarning] = useState('');
  const [consumerList, setConsumerList] = useState<Consumer[]>([]);
  const [domainList, setDomainList] = useState<Domain[]>([]);

  const publishedProviderOptions = useMemo(
    () =>
      Array.from(new Set(bindingOptions.map((item) => item.providerName).filter(Boolean)))
        .sort((left, right) => left.localeCompare(right))
        .map((providerName) => ({ label: providerName, value: providerName })),
    [bindingOptions],
  );
  const publishedProviderSet = useMemo(
    () => new Set(publishedProviderOptions.map((item) => item.value)),
    [publishedProviderOptions],
  );
  const publishedBindingOptionsByProvider = useMemo(
    () =>
      bindingOptions.reduce<Record<string, PublishedBindingOption[]>>((accumulator, item) => {
        if (!item.providerName) {
          return accumulator;
        }
        accumulator[item.providerName] = accumulator[item.providerName] || [];
        accumulator[item.providerName].push(item);
        return accumulator;
      }, {}),
    [bindingOptions],
  );

  const bindingResult = useRequest(getModelAssets, {
    manual: true,
    onSuccess: (result) => {
      const options = ((result || []) as ModelAsset[]).flatMap((asset) => {
        return (asset.bindings || [])
          .filter((binding) => binding.status === 'published')
          .map((binding) => toBindingOption(asset, binding));
      });
      options.sort((left, right) => left.label.localeCompare(right.label));
      setBindingOptions(options);
      setBindingsLoaded(true);
    },
    onError: () => {
      setBindingOptions([]);
      setBindingsLoaded(true);
    },
  });

  const consumerResult = useRequest(getConsumers, {
    manual: true,
    onSuccess: (result) => {
      setConsumerList((result || []) as Consumer[]);
    },
  });

  const domainsResult = useRequest(getGatewayDomains, {
    manual: true,
    onSuccess: (result) => {
      const domains = (result || []) as Domain[];
      setDomainList(domains.filter((item) => item.name !== DEFAULT_DOMAIN));
    },
  });

  const defaultFallbackResponseCodes = ['4xx', '5xx'];

  useEffect(() => {
    bindingResult.run();
    consumerResult.run();
    domainsResult.run();
    return () => {
      setAuthConfigEnabled(false);
      setFallbackConfigEnabled(false);
    };
  }, []);

  useEffect(() => {
    if (!bindingsLoaded) {
      return;
    }
    form.resetFields();
    initForm();
  }, [bindingsLoaded, value]);

  const getOptionsForProvider = (providerName?: string) => {
    if (!providerName) {
      return [];
    }
    return (publishedBindingOptionsByProvider[providerName] || []).map((item, index) => ({
      key: `${providerName}-${item.targetModel}-${item.modelId}-${index}`,
      label: item.targetModel === item.modelId
        ? `${item.targetModel} / ${item.displayName}`
        : `${item.targetModel} / ${item.displayName} / ${item.modelId}`,
      value: item.targetModel,
    }));
  };

  const renderLegacyProviderOption = (providerName?: string) => {
    if (!providerName || publishedProviderSet.has(providerName)) {
      return null;
    }
    return (
      <Select.Option key={providerName} value={providerName}>
        {`${t('aiRoute.routeForm.legacyBindingOption')} / ${providerName}`}
      </Select.Option>
    );
  };

  const initForm = () => {
    const {
      name = '',
      domains,
      pathPredicate = { matchType: 'PRE', matchValue: '/', caseSensitive: false },
      headerPredicates = [],
      urlParamPredicates = [],
      upstreams = [{ provider: '', weight: 100 }],
      modelPredicates,
    } = value || {};
    const authConfigEnabledValue = value?.authConfig?.enabled || false;
    const fallbackConfigEnabledValue = value?.fallbackConfig?.enabled || false;
    const legacyProviders: string[] = [];

    setAuthConfigEnabled(authConfigEnabledValue);
    setFallbackConfigEnabled(fallbackConfigEnabledValue);

    const fallbackInitValues = { fallbackConfig_enabled: fallbackConfigEnabledValue };
    if (fallbackConfigEnabledValue && value?.fallbackConfig?.upstreams) {
      fallbackInitValues.fallbackConfig_responseCodes = value.fallbackConfig.responseCodes?.length
        ? value.fallbackConfig.responseCodes
        : defaultFallbackResponseCodes;
      fallbackInitValues.fallbackConfig_upstreams = value.fallbackConfig.upstreams?.[0]?.provider;
      if (value.fallbackConfig.upstreams?.[0]?.provider && !publishedProviderSet.has(value.fallbackConfig.upstreams?.[0]?.provider)) {
        legacyProviders.push(value.fallbackConfig.upstreams?.[0]?.provider);
      }
      try {
        fallbackInitValues.fallbackConfig_modelNames = modelMapping2String(
          value.fallbackConfig.upstreams?.[0]?.modelMapping,
        );
      } catch (error) {
        fallbackInitValues.fallbackConfig_modelNames = '';
      }
    }

    let normalizedDomains: string[] = [];
    if (Array.isArray(domains)) {
      normalizedDomains = domains;
    } else if (domains) {
      normalizedDomains = [domains];
    }
    const initValues: Record<string, any> = {
      name,
      domains: normalizedDomains.filter(Boolean),
      pathPredicate: Object.assign({ ...pathPredicate }, { ignoreCase: pathPredicate.caseSensitive === false ? ['ignore'] : [] }),
      headerPredicates: headerPredicates.map((item) => ({ ...item, uid: uniqueId() })),
      urlParamPredicates: urlParamPredicates.map((item) => ({ ...item, uid: uniqueId() })),
      authConfig_enabled: authConfigEnabledValue,
      authConfig_allowedConsumerLevels: normalizeUserLevels(value?.authConfig?.allowedConsumerLevels),
      modelPredicates: modelPredicates ? modelPredicates.map((item) => ({ ...item })) : [],
      ...fallbackInitValues,
    };

    initValues.upstreams = upstreams.map((item, index) => {
      const nextItem: Record<string, any> = {
        providerName: item.provider,
        weight: item.weight,
      };
      if (item.provider && !publishedProviderSet.has(item.provider)) {
        legacyProviders.push(item.provider);
      }
      if (item.modelMapping) {
        nextItem.modelMapping = modelMapping2String(item.modelMapping);
      }
      return nextItem;
    });

    if (legacyProviders.length) {
      setLegacyProviderWarning(
        `${t('aiRoute.routeForm.legacyBindingWarning')} ${Array.from(new Set(legacyProviders)).join(', ')}`,
      );
    } else {
      setLegacyProviderWarning('');
    }

    form.setFieldsValue(initValues);
  };

  useImperativeHandle(ref, () => ({
    reset: () => {
      form.resetFields();
    },
    handleSubmit: async () => {
      setUpstreamsError(false);
      const values = await form.validateFields();
      const { upstreams = [] } = values;

      if (!upstreams.length) {
        setUpstreamsError('noUpstream');
        return false;
      }

      const sumWeights = upstreams.reduce((accumulator, currentObject) => {
        return parseInt(accumulator, 10) + parseInt(currentObject.weight, 10);
      }, 0);

      if (sumWeights !== 100) {
        setUpstreamsError('badWeightSum');
        return false;
      }

      const {
        name,
        domains,
        pathPredicate,
        headerPredicates,
        urlParamPredicates,
        fallbackConfig_upstreams = '',
        authConfig_allowedConsumerLevels = [],
        fallbackConfig_modelNames = '',
        fallbackConfig_responseCodes = [],
        modelPredicates = [],
      } = values;
      const normalizedLevels = normalizeUserLevels(authConfig_allowedConsumerLevels);
      const expandedConsumers = expandConsumersByAllowedLevels(normalizedLevels, consumerList);

      const payload: AiRoute = {
        name,
        domains: domains && !Array.isArray(domains) ? [domains] : domains,
        pathPredicate,
        headerPredicates,
        urlParamPredicates,
        fallbackConfig: {
          enabled: fallbackConfigEnabled,
          upstreams: [],
        },
        authConfig: {
          enabled: authConfigEnabled,
        },
        customConfigs: {
          ...(value?.customConfigs || {}),
        },
        customLabels: value?.customLabels,
        upstreams: [],
      };
      delete payload.customConfigs[AI_ROUTE_BINDING_REFS_CONFIG_KEY];

      payload.upstreams = upstreams.map(({ providerName, weight, modelMapping }) => {
        return {
          provider: providerName,
          weight,
          modelMapping: string2ModelMapping(modelMapping),
        };
      });

      payload.modelPredicates = modelPredicates
        ? modelPredicates.map(({ matchType, matchValue }) => ({ matchType, matchValue }))
        : null;

      if (fallbackConfigEnabled) {
        payload.fallbackConfig.upstreams = [{
          provider: fallbackConfig_upstreams,
          modelMapping: string2ModelMapping(fallbackConfig_modelNames),
        }];
        payload.fallbackConfig.strategy = 'SEQ';
        payload.fallbackConfig.responseCodes = fallbackConfig_responseCodes;
      }
      payload.authConfig.allowedConsumerLevels = normalizedLevels;
      payload.authConfig.allowedConsumers = expandedConsumers;
      return payload;
    },
  }));

  const getOptions = (index: number) => {
    try {
      const upstreams = form.getFieldValue('upstreams');
      if (upstreams[index]?.providerName) {
        return getOptionsForProvider(upstreams[index].providerName);
      }
    } catch (error) {
      return [];
    }
    return [];
  };

  return (
    <Form form={form} layout="vertical">
      <Form.Item
        label={t('aiRoute.routeForm.label.name')}
        required
        name="name"
        rules={[
          {
            required: true,
            pattern: /^[a-z0-9](?:[a-z0-9.-]{0,61}[a-z0-9])?$/,
            message: tr('aiRoute.routeForm.rule.nameRequired'),
          },
        ]}
      >
        <Input
          showCount
          allowClear
          maxLength={63}
          disabled={!!value}
          placeholder={tr('aiRoute.routeForm.rule.nameRequired')}
        />
      </Form.Item>

      <div style={{ display: 'flex' }}>
        <Form.Item
          style={{ flex: 1, marginRight: '8px' }}
          label={t('aiRoute.routeForm.label.domain')}
          name="domains"
          extra={<HistoryButton text={t('domain.createDomain')} path="/domain" />}
        >
          <Select allowClear mode="multiple">
            {domainList.map((item) => (
              <Select.Option key={item.name} value={item.name}>
                {item.name}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
        <RedoOutlinedBtn getList={domainsResult} />
      </div>

      <Form.Item label={t('route.routeForm.path')} required>
        <Input.Group compact>
          <Form.Item
            name={['pathPredicate', 'matchType']}
            noStyle
            rules={[
              {
                required: true,
                message: tr('route.routeForm.pathPredicateRequired'),
              },
            ]}
          >
            <Select style={{ width: '20%' }} placeholder={t('route.routeForm.matchType')}>
              <Option value="PRE">{t('route.matchTypes.PRE')}</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name={['pathPredicate', 'matchValue']}
            noStyle
            rules={[
              {
                required: true,
                message: tr('route.routeForm.pathMatcherRequired'),
              },
            ]}
          >
            <Input style={{ width: '60%' }} placeholder={tr('route.routeForm.pathMatcherPlacedholder')} />
          </Form.Item>
          <Form.Item name={['pathPredicate', 'ignoreCase']} noStyle>
            <Checkbox.Group
              options={[
                {
                  label: t('route.routeForm.caseInsensitive'),
                  value: 'ignore',
                },
              ]}
              style={{ width: '18%', display: 'inline-flex', marginLeft: 12, marginTop: 4 }}
            />
          </Form.Item>
        </Input.Group>
      </Form.Item>

      <Form.Item label={t('route.routeForm.header')} name="headerPredicates" tooltip={t('route.routeForm.headerTooltip')}>
        <FactorGroup />
      </Form.Item>

      <Form.Item label={t('route.routeForm.query')} name="urlParamPredicates" tooltip={t('route.routeForm.queryTooltip')}>
        <FactorGroup />
      </Form.Item>

      <Form.Item label={t('aiRoute.routeForm.label.modelPredicates')}>
        <Form.List name="modelPredicates" initialValue={[{}]}>
          {(fields, { add, remove }) => (
            <>
              <div className="ant-table ant-table-small">
                <div className="ant-table-content">
                  <table style={{ tableLayout: 'auto' }}>
                    <thead className="ant-table-thead">
                      <tr>
                        <th className="ant-table-cell">Key</th>
                        <th className="ant-table-cell">{t('aiRoute.routeForm.modelMatchType')}</th>
                        <th className="ant-table-cell">{t('aiRoute.routeForm.modelMatchValue')}</th>
                        <th className="ant-table-cell">{t('misc.action')}</th>
                      </tr>
                    </thead>
                    <tbody className="ant-table-tbody">
                      {fields.length ? fields.map(({ key, name, ...restField }) => (
                        <tr className="ant-table-row ant-table-row-level-0" key={key}>
                          <td className="ant-table-cell">model</td>
                          <td className="ant-table-cell">
                            <Form.Item
                              name={[name, 'matchType']}
                              noStyle
                              rules={[{ required: true, message: tr('aiRoute.routeForm.rule.matchTypeRequired') }]}
                            >
                              <Select style={{ width: '200px' }}>
                                {[
                                  { name: 'EQUAL', label: t('route.matchTypes.EQUAL') },
                                  { name: 'PRE', label: t('route.matchTypes.PRE') },
                                ].map((item) => (
                                  <Select.Option key={item.name} value={item.name}>
                                    {item.label}
                                  </Select.Option>
                                ))}
                              </Select>
                            </Form.Item>
                          </td>
                          <td className="ant-table-cell">
                            <Form.Item
                              {...restField}
                              name={[name, 'matchValue']}
                              noStyle
                              rules={[{ required: true, message: t('aiRoute.routeForm.rule.matchValueRequired') || '' }]}
                            >
                              <Input style={{ width: '200px' }} />
                            </Form.Item>
                          </td>
                          <td className="ant-table-cell">
                            <MinusCircleOutlined onClick={() => remove(name)} />
                          </td>
                        </tr>
                      )) : (
                        <tr className="ant-table-row ant-table-row-level-0">
                          <td className="ant-table-cell" colSpan={4}>
                            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ margin: 0 }} />
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
              <div>
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add()}>
                  {t('aiRoute.routeForm.addModelPredicate')}
                </Button>
              </div>
            </>
          )}
        </Form.List>
      </Form.Item>

      {legacyProviderWarning ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message={legacyProviderWarning}
          description={t('aiRoute.routeForm.legacyBindingDescription')}
        />
      ) : null}

      <Form.Item
        label={t('aiRoute.routeForm.label.services')}
        extra={(
          <>
            {upstreamsError ? (
              <div className="ant-form-item-explain-error">{t(`aiRoute.routeForm.rule.${upstreamsError}`)}</div>
            ) : null}
            <HistoryButton text={t('menu.modelAssetManagement')} path="/ai/model-assets" />
          </>
        )}
      >
        <Form.List name="upstreams" initialValue={[null]}>
          {(fields, { add, remove }) => {
            const baseStyle = { width: 280 };
            const weightStyle = { width: 140 };
            const requiredStyle = { display: 'inline-block', marginRight: '4px', color: '#ff4d4f' };

            return (
              <>
                <Space style={{ display: 'flex', color: '#808080' }} align="start">
                  <div style={baseStyle}>
                    <span style={requiredStyle}>*</span>
                    {t('aiRoute.routeForm.label.serviceName')}
                  </div>
                  <div style={weightStyle}>
                    <span style={requiredStyle}>*</span>
                    {t('aiRoute.routeForm.label.serviceWeight')}
                  </div>
                  <div style={baseStyle}>{t('aiRoute.routeForm.label.targetModel')}</div>
                </Space>

                {fields.map(({ key, name, ...restField }) => {
                  const currentProviderValue = form.getFieldValue(['upstreams', name, 'providerName']);
                  const selectedProviderValues = ((form.getFieldValue('upstreams') || []) as Array<{ providerName?: string }>)
                    .map((item) => item?.providerName)
                    .filter((item) => !!item);

                  return (
                    <Space key={key} style={{ display: 'flex' }} align="start">
                      <Form.Item
                        {...restField}
                        name={[name, 'providerName']}
                        style={{ marginBottom: '0.5rem' }}
                        rules={[{ required: true, message: tr('aiRoute.routeForm.rule.targetServiceRequired') }]}
                      >
                        <Select showSearch style={baseStyle} optionFilterProp="children">
                          {renderLegacyProviderOption(currentProviderValue)}
                          {publishedProviderOptions.map((item) => (
                            <Select.Option
                              key={item.value}
                              value={item.value}
                              disabled={selectedProviderValues.includes(item.value) && item.value !== currentProviderValue}
                            >
                              {item.label}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>

                      <Form.Item
                        {...restField}
                        name={[name, 'weight']}
                        style={{ ...weightStyle, marginBottom: 0 }}
                        rules={[{ required: true, message: tr('aiRoute.routeForm.rule.serviceWeightRequired') }]}
                      >
                        <InputNumber style={weightStyle} min={0} max={100} addonAfter="%" />
                      </Form.Item>

                      <Form.Item {...restField} name={[name, 'modelMapping']} noStyle>
                        <ModelMappingEditor style={baseStyle} options={getOptions(name)} />
                      </Form.Item>

                      {fields.length > 1 ? (
                        <Form.Item noStyle>
                          <MinusCircleOutlined onClick={() => remove(name)} />
                        </Form.Item>
                      ) : null}
                    </Space>
                  );
                })}

                <div>
                  <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add()}>
                    {t('aiRoute.routeForm.addTargetService')}
                  </Button>
                </div>
              </>
            );
          }}
        </Form.List>
      </Form.Item>

      <Form.Item
        name="fallbackConfig_enabled"
        label={t('aiRoute.routeForm.label.fallbackConfig')}
        valuePropName="checked"
        initialValue={false}
        extra={t('aiRoute.routeForm.label.fallbackConfigExtra')}
      >
        <Switch
          onChange={(checked) => {
            setFallbackConfigEnabled(checked);
            form.resetFields(['fallbackConfig_upstreams']);
          }}
        />
      </Form.Item>

      {fallbackConfigEnabled ? (
        <div style={{ display: 'flex' }}>
          <Form.Item
            style={{ flex: 1, marginRight: '8px' }}
            required
            name="fallbackConfig_responseCodes"
            label={t('aiRoute.routeForm.label.fallbackResponseCodes')}
            rules={[{ required: true, message: tr('aiRoute.routeForm.rule.fallbackResponseCodesRequired') }]}
          >
            <Select mode="multiple">
              {defaultFallbackResponseCodes.map((item) => (
                <Select.Option key={item} value={item}>
                  {item}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            style={{ flex: 1, marginRight: '8px' }}
            required
            name="fallbackConfig_upstreams"
            label={t('aiRoute.routeForm.label.fallbackUpstream')}
            rules={[{ required: true, message: tr('aiRoute.routeForm.rule.fallbackUpstreamRequired') }]}
          >
            <Select showSearch optionFilterProp="children">
              {renderLegacyProviderOption(form.getFieldValue('fallbackConfig_upstreams'))}
              {publishedProviderOptions.map((item) => (
                <Select.Option key={item.value} value={item.value}>
                  {item.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            required
            style={{ flex: 1 }}
            name="fallbackConfig_modelNames"
            label={t('aiRoute.routeForm.label.targetModel')}
            rules={[{ required: true, message: tr('aiRoute.routeForm.rule.modelNameRequired') }]}
          >
            <ModelMappingEditor options={getOptionsForProvider(form.getFieldValue('fallbackConfig_upstreams'))} />
          </Form.Item>
        </div>
      ) : null}

      <Form.Item
        name="authConfig_enabled"
        label={t('aiRoute.routeForm.label.authConfig')}
        valuePropName="checked"
        initialValue={false}
        extra={t('aiRoute.routeForm.label.authConfigExtra')}
      >
        <Switch onChange={(checked) => setAuthConfigEnabled(checked)} />
      </Form.Item>

      <Form.Item
        label={t('misc.authType')}
        name="authType"
        initialValue={CredentialType.KEY_AUTH.key}
        extra={t('misc.keyAuthOnlyTip')}
      >
        <Select disabled>
          {Object.values(CredentialType)
            .filter((ct) => !!ct.enabled)
            .map((ct) => (
              <Select.Option key={ct.key} value={ct.key}>
                {ct.displayName}
              </Select.Option>
            ))}
        </Select>
      </Form.Item>

      <Form.Item
        label={t('aiRoute.routeForm.label.authConfigLevelList')}
        extra={<HistoryButton text={t('consumer.create')} path="/consumer" />}
      >
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Form.Item name="authConfig_allowedConsumerLevels" noStyle>
            <Select
              allowClear
              mode="multiple"
              placeholder={t('aiRoute.routeForm.label.authConfigLevelList')}
              style={{ flex: 1 }}
            >
              {USER_LEVELS.map((level) => (
                <Select.Option key={level} value={level}>
                  {t(`consumer.userLevel.${level}`)}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Button style={{ marginLeft: 8 }} onClick={() => consumerResult.run()} icon={<RedoOutlined />} />
        </div>
      </Form.Item>
    </Form>
  );
});

export default AiRouteForm;
