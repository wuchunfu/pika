import {useEffect} from 'react';
import {App, Button, Card, Form, Input, InputNumber, Radio, Select, Space, Spin, Switch} from 'antd';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import type {PublicIPConfig} from '@/api/property';
import {getPublicIPConfig, savePublicIPConfig} from '@/api/property';
import {listAgentsByAdmin} from '@/api/agent.ts';
import {getErrorMessage} from '@/lib/utils';
import type {Agent} from '@/types';

interface PublicIPConfigProps {
    defaultIPv4APIs: string[];
    defaultIPv6APIs: string[];
}

const formatApiList = (apis: string[] | undefined, defaults: string[]) => {
    const list = apis && apis.length > 0 ? apis : defaults;
    return list.join('\n');
};

const parseApiList = (text: string, defaults: string[]) => {
    const items = text
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean);
    return items.length > 0 ? items : defaults;
};

const formatAgentLabel = (agent: Agent) => {
    if (agent.name && agent.hostname) {
        return `${agent.name} (${agent.hostname})`;
    }
    return agent.name || agent.hostname || agent.id;
};

const PublicIPConfigComponent = ({defaultIPv4APIs, defaultIPv6APIs}: PublicIPConfigProps) => {
    const [form] = Form.useForm();
    const {message: messageApi} = App.useApp();
    const queryClient = useQueryClient();

    const {data: config, isLoading} = useQuery({
        queryKey: ['publicIPConfig'],
        queryFn: getPublicIPConfig,
    });

    const {data: agentsResponse} = useQuery({
        queryKey: ['admin', 'agents', 'public-ip'],
        queryFn: () => listAgentsByAdmin(),
    });

    const agentOptions = (agentsResponse?.data || []).map((agent) => ({
        label: formatAgentLabel(agent),
        value: agent.id,
    }));

    const saveMutation = useMutation({
        mutationFn: savePublicIPConfig,
        onSuccess: () => {
            messageApi.success('保存成功');
            queryClient.invalidateQueries({queryKey: ['publicIPConfig']});
        },
        onError: (error: unknown) => {
            messageApi.error(getErrorMessage(error, '保存失败'));
        },
    });

    useEffect(() => {
        if (config) {
            form.setFieldsValue({
                enabled: config.enabled ?? false,
                intervalSeconds: config.intervalSeconds ?? 300,
                ipv4Scope: config.ipv4Scope ?? 'all',
                ipv4AgentIds: config.ipv4AgentIds ?? [],
                ipv6Scope: config.ipv6Scope ?? 'all',
                ipv6AgentIds: config.ipv6AgentIds ?? [],
                ipv4Enabled: config.ipv4Enabled ?? true,
                ipv6Enabled: config.ipv6Enabled ?? true,
                ipv4ApisText: formatApiList(config.ipv4Apis, defaultIPv4APIs),
                ipv6ApisText: formatApiList(config.ipv6Apis, defaultIPv6APIs),
            });
        }
    }, [config, form, defaultIPv4APIs, defaultIPv6APIs]);

    const handleSave = async () => {
        try {
            const values = await form.validateFields();
            const payload: PublicIPConfig = {
                enabled: values.enabled ?? false,
                intervalSeconds: values.intervalSeconds ?? 300,
                ipv4Scope: values.ipv4Scope ?? 'all',
                ipv4AgentIds: values.ipv4Scope === 'custom' ? values.ipv4AgentIds || [] : [],
                ipv6Scope: values.ipv6Scope ?? 'all',
                ipv6AgentIds: values.ipv6Scope === 'custom' ? values.ipv6AgentIds || [] : [],
                ipv4Enabled: values.ipv4Enabled ?? true,
                ipv6Enabled: values.ipv6Enabled ?? true,
                ipv4Apis: parseApiList(values.ipv4ApisText || '', defaultIPv4APIs),
                ipv6Apis: parseApiList(values.ipv6ApisText || '', defaultIPv6APIs),
            };
            saveMutation.mutate(payload);
        } catch (error) {
            // 表单验证失败
        }
    };

    const handleReset = () => {
        if (config) {
            form.setFieldsValue({
                enabled: config.enabled ?? false,
                intervalSeconds: config.intervalSeconds ?? 300,
                ipv4Scope: config.ipv4Scope ?? 'all',
                ipv4AgentIds: config.ipv4AgentIds ?? [],
                ipv6Scope: config.ipv6Scope ?? 'all',
                ipv6AgentIds: config.ipv6AgentIds ?? [],
                ipv4Enabled: config.ipv4Enabled ?? true,
                ipv6Enabled: config.ipv6Enabled ?? true,
                ipv4ApisText: formatApiList(config.ipv4Apis, defaultIPv4APIs),
                ipv6ApisText: formatApiList(config.ipv6Apis, defaultIPv6APIs),
            });
        }
    };

    const handleUseDefaults = () => {
        form.setFieldsValue({
            ipv4ApisText: defaultIPv4APIs.join('\n'),
            ipv6ApisText: defaultIPv6APIs.join('\n'),
        });
    };

    if (isLoading) {
        return (
            <div className="flex justify-center items-center py-20">
                <Spin/>
            </div>
        );
    }

    return (
        <div>
            <div className="mb-4">
                <h2 className="text-xl font-bold">公网 IP 采集</h2>
                <p className="text-gray-500 mt-2">通过 HTTP 接口采集探针公网 IPv4/IPv6 地址</p>
            </div>

            <Form form={form} layout="vertical" onFinish={handleSave}>
                <Space direction={'vertical'} className={'w-full'}>
                    <Card title="采集设置" type="inner" className="mb-4">
                        <div className="flex flex-wrap items-center gap-6">
                            <Form.Item label="启用采集" name="enabled" valuePropName="checked">
                                <Switch/>
                            </Form.Item>
                            <Form.Item
                                label="采集间隔(秒)"
                                name="intervalSeconds"
                                rules={[{type: 'number', min: 30, message: '采集间隔不能小于 30 秒'}]}
                            >
                                <InputNumber min={30} max={86400}/>
                            </Form.Item>
                        </div>
                    </Card>

                    <Card title="IPv4 配置" type="inner" className="mb-4">
                        <Form.Item label="启用 IPv4" name="ipv4Enabled" valuePropName="checked">
                            <Switch/>
                        </Form.Item>
                        <Form.Item label="IPv4 采集范围" name="ipv4Scope">
                            <Radio.Group>
                                <Radio.Button value="all">全部探针</Radio.Button>
                                <Radio.Button value="custom">自定义探针</Radio.Button>
                            </Radio.Group>
                        </Form.Item>
                        <Form.Item noStyle shouldUpdate>
                            {({getFieldValue}) => {
                                const enabled = getFieldValue('ipv4Enabled');
                                const scope = getFieldValue('ipv4Scope');
                                if (!enabled || scope !== 'custom') {
                                    return null;
                                }
                                return (
                                    <Form.Item
                                        label="选择 IPv4 探针"
                                        name="ipv4AgentIds"
                                        rules={[{required: true, message: '请选择至少一个探针'}]}
                                    >
                                        <Select
                                            mode="multiple"
                                            placeholder="选择需要采集 IPv4 的探针"
                                            options={agentOptions}
                                            optionFilterProp="label"
                                            showSearch
                                        />
                                    </Form.Item>
                                );
                            }}
                        </Form.Item>
                        <Form.Item
                            label="IPv4 API 列表"
                            name="ipv4ApisText"
                            tooltip="每行一个 HTTP/HTTPS API 地址"
                        >
                            <Input.TextArea rows={6} placeholder="每行一个 IPv4 API"/>
                        </Form.Item>
                    </Card>

                    <Card title="IPv6 配置" type="inner" className="mb-4">
                        <Form.Item label="启用 IPv6" name="ipv6Enabled" valuePropName="checked">
                            <Switch/>
                        </Form.Item>
                        <Form.Item label="IPv6 采集范围" name="ipv6Scope">
                            <Radio.Group>
                                <Radio.Button value="all">全部探针</Radio.Button>
                                <Radio.Button value="custom">自定义探针</Radio.Button>
                            </Radio.Group>
                        </Form.Item>
                        <Form.Item noStyle shouldUpdate>
                            {({getFieldValue}) => {
                                const enabled = getFieldValue('ipv6Enabled');
                                const scope = getFieldValue('ipv6Scope');
                                if (!enabled || scope !== 'custom') {
                                    return null;
                                }
                                return (
                                    <Form.Item
                                        label="选择 IPv6 探针"
                                        name="ipv6AgentIds"
                                        rules={[{required: true, message: '请选择至少一个探针'}]}
                                    >
                                        <Select
                                            mode="multiple"
                                            placeholder="选择需要采集 IPv6 的探针"
                                            options={agentOptions}
                                            optionFilterProp="label"
                                            showSearch
                                        />
                                    </Form.Item>
                                );
                            }}
                        </Form.Item>
                        <Form.Item
                            label="IPv6 API 列表"
                            name="ipv6ApisText"
                            tooltip="每行一个 HTTP/HTTPS API 地址"
                        >
                            <Input.TextArea rows={6} placeholder="每行一个 IPv6 API"/>
                        </Form.Item>
                    </Card>

                    <Form.Item>
                        <Space>
                            <Button type="primary" htmlType="submit" loading={saveMutation.isPending}>
                                保存配置
                            </Button>
                            <Button onClick={handleReset}>
                                恢复当前配置
                            </Button>
                            <Button onClick={handleUseDefaults}>
                                使用默认 API
                            </Button>
                        </Space>
                    </Form.Item>
                </Space>
            </Form>
        </div>
    );
};

export default PublicIPConfigComponent;
