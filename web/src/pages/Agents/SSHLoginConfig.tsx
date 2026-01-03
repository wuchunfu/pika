import React, {useEffect, useState} from 'react';
import {Alert, App, Button, Card, Form, Switch} from 'antd';
import {Save, Terminal} from 'lucide-react';
import type {SSHLoginConfig as SSHLoginConfigType} from '@/types';
import {getSSHLoginConfig, updateSSHLoginConfig} from '@/api/agent';

interface SSHLoginConfigProps {
    agentId: string;
}

const SSHLoginConfig: React.FC<SSHLoginConfigProps> = ({agentId}) => {
    const {message} = App.useApp();
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [config, setConfig] = useState<SSHLoginConfigType | null>(null);

    // 加载配置
    const loadConfig = async () => {
        try {
            setLoading(true);
            const cfg = await getSSHLoginConfig(agentId);
            setConfig(cfg);
            // 设置表单值
            form.setFieldsValue({
                enabled: cfg.enabled || false,
                recordFailed: cfg.recordFailed !== undefined ? cfg.recordFailed : true,
            });
        } catch (error) {
            console.error('Failed to load SSH login config:', error);
            // 发生错误时使用默认配置
            form.setFieldsValue({
                enabled: false,
                recordFailed: true,
            });
        } finally {
            setLoading(false);
        }
    };

    // 保存配置
    const handleSaveConfig = async () => {
        try {
            setSaving(true);
            const values = form.getFieldsValue();
            await updateSSHLoginConfig(agentId, {
                enabled: values.enabled,
                recordFailed: values.recordFailed,
            });
            // 显示来自后端的消息
            message.success('配置已保存');
            await loadConfig();
        } catch (error: any) {
            console.error('Failed to save SSH login config:', error);
            message.error(error.response?.data?.error || '配置保存失败');
        } finally {
            setSaving(false);
        }
    };

    useEffect(() => {
        loadConfig();
    }, [agentId]);

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
                    onClick={handleSaveConfig}
                    loading={saving}
                >
                    保存配置
                </Button>
            }
            loading={loading}
        >
            <Form
                form={form}
                layout="vertical"
                initialValues={{
                    enabled: false,
                    recordFailed: true,
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
                    label="记录失败登录"
                    name="recordFailed"
                    valuePropName="checked"
                    extra={'是否记录失败的 SSH 登录尝试（有助于发现暴力破解攻击）'}
                >
                    <Switch
                        checkedChildren="记录"
                        unCheckedChildren="不记录"
                    />
                </Form.Item>
            </Form>
        </Card>
    );
};

export default SSHLoginConfig;
