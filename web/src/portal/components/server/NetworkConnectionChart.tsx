import {useMemo} from 'react';
import {Network} from 'lucide-react';
import {CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder} from '@portal/components/ChartPlaceholder';
import {CustomTooltip} from '@portal/components/CustomTooltip';
import {useMetricsQuery} from '@portal/hooks/server';
import {useLiveBuffer} from '@portal/hooks/useLiveBuffer';
import {LIVE_INITIAL_RANGE, LIVE_WINDOW_MS} from '@portal/constants/time';
import {ChartContainer} from './ChartContainer';
import {formatChartTime} from '@/lib/format.ts';
import type {LatestMetrics} from '@/types';

interface NetworkConnectionChartProps {
    agentId: string;
    timeRange: string;
    start?: number;
    end?: number;
    isLive?: boolean;
    latestMetrics?: LatestMetrics | null;
}

interface ConnPoint {
    timestamp: number;
    established: number;
    time_wait: number;
    close_wait: number;
    listen: number;
}

/**
 * 网络连接统计图表组件
 * 实时模式下与 CPU/内存等快指标一样按 1s 节奏 append，便于压测观察
 */
export const NetworkConnectionChart = ({agentId, timeRange, start, end, isLive, latestMetrics}: NetworkConnectionChartProps) => {
    const rangeMs = start !== undefined && end !== undefined ? end - start : undefined;
    const effectiveRange = isLive ? LIVE_INITIAL_RANGE : timeRange;
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'network_connection',
        range: start !== undefined && end !== undefined ? undefined : effectiveRange,
        start,
        end,
    });

    // 历史数据
    const initialData = useMemo<ConnPoint[]>(() => {
        if (!metricsResponse?.data.series || metricsResponse.data.series?.length === 0) return [];

        const timeMap = new Map<number, ConnPoint>();

        metricsResponse.data.series?.forEach(series => {
            const stateName = series.name;
            series.data.forEach(point => {
                if (!timeMap.has(point.timestamp)) {
                    timeMap.set(point.timestamp, {
                        timestamp: point.timestamp,
                        established: 0,
                        time_wait: 0,
                        close_wait: 0,
                        listen: 0,
                    });
                }
                const existing = timeMap.get(point.timestamp)!;
                (existing as Record<string, number>)[stateName] = Number(point.value.toFixed(0));
            });
        });

        return Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp);
    }, [metricsResponse]);

    // 实时点
    const livePoint = useMemo<ConnPoint | null>(() => {
        if (!isLive || !latestMetrics?.networkConnection || !latestMetrics.timestamp) return null;
        const c = latestMetrics.networkConnection;
        return {
            timestamp: latestMetrics.timestamp,
            established: c.established ?? 0,
            time_wait: c.timeWait ?? 0,
            close_wait: c.closeWait ?? 0,
            listen: c.listen ?? 0,
        };
    }, [isLive, latestMetrics]);

    const chartData = useLiveBuffer(initialData, !!isLive, livePoint, LIVE_WINDOW_MS, agentId);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="网络连接统计" icon={Network}>
                <ChartPlaceholder/>
            </ChartContainer>
        );
    }

    return (
        <ChartContainer title="网络连接统计" icon={Network}>
            {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={250}>
                    <LineChart data={chartData}>
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
                            height={45}
                        />
                        <YAxis
                            stroke="currentColor"
                            className="stroke-gray-400 dark:stroke-cyan-600 text-xs"
                        />
                        <Tooltip content={<CustomTooltip unit="" timeFormat={isLive ? 'HH:mm:ss' : undefined}/>}/>
                        <Legend/>
                        <Line
                            type="monotone"
                            dataKey="established"
                            name="ESTABLISHED"
                            stroke="#10b981"
                            strokeWidth={2}
                            dot={false}
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                        <Line
                            type="monotone"
                            dataKey="time_wait"
                            name="TIME_WAIT"
                            stroke="#f59e0b"
                            strokeWidth={2}
                            dot={false}
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                        <Line
                            type="monotone"
                            dataKey="close_wait"
                            name="CLOSE_WAIT"
                            stroke="#ef4444"
                            strokeWidth={2}
                            dot={false}
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                        <Line
                            type="monotone"
                            dataKey="listen"
                            name="LISTEN"
                            stroke="#3b82f6"
                            strokeWidth={2}
                            dot={false}
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                    </LineChart>
                </ResponsiveContainer>
            ) : (
                <ChartPlaceholder subtitle="暂无网络连接统计数据"/>
            )}
        </ChartContainer>
    );
};
