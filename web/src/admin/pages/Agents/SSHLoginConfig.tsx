import React, {useEffect} from 'react';
import {Alert, App, Button, Card, Form, Input, Space, Switch} from 'antd';
import {Save, Terminal} from 'lucide-react';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import type {SSHLoginConfig as SSHLoginConfigType} from '@/types';
import {getSSHLoginConfig, updateSSHLoginConfig} from '@/api/agent';
import {getErrorMessage} from '@/lib/utils';

const {TextArea} = Input;

// 验证 IP 地址格式（支持 IPv4 和 CIDR）
const validateIPOrCIDR = (value: string): boolean => {
    // IPv4 地址正则
    const ipv4Regex = /^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
    // CIDR 格式正则 (IPv4/prefix)
    const cidrRegex = /^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\/(3[0-2]|[12]?[0-9])$/;

    return ipv4Regex.test(value) || cidrRegex.test(value);
};

// 解析文本为 IP 数组
const parseIPWhitelist = (text: string): string[] => {
    return text
        .split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);
};

// 格式化 IP 数组为文本
const formatIPWhitelist = (ips: string[]): string => {
    return ips.join('\n');
};

interface SSHLoginConfigProps {
    agentId: string;
}

const SSHLoginConfig: React.FC<SSHLoginConfigProps> = ({agentId}) => {
    const {message} = App.useApp();
    const [form] = Form.useForm();
    const queryClient = useQueryClient();

    // 获取 SSH 登录监控配置
    const {data: config, isLoading} = useQuery({
        queryKey: ['sshLoginConfig', agentId],
        queryFn: () => getSSHLoginConfig(agentId),
    });

    // 保存配置 mutation
    const saveMutation = useMutation({
        mutationFn: async () => {
            const values = form.getFieldsValue();
            const ipWhitelistText = values.ipWhitelistText || '';
            const ipWhitelist = parseIPWhitelist(ipWhitelistText);

            // 验证所有 IP 地址格式
            const invalidIPs: string[] = [];
            ipWhitelist.forEach(ip => {
                if (!validateIPOrCIDR(ip)) {
                    invalidIPs.push(ip);
                }
            });

            if (invalidIPs.length > 0) {
                throw new Error(`以下 IP 地址或 CIDR 格式不正确：\n${invalidIPs.join('\n')}`);
            }

            return updateSSHLoginConfig(agentId, {
                enabled: values.enabled,
                ipWhitelist: ipWhitelist,
            });
        },
        onSuccess: () => {
            message.success('配置已保存');
            queryClient.invalidateQueries({queryKey: ['sshLoginConfig', agentId]});
        },
        onError: (error: unknown) => {
            console.error('Failed to save SSH login config:', error);
            message.error(getErrorMessage(error, '配置保存失败'));
        },
    });

    // 初始化表单值
    useEffect(() => {
        if (config) {
            form.setFieldsValue({
                enabled: config.enabled || false,
                ipWhitelistText: formatIPWhitelist(config.ipWhitelist || []),
            });
        } else {
            form.setFieldsValue({
                enabled: false,
                ipWhitelistText: '',
            });
        }
    }, [config, form]);

    return (
        <Card
            title={
                <div className="flex items-center gap-2">
                    <Terminal size={18}/>
                    <span>SSH 登录监控配置</span>
                </div>
            }
            extra={
                <Button
                    type="primary"
                    icon={<Save size={16}/>}
                    onClick={() => saveMutation.mutate()}
                    loading={saveMutation.isPending}
                >
                    保存配置
                </Button>
            }
            loading={isLoading}
        >
            <Space direction="vertical" style={{width: '100%'}} size="large">
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{
                        enabled: false,
                        ipWhitelistText: '',
                    }}
                >
                    <Form.Item
                        label="启用监控"
                        name="enabled"
                        valuePropName="checked"
                        extra={'启用后，探针将自动安装 PAM Hook 并开始监控 SSH 登录事件'}
                    >
                        <Switch
                            checkedChildren="已启用"
                            unCheckedChildren="已禁用"
                        />
                    </Form.Item>

                    <Form.Item
                        label="IP 白名单"
                        name="ipWhitelistText"
                        extra={'白名单中的 IP 地址登录时只记录不发送通知。每行一个 IP 地址或 CIDR 网段，例如：192.168.1.1 或 192.168.1.0/24'}
                    >
                        <TextArea
                            rows={6}
                            placeholder={'每行一个 IP 地址或 CIDR 网段，例如：\n192.168.1.1\n192.168.1.0/24\n10.0.0.0/8'}
                        />
                    </Form.Item>
                </Form>

                {config?.applyStatus && (
                    <Alert
                        message={
                            config.applyStatus === 'success' ? '配置应用成功' :
                                config.applyStatus === 'failed' ? '配置应用失败' :
                                    '配置应用中...'
                        }
                        description={config.applyMessage}
                        type={
                            config.applyStatus === 'success' ? 'success' :
                                config.applyStatus === 'failed' ? 'error' :
                                    'info'
                        }
                        showIcon
                    />
                )}
            </Space>
        </Card>
    );
};

export default SSHLoginConfig;
