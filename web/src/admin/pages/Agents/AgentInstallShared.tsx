import React, { type ReactNode } from 'react';
import { Alert, Button, Card, Input, Select, Space, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import type { ApiKey } from '@/types';

const { Paragraph, Text } = Typography;

export const AGENT_NAME = 'pika-agent';
export const AGENT_NAME_EXE = 'pika-agent.exe';
export const CONFIG_PATH = '~/.pika/agent.yaml';

type InstallNavKey = 'one-click' | 'manual';

export type ApiKeyOption = {
    label: string;
    value: string;
};

export type ApiChooserProps = {
    apiKeys: ApiKey[];
    selectedApiKey: string;
    apiKeyOptions: ApiKeyOption[];
    loading: boolean;
    customAgentName: string;
    onSelectApiKey: (value: string) => void;
    onCustomNameBlur: (value: string) => void;
};

export const AgentInstallLayout = ({ activeKey, children }: { activeKey: InstallNavKey; children: ReactNode }) => {
    const navigate = useNavigate();

    return (
        <Space direction="vertical" className="w-full" size="large">
            <div className="flex gap-2 items-center">
                <div
                    className="text-sm cursor-pointer hover:underline text-gray-600 dark:text-slate-300"
                    onClick={() => navigate(-1)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => e.key === 'Enter' && navigate(-1)}
                >
                    返回 |
                </div>
                <h1 className="text-2xl font-semibold text-gray-900 dark:text-slate-100">探针部署指南</h1>
            </div>

            <Space size={8}>
                <Button
                    type={activeKey === 'one-click' ? 'primary' : 'default'}
                    onClick={() => navigate('/admin/agents-install/one-click')}
                >
                    一键安装
                </Button>
                <Button
                    type={activeKey === 'manual' ? 'primary' : 'default'}
                    onClick={() => navigate('/admin/agents-install/manual')}
                >
                    手动安装
                </Button>
            </Space>

            {children}
        </Space>
    );
};

export const ServerUrlChecker = ({ backendServerUrl, frontendUrl }: { backendServerUrl: string; frontendUrl: string }) => {
    const hasAddressMismatch = backendServerUrl && backendServerUrl !== frontendUrl;

    if (!hasAddressMismatch) {
        return null;
    }

    return (
        <Alert
            message="检测到地址不一致"
            description={
                <Space direction="vertical" className="w-full">
                    <div>
                        当前访问地址: <Text code>{frontendUrl}</Text>
                        <br />
                        后端检测地址: <Text code>{backendServerUrl}</Text>
                    </div>
                    <div>
                        <Text strong>这通常是因为您使用了反向代理，但未正确配置转发头部。</Text>
                    </div>
                    <div>
                        <Text>请在反向代理配置中添加以下头部：</Text>
                    </div>
                    <div>
                        <Text strong>Nginx 配置示例：</Text>
                        <pre className="m-0 mt-2 overflow-auto text-xs bg-gray-100 dark:bg-slate-900 p-2 rounded">
                            {`location / {
    proxy_pass http://backend;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}`}
                        </pre>
                    </div>
                    <div>
                        <Text strong>Caddy 配置示例：</Text>
                        <pre className="m-0 mt-2 overflow-auto text-xs bg-gray-100 dark:bg-slate-900 p-2 rounded">
                            {`reverse_proxy backend:8080 {
    header_up X-Forwarded-Proto {scheme}
    header_up X-Forwarded-Host {host}
}`}
                        </pre>
                    </div>
                    <div>
                        <Text strong>Traefik 配置说明：</Text>
                        <pre className="m-0 mt-2 overflow-auto text-xs bg-gray-100 dark:bg-slate-900 p-2 rounded">
                            {`# Traefik 默认会自动添加 X-Forwarded-* 头部
# 无需额外配置`}
                        </pre>
                    </div>
                    <div className="mt-2">
                        <Text type="secondary">配置完成后，刷新页面即可生效。</Text>
                    </div>
                </Space>
            }
            type="warning"
            showIcon
            closable
        />
    );
};

export const ApiChooser = ({
    apiKeys,
    selectedApiKey,
    apiKeyOptions,
    loading,
    customAgentName,
    onSelectApiKey,
    onCustomNameBlur,
}: ApiChooserProps) => (
    <Card type="inner" title="配置选项">
        <Space direction="vertical" className="w-full">
            <div>
                <div className="mb-1 text-gray-600 dark:text-slate-400">选择 API Token</div>
                {apiKeys.length === 0 ? (
                    <Alert
                        message="暂无可用的 API Token"
                        description={
                            <span>
                                请先前往 <a href="/admin/api-keys">API密钥管理</a> 页面生成一个 API Token
                            </span>
                        }
                        type="warning"
                        showIcon
                        className="mt-2"
                    />
                ) : (
                    <Select
                        className="w-full"
                        value={selectedApiKey}
                        onChange={onSelectApiKey}
                        options={apiKeyOptions}
                        loading={loading}
                        placeholder="请选择 API Token"
                    />
                )}
            </div>

            <div>
                <div className="mb-1 text-gray-600 dark:text-slate-400">
                    自定义名称 <span className="text-xs text-gray-400">(可选，留空则使用主机名)</span>
                </div>
                <Input
                    key={customAgentName}
                    placeholder="请输入自定义名称，例如: my-server-01"
                    defaultValue={customAgentName}
                    onBlur={(e) => {
                        const trimmed = e.currentTarget.value.trim();
                        onCustomNameBlur(trimmed);
                        e.currentTarget.value = trimmed;
                    }}
                    className="w-full"
                    allowClear
                />
            </div>
        </Space>
    </Card>
);

const getCommonCommands = (os: string) => {
    const agentCmd = os.startsWith('windows') ? `.\\${AGENT_NAME_EXE}` : AGENT_NAME;
    const sudo = os.startsWith('windows') ? '' : 'sudo ';

    return `# 查看服务状态
${sudo}${agentCmd} status

# 停止服务
${sudo}${agentCmd} stop

# 启动服务
${sudo}${agentCmd} start

# 重启服务
${sudo}${agentCmd} restart

# 卸载服务
${sudo}${agentCmd} uninstall

# 查看版本
${agentCmd} version`;
};

export const ServiceHelper = ({ os }: { os: string }) => (
    <Card type="inner" title="服务管理命令">
        <Paragraph type="secondary" className="mb-3 text-gray-600 dark:text-slate-400">
            注册完成后，您可以使用以下命令管理探针服务：
        </Paragraph>
        <pre
            className="m-0 overflow-auto text-sm bg-gray-50 dark:bg-slate-800 p-3 rounded text-gray-900 dark:text-slate-100">
            <code>{getCommonCommands(os)}</code>
        </pre>
    </Card>
);

export const ConfigHelper = () => (
    <Card type="inner" title="配置文件说明">
        <Paragraph className="text-gray-900 dark:text-slate-100">
            注册完成后，配置文件会保存在:
        </Paragraph>
        <ul className="space-y-2 text-gray-600 dark:text-slate-400">
            <li>
                <Text code>{CONFIG_PATH}</Text> - 配置文件路径
            </li>
            <li>
                您可以手动编辑此文件来修改配置，修改后需要重启服务生效
            </li>
        </ul>
    </Card>
);
