import {useMemo} from 'react';
import {MemoryStick} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder, CustomTooltip} from '@/components/common';
import {useMetricsQuery} from '@/hooks/server/queries';
import {ChartContainer} from './ChartContainer';
import {formatChartTime} from '@/utils/util';

interface MemoryChartProps {
    agentId: string;
    timeRange: string;
}

/**
 * 内存使用率图表组件
 */
export const MemoryChart = ({agentId, timeRange}: MemoryChartProps) => {
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'memory',
        range: timeRange,
    });

    // 数据转换
    const chartData = useMemo(() => {
        const memorySeries = metricsResponse?.data.series?.find(s => s.name === 'usage');
        if (!memorySeries) return [];

        return memorySeries.data.map((point) => ({
            time: formatChartTime(point.timestamp, timeRange),
            usage: Number(point.value.toFixed(2)),
            timestamp: point.timestamp,
        }));
    }, [metricsResponse, timeRange]);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="内存使用率" icon={MemoryStick}>
                <ChartPlaceholder variant="dark"/>
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
                        <CartesianGrid stroke="currentColor" strokeDasharray="4 4" className="stroke-cyan-900/30"/>
                        <XAxis
                            dataKey="time"
                            stroke="currentColor"
                            angle={-15}
                            textAnchor="end"
                            className="text-xs text-cyan-600 font-mono"
                        />
                        <YAxis
                            domain={[0, 100]}
                            stroke="currentColor"
                            className="stroke-cyan-600 text-xs"
                            tickFormatter={(value) => `${value}%`}
                        />
                        <Tooltip content={<CustomTooltip unit="%" variant="dark"/>}/>
                        <Area
                            type="monotone"
                            dataKey="usage"
                            name="内存使用率"
                            stroke="#10b981"
                            strokeWidth={2}
                            fill="url(#memoryAreaGradient)"
                            activeDot={{r: 3}}
                        />
                    </AreaChart>
                </ResponsiveContainer>
            ) : (
                <ChartPlaceholder variant="dark"/>
            )}
        </ChartContainer>
    );
};
