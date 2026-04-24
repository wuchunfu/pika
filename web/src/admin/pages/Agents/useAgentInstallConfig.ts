import { useEffect, useMemo, useState } from 'react';
import { App } from 'antd';
import { listApiKeys, getApiKeyRaw } from '@/api/apiKey.ts';
import { getServerUrl } from '@/api/agent.ts';
import type { ApiKey } from '@/types';
import type { ApiKeyOption } from './AgentInstallShared';

type UseAgentInstallConfigResult = {
    apiKeys: ApiKey[];
    selectedApiKeyId: string;
    setSelectedApiKeyId: (id: string) => void;
    selectedApiKey: string; // 完整 key 值，用于安装命令
    customAgentName: string;
    setCustomAgentName: (value: string) => void;
    loading: boolean;
    backendServerUrl: string;
    apiKeyOptions: ApiKeyOption[];
};

export const useAgentInstallConfig = (): UseAgentInstallConfigResult => {
    const [selectedApiKeyId, setSelectedApiKeyId] = useState<string>('');
    const [customAgentName, setCustomAgentName] = useState<string>('');
    const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [backendServerUrl, setBackendServerUrl] = useState<string>('');
    const [rawKey, setRawKey] = useState<string>('');
    const { message } = App.useApp();

    // 加载 API 密钥列表（key 已遮蔽）
    useEffect(() => {
        const fetchApiKeys = async () => {
            setLoading(true);
            try {
                const keys = await listApiKeys();
                const enabledKeys = keys.data?.items.filter(k => k.enabled) || [];
                setApiKeys(enabledKeys);
                setSelectedApiKeyId((prev) => prev || enabledKeys[0]?.id || '');
            } catch (error) {
                console.error('Failed to load API keys:', error);
                message.error('加载 API Token 失败');
            } finally {
                setLoading(false);
            }
        };
        void fetchApiKeys();
    }, [message]);

    // 加载服务端地址
    useEffect(() => {
        const fetchServerUrl = async () => {
            try {
                const response = await getServerUrl();
                const backendUrl = response.data.serverUrl || '';
                setBackendServerUrl(backendUrl);
            } catch (error) {
                console.error('Failed to load server URL:', error);
            }
        };

        void fetchServerUrl();
    }, []);

    // 当选中的密钥变化时，获取完整 key 值
    useEffect(() => {
        if (!selectedApiKeyId) {
            setRawKey('');
            return;
        }
        const fetchRawKey = async () => {
            try {
                const response = await getApiKeyRaw(selectedApiKeyId);
                setRawKey(response.data.key || '');
            } catch (error) {
                console.error('Failed to load raw API key:', error);
                setRawKey('');
            }
        };
        void fetchRawKey();
    }, [selectedApiKeyId]);

    const apiKeyOptions = useMemo(
        () => apiKeys.map(key => ({
            label: `${key.name} (${key.key.substring(0, 8)}...)`,
            value: key.id,
        })),
        [apiKeys]
    );

    return {
        apiKeys,
        selectedApiKeyId,
        setSelectedApiKeyId,
        selectedApiKey: rawKey,
        customAgentName,
        setCustomAgentName,
        loading,
        backendServerUrl,
        apiKeyOptions,
    };
};
