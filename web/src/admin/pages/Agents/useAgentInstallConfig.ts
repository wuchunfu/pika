import { useEffect, useMemo, useState } from 'react';
import { App } from 'antd';
import { listApiKeys } from '@/api/apiKey.ts';
import { getServerUrl } from '@/api/agent.ts';
import type { ApiKey } from '@/types';
import type { ApiKeyOption } from './AgentInstallShared';

type UseAgentInstallConfigResult = {
    apiKeys: ApiKey[];
    selectedApiKey: string;
    setSelectedApiKey: (value: string) => void;
    customAgentName: string;
    setCustomAgentName: (value: string) => void;
    loading: boolean;
    backendServerUrl: string;
    apiKeyOptions: ApiKeyOption[];
};

export const useAgentInstallConfig = (): UseAgentInstallConfigResult => {
    const [selectedApiKey, setSelectedApiKey] = useState<string>('');
    const [customAgentName, setCustomAgentName] = useState<string>('');
    const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [backendServerUrl, setBackendServerUrl] = useState<string>('');
    const { message } = App.useApp();

    useEffect(() => {
        const fetchApiKeys = async () => {
            setLoading(true);
            try {
                const keys = await listApiKeys();
                const enabledKeys = keys.data?.items.filter(k => k.enabled) || [];
                setApiKeys(enabledKeys);
                setSelectedApiKey((prev) => prev || enabledKeys[0]?.key || '');
            } catch (error) {
                console.error('Failed to load API keys:', error);
                message.error('加载 API Token 失败');
            } finally {
                setLoading(false);
            }
        };
        void fetchApiKeys();
    }, [message]);

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

    const apiKeyOptions = useMemo(
        () => apiKeys.map(key => ({
            label: `${key.name} (${key.key.substring(0, 8)}...)`,
            value: key.key,
        })),
        [apiKeys]
    );

    return {
        apiKeys,
        selectedApiKey,
        setSelectedApiKey,
        customAgentName,
        setCustomAgentName,
        loading,
        backendServerUrl,
        apiKeyOptions,
    };
};
