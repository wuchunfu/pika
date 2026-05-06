import {useQuery} from '@tanstack/react-query';
import {
    getAgent,
    getAgentLatestMetrics,
    getAgentMetrics,
    getAvailableNetworkInterfaces,
    type GetAgentMetricsRequest,
    type MetricsAggregation,
} from '@/api/agent';

interface UseMetricsQueryOptions {
    agentId: string;
    type: GetAgentMetricsRequest['type'];
    range?: string;
    start?: number;
    end?: number;
    interfaceName?: string;
    aggregation?: MetricsAggregation;
    // 自动刷新间隔（毫秒），不传或 0 表示不自动刷新
    refetchIntervalMs?: number;
}

/**
 * 查询 Agent 基础信息
 * @param agentId Agent ID
 * @returns Agent 查询结果
 */
export const useAgentQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId],
        queryFn: () => getAgent(agentId!),
        enabled: !!agentId,
        staleTime: 60000, // 1分钟缓存
    });
};

/**
 * 查询 Agent 最新指标
 * @param agentId Agent ID
 * @param intervalMs 自动刷新间隔（毫秒），默认 5000
 * @returns 最新指标查询结果
 */
export const useLatestMetricsQuery = (agentId?: string, intervalMs: number = 5000) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', 'latest'],
        queryFn: () => getAgentLatestMetrics(agentId!),
        enabled: !!agentId,
        refetchInterval: intervalMs > 0 ? intervalMs : false,
    });
};

/**
 * 查询 Agent 历史指标数据
 * @param options 查询选项
 * @returns 历史指标查询结果
 */
export const useMetricsQuery = ({agentId, type, range, start, end, interfaceName, aggregation, refetchIntervalMs}: UseMetricsQueryOptions) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', type, range, start, end, interfaceName, aggregation],
        queryFn: () =>
            getAgentMetrics({
                agentId,
                type,
                range,
                start,
                end,
                interface: interfaceName,
                aggregation,
            }),
        enabled: !!agentId,
        refetchInterval: refetchIntervalMs && refetchIntervalMs > 0 ? refetchIntervalMs : false,
    });
};

/**
 * 查询 Agent 可用网卡列表
 * @param agentId Agent ID
 * @returns 网卡列表查询结果
 */
export const useNetworkInterfacesQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId, 'network-interfaces'],
        queryFn: () => getAvailableNetworkInterfaces(agentId!),
        enabled: !!agentId,
    });
};
