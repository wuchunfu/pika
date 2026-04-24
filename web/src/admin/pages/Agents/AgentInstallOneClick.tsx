import React, { useEffect, useMemo, useState } from 'react';
import { Alert, App, Button, Card, Input, Select, Space, Spin, Typography } from 'antd';
import { CopyIcon } from 'lucide-react';
import copy from 'copy-to-clipboard';
import { Link } from 'react-router-dom';
import {
    AgentInstallLayout,
    ConfigHelper,
    ServiceHelper,
    AGENT_NAME,
} from './AgentInstallShared';
import { useAgentInstallConfig } from './useAgentInstallConfig';
import { getAgentInstallConfig, saveAgentInstallConfig } from '@/api/property';

const { Paragraph } = Typography;

const AgentInstallOneClick = () => {
    const { message } = App.useApp();
    const {
        apiKeys,
        selectedApiKeyId,
        setSelectedApiKeyId,
        selectedApiKey,
        customAgentName,
        setCustomAgentName,
        loading,
        backendServerUrl,
        apiKeyOptions,
    } = useAgentInstallConfig();

    const [serverUrl, setServerUrl] = useState<string>('');
    const [serverUrlError, setServerUrlError] = useState<string>('');
    const [serverUrlLoading, setServerUrlLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const effectiveServerUrl = serverUrl.trim() || backendServerUrl;

    // 加载服务端地址配置
    useEffect(() => {
        const fetchConfig = async () => {
            try {
                const config = await getAgentInstallConfig();
                const configuredUrl = (config.serverUrl || '').trim();
                setServerUrl(configuredUrl);
                setServerUrlError(configuredUrl ? '' : '请先配置服务端地址');
            } catch (error) {
                console.error('加载服务端地址配置失败:', error);
            } finally {
                setServerUrlLoading(false);
            }
        };

        void fetchConfig();
    }, []);

    // 保存服务端地址配置（防抖）
    const saveServerUrl = async (value: string) => {
        const trimmed = value.trim();
        if (!trimmed) {
            setServerUrlError('请先配置服务端地址');
            return;
        }
        setServerUrlError('');
        setSaving(true);
        try {
            await saveAgentInstallConfig({ serverUrl: trimmed });
            message.success('服务端地址已保存');
        } catch (error) {
            console.error('保存服务端地址失败:', error);
            message.error('保存服务端地址失败');
        } finally {
            setSaving(false);
        }
    };

    const handleFillCurrentUrl = () => {
        const currentUrl = window.location.origin;
        setServerUrl(currentUrl);
        setServerUrlError('');
        void saveServerUrl(currentUrl);
    };

    const installCommand = useMemo(() => {
        if (!effectiveServerUrl || !selectedApiKey) {
            return '';
        }
        const trimmedName = customAgentName.trim();
        const nameParam = trimmedName ? `&name=${encodeURIComponent(trimmedName)}` : '';
        return `curl -fsSL "${effectiveServerUrl}/api/agent/install.sh?token=${selectedApiKey}${nameParam}" | sudo bash`;
    }, [effectiveServerUrl, selectedApiKey, customAgentName]);

    const copyToClipboard = (text: string) => {
        copy(text);
        message.success('已复制到剪贴板');
    };

    return (
        <AgentInstallLayout activeKey="one-click">
            <Space direction="vertical" className="w-full">
                <Card type="inner" title="配置选项">
                    <Space direction="vertical" className="w-full">
                        <div>
                            <div className="mb-1 text-gray-600 dark:text-slate-400">
                                服务端地址 <span className="text-xs text-gray-400">(必填)</span>
                            </div>
                            <Spin spinning={serverUrlLoading}>
                                <Input
                                    placeholder="例如: https://monitor.example.com"
                                    value={serverUrl}
                                    onChange={(e) => setServerUrl(e.target.value)}
                                    onBlur={(e) => void saveServerUrl(e.target.value)}
                                    className="w-full"
                                />
                            </Spin>
                            <div className="mt-2 flex gap-2">
                                <Button size="small" onClick={handleFillCurrentUrl}>
                                    使用当前访问地址
                                </Button>
                                <Button size="small" loading={saving} onClick={() => void saveServerUrl(serverUrl)}>
                                    保存
                                </Button>
                            </div>
                            {serverUrlError ? (
                                <div className="mt-1 text-xs text-red-500">{serverUrlError}</div>
                            ) : (
                                <div className="mt-1 text-xs text-gray-400">
                                    请配置可访问的服务端地址，否则无法生成安装命令
                                </div>
                            )}
                        </div>

                        <div>
                            <div className="mb-1 text-gray-600 dark:text-slate-400">选择通信密钥</div>
                            {apiKeys.length === 0 ? (
                                <Alert
                                    message="暂无可用的通信密钥"
                                    description={
                                        <span>
                                            请先前往 <Link to="/admin/api-keys">通信密钥管理</Link> 页面生成一个通信密钥
                                        </span>
                                    }
                                    type="warning"
                                    showIcon
                                    className="mt-2"
                                />
                            ) : (
                                <Select
                                    className="w-full"
                                    value={selectedApiKeyId}
                                    onChange={setSelectedApiKeyId}
                                    options={apiKeyOptions}
                                    loading={loading}
                                    placeholder="请选择通信密钥"
                                />
                            )}
                        </div>

                        <div>
                            <div className="mb-1 text-gray-600 dark:text-slate-400">
                                自定义名称 <span className="text-xs text-gray-400">(可选，留空则使用主机名)</span>
                            </div>
                            <Input
                                placeholder="请输入自定义名称，例如: my-server-01"
                                value={customAgentName}
                                onChange={(e) => setCustomAgentName(e.target.value)}
                                className="w-full"
                                allowClear
                            />
                        </div>
                    </Space>
                </Card>

                <Alert
                    description="一键安装脚本仅支持 Linux/macOS 系统。"
                    type="info"
                    showIcon
                    className="mt-2"
                />
                {!effectiveServerUrl && (
                    <Alert
                        description="请先配置服务端地址后再生成安装命令。"
                        type="warning"
                        showIcon
                        className="mt-2"
                    />
                )}
                <Card type="inner" title="一键安装">
                    <Paragraph type="secondary" className="mb-3 text-gray-600 dark:text-slate-400">
                        脚本会自动检测系统架构并下载对应版本的探针，然后完成注册和安装。
                    </Paragraph>
                    <pre
                        className="m-0 overflow-auto text-sm bg-gray-50 dark:bg-slate-800 p-3 rounded text-gray-900 dark:text-slate-100">
                        <code>{installCommand}</code>
                    </pre>
                    <Button
                        type="link"
                        onClick={() => void copyToClipboard(installCommand)}
                        icon={<CopyIcon className="h-4 w-4" />}
                        style={{ margin: 0, padding: 0 }}
                        disabled={!selectedApiKey || !effectiveServerUrl}
                    >
                        复制命令
                    </Button>
                </Card>

                <ServiceHelper os={AGENT_NAME} />
                <ConfigHelper />
            </Space>
        </AgentInstallLayout>
    );
};

export default AgentInstallOneClick;