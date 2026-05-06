import {useMemo} from 'react';
import {MemoryStick} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder} from '@portal/components/ChartPlaceholder';
import {CustomTooltip} from '@portal/components/CustomTooltip';
import {useMetricsQuery} from '@portal/hooks/server';
import {useLiveBuffer} from '@portal/hooks/useLiveBuffer';
import {LIVE_INITIAL_RANGE, LIVE_WINDOW_MS} from '@portal/constants/time';
import {ChartContainer} from './ChartContainer';
import {formatChartTime} from '@/lib/format.ts';
import type {LatestMetrics} from '@/types';

interface MemoryChartProps {
    agentId: string;
    timeRange: string;
    start?: number;
    end?: number;
    isLive?: boolean;
    latestMetrics?: LatestMetrics | null;
}

interface MemoryPoint {
    timestamp: number;
    usage: number;
}

/**
 * 内存使用率图表组件
 */
export const MemoryChart = ({agentId, timeRange, start, end, isLive, latestMetrics}: MemoryChartProps) => {
    const rangeMs = start !== undefined && end !== undefined ? end - start : undefined;
    const effectiveRange = isLive ? LIVE_INITIAL_RANGE : timeRange;
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'memory',
        range: start !== undefined && end !== undefined ? undefined : effectiveRange,
        start,
        end,
    });

    // 历史数据
    const initialData = useMemo<MemoryPoint[]>(() => {
        const memorySeries = metricsResponse?.data.series?.find(s => s.name === 'usage');
        if (!memorySeries) return [];
        return memorySeries.data.map((point) => ({
            usage: Number(point.value.toFixed(2)),
            timestamp: point.timestamp,
        }));
    }, [metricsResponse]);

    // 实时点
    const livePoint = useMemo<MemoryPoint | null>(() => {
        if (!isLive || !latestMetrics?.memory || !latestMetrics.timestamp) return null;
        const usage = latestMetrics.memory.usagePercent;
        if (typeof usage !== 'number' || !Number.isFinite(usage)) return null;
        return {timestamp: latestMetrics.timestamp, usage: Number(usage.toFixed(2))};
    }, [isLive, latestMetrics]);

    const chartData = useLiveBuffer(initialData, !!isLive, livePoint, LIVE_WINDOW_MS, agentId);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="内存使用率" icon={MemoryStick}>
                <ChartPlaceholder/>
            </ChartContainer>
        );
    }

    return (
        <ChartContainer title="内存使用率" icon={MemoryStick}>
            {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={220}>
                    <AreaChart data={chartData}>
                        <defs>
                            <linearGradient id="memoryAreaGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#10b981" stopOpacity={0.4}/>
                                <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                            </linearGradient>
                        </defs>
                        <CartesianGrid stroke="currentColor" strokeDasharray="4 4" className="stroke-slate-200 dark:stroke-cyan-900/30"/>
                        <XAxis
                            dataKey="timestamp"
                            type="number"
                            scale="time"
                            domain={['dataMin', 'dataMax']}
                            tickFormatter={(value) => formatChartTime(Number(value), timeRange, rangeMs)}
                            stroke="currentColor"
                            angle={-15}
                            textAnchor="end"
                            className="text-xs text-gray-600 dark:text-cyan-500 font-mono"
                        />
                        <YAxis
                            domain={[0, 100]}
                            stroke="currentColor"
                            className="stroke-gray-400 dark:stroke-cyan-600 text-xs"
                            tickFormatter={(value) => `${value}%`}
                        />
                        <Tooltip content={<CustomTooltip unit="%" timeFormat={isLive ? 'HH:mm:ss' : undefined}/>}/>
                        <Area
                            type="monotone"
                            dataKey="usage"
                            name="内存使用率"
                            stroke="#10b981"
                            strokeWidth={2}
                            fill="url(#memoryAreaGradient)"
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                    </AreaChart>
                </ResponsiveContainer>
            ) : (
                <ChartPlaceholder/>
            )}
        </ChartContainer>
    );
};
