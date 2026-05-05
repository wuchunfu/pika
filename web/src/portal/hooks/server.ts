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
 * 自动每 5 秒刷新一次
 * @param agentId Agent ID
 * @returns 最新指标查询结果
 */
export const useLatestMetricsQuery = (agentId?: string) => {
    return useQuery({
        queryKey: ['agent', agentId, 'metrics', 'latest'],
        queryFn: () => getAgentLatestMetrics(agentId!),
        enabled: !!agentId,
        refetchInterval: 5000, // 5秒自动刷新
    });
};

/**
 * 查询 Agent 历史指标数据
 * @param options 查询选项
 * @returns 历史指标查询结果
 */
export const useMetricsQuery = ({agentId, type, range, start, end, interfaceName, aggregation}: UseMetricsQueryOptions) => {
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
        // refetchInterval: 30000, // 30秒自动刷新
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
