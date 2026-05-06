import {memo, useMemo} from 'react';
import {Zap} from 'lucide-react';
import {CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder} from '@portal/components/ChartPlaceholder';
import {CustomTooltip} from '@portal/components/CustomTooltip';
import {useMetricsQuery} from '@portal/hooks/server';
import {LIVE_INITIAL_RANGE} from '@portal/constants/time';
import {ChartContainer} from './ChartContainer';
import {formatChartTime} from '@/lib/format.ts';

interface GpuChartProps {
    agentId: string;
    timeRange: string;
    start?: number;
    end?: number;
    isLive?: boolean;
}

/**
 * GPU 使用率与温度图表组件
 * 使用双 Y 轴显示使用率和温度
 *
 * 实时模式下不走 livePoint 追加（多卡聚合复杂），改为 5s 轮询 refetch；用 React.memo
 * 阻断 ServerDetail 1s 节奏的级联重渲染，避免 Recharts 每秒空转。
 */
const GpuChartImpl = ({agentId, timeRange, start, end, isLive}: GpuChartProps) => {
    const rangeMs = start !== undefined && end !== undefined ? end - start : undefined;
    const effectiveRange = isLive ? LIVE_INITIAL_RANGE : timeRange;
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'gpu',
        range: start !== undefined && end !== undefined ? undefined : effectiveRange,
        start,
        end,
        refetchIntervalMs: isLive ? 5000 : undefined,
    });

    // 数据转换
    const chartData = useMemo(() => {
        if (!metricsResponse?.data.series || metricsResponse.data.series?.length === 0) return [];

        const timeMap = new Map<number, { timestamp: number; utilization?: number; temperature?: number }>();

        const utilizationSeries = metricsResponse.data?.series?.find(s => s.name === 'utilization');
        const temperatureSeries = metricsResponse.data?.series?.find(s => s.name === 'temperature');

        utilizationSeries?.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.utilization = Number(point.value.toFixed(2));
            } else {
                timeMap.set(point.timestamp, {
                    timestamp: point.timestamp,
                    utilization: Number(point.value.toFixed(2)),
                });
            }
        });

        temperatureSeries?.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.temperature = Number(point.value.toFixed(2));
            } else {
                timeMap.set(point.timestamp, {
                    timestamp: point.timestamp,
                    temperature: Number(point.value.toFixed(2)),
                });
            }
        });

        return Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp);
    }, [metricsResponse]);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="GPU 使用率与温度" icon={Zap}>
                <ChartPlaceholder/>
            </ChartContainer>
        );
    }

    // 如果没有 GPU 数据，不渲染组件
    if (chartData.length === 0) {
        return null;
    }

    return (
        <ChartContainer title="GPU 使用率与温度" icon={Zap}>
            <ResponsiveContainer width="100%" height={220}>
                <LineChart data={chartData}>
                    <CartesianGrid stroke="currentColor" strokeDasharray="4 4" className="stroke-slate-200 dark:stroke-cyan-900/30"/>
                    <XAxis
                        dataKey="timestamp"
                        type="number"
                        scale="time"
                        domain={['dataMin', 'dataMax']}
                        tickFormatter={(value) => formatChartTime(Number(value), timeRange, rangeMs)}
                        stroke="currentColor"
                        className="stroke-gray-400 dark:stroke-cyan-600"
                        style={{fontSize: '12px'}}
                    />
                    <YAxis
                        yAxisId="left"
                        stroke="currentColor"
                        className="stroke-gray-400 dark:stroke-cyan-600"
                        style={{fontSize: '12px'}}
                        tickFormatter={(value) => `${value}%`}
                    />
                    <YAxis
                        yAxisId="right"
                        orientation="right"
                        stroke="currentColor"
                        className="stroke-gray-400 dark:stroke-cyan-600"
                        style={{fontSize: '12px'}}
                        tickFormatter={(value) => `${value}°C`}
                    />
                    <Tooltip content={<CustomTooltip unit=""/>}/>
                    <Legend/>
                    <Line
                        yAxisId="left"
                        type="monotone"
                        dataKey="utilization"
                        name="使用率 (%)"
                        stroke="#7c3aed"
                        strokeWidth={2}
                        dot={false}
                        activeDot={{r: 3}}
                        connectNulls
                        isAnimationActive={!isLive}
                    />
                    <Line
                        yAxisId="right"
                        type="monotone"
                        dataKey="temperature"
                        name="温度 (°C)"
                        stroke="#f97316"
                        strokeWidth={2}
                        dot={false}
                        activeDot={{r: 3}}
                        connectNulls
                        isAnimationActive={!isLive}
                    />
                </LineChart>
            </ResponsiveContainer>
        </ChartContainer>
    );
};

export const GpuChart = memo(GpuChartImpl);
