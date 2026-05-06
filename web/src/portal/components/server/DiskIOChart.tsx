import {useMemo} from 'react';
import {HardDrive} from 'lucide-react';
import {Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {ChartPlaceholder} from '@portal/components/ChartPlaceholder';
import {CustomTooltip} from '@portal/components/CustomTooltip';
import {useMetricsQuery} from '@portal/hooks/server';
import {useLiveBuffer} from '@portal/hooks/useLiveBuffer';
import {LIVE_INITIAL_RANGE, LIVE_WINDOW_MS} from '@portal/constants/time';
import {ChartContainer} from './ChartContainer';
import {formatChartTime} from '@/lib/format.ts';
import type {LatestMetrics} from '@/types';

interface DiskIOChartProps {
    agentId: string;
    timeRange: string;
    start?: number;
    end?: number;
    isLive?: boolean;
    latestMetrics?: LatestMetrics | null;
}

interface DiskIOPoint {
    timestamp: number;
    read: number;
    write: number;
}

const toMB = (bytes: number) => Number((bytes / 1024 / 1024).toFixed(2));

/**
 * 磁盘 I/O 图表组件
 */
export const DiskIOChart = ({agentId, timeRange, start, end, isLive, latestMetrics}: DiskIOChartProps) => {
    const rangeMs = start !== undefined && end !== undefined ? end - start : undefined;
    const effectiveRange = isLive ? LIVE_INITIAL_RANGE : timeRange;
    // 数据查询
    const {data: metricsResponse, isLoading} = useMetricsQuery({
        agentId,
        type: 'disk_io',
        range: start !== undefined && end !== undefined ? undefined : effectiveRange,
        start,
        end,
    });

    // 历史数据
    const initialData = useMemo<DiskIOPoint[]>(() => {
        if (!metricsResponse?.data.series || metricsResponse.data.series?.length === 0) return [];

        const readSeries = metricsResponse.data.series?.find(s => s.name === 'read');
        const writeSeries = metricsResponse.data.series?.find(s => s.name === 'write');

        if (!readSeries || !writeSeries) return [];

        const timeMap = new Map<number, DiskIOPoint>();

        readSeries.data.forEach(point => {
            timeMap.set(point.timestamp, {
                timestamp: point.timestamp,
                read: toMB(point.value),
                write: 0,
            });
        });

        writeSeries.data.forEach(point => {
            const existing = timeMap.get(point.timestamp);
            if (existing) {
                existing.write = toMB(point.value);
            } else {
                timeMap.set(point.timestamp, {
                    timestamp: point.timestamp,
                    read: 0,
                    write: toMB(point.value),
                });
            }
        });

        return Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp);
    }, [metricsResponse]);

    // 实时点：用 latestMetrics.timestamp 作为同批次时间锚
    const livePoint = useMemo<DiskIOPoint | null>(() => {
        if (!isLive || !latestMetrics?.diskIO || !latestMetrics.timestamp) return null;
        return {
            timestamp: latestMetrics.timestamp,
            read: toMB(latestMetrics.diskIO.totalReadBytesRate),
            write: toMB(latestMetrics.diskIO.totalWriteBytesRate),
        };
    }, [isLive, latestMetrics]);

    const chartData = useLiveBuffer(initialData, !!isLive, livePoint, LIVE_WINDOW_MS, agentId);

    // 渲染
    if (isLoading) {
        return (
            <ChartContainer title="磁盘 I/O (MB/s)" icon={HardDrive}>
                <ChartPlaceholder/>
            </ChartContainer>
        );
    }

    return (
        <ChartContainer title="磁盘 I/O (MB/s)" icon={HardDrive}>
            {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={250}>
                    <AreaChart data={chartData}>
                        <defs>
                            <linearGradient id="colorDiskRead" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#2C70F6" stopOpacity={0.3}/>
                                <stop offset="95%" stopColor="#2C70F6" stopOpacity={0}/>
                            </linearGradient>
                            <linearGradient id="colorDiskWrite" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#6FD598" stopOpacity={0.3}/>
                                <stop offset="95%" stopColor="#6FD598" stopOpacity={0}/>
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
                            height={45}
                        />
                        <YAxis
                            stroke="currentColor"
                            className="stroke-gray-400 dark:stroke-cyan-600 text-xs"
                            tickFormatter={(value) => `${value} MB`}
                        />
                        <Tooltip content={<CustomTooltip unit=" MB" timeFormat={isLive ? 'HH:mm:ss' : undefined}/>}/>
                        <Legend/>
                        <Area
                            type="monotone"
                            dataKey="read"
                            name="读取"
                            stroke="#2C70F6"
                            strokeWidth={2}
                            fill="url(#colorDiskRead)"
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                        <Area
                            type="monotone"
                            dataKey="write"
                            name="写入"
                            stroke="#6FD598"
                            strokeWidth={2}
                            fill="url(#colorDiskWrite)"
                            activeDot={{r: 3}}
                            connectNulls
                            isAnimationActive={!isLive}
                        />
                    </AreaChart>
                </ResponsiveContainer>
            ) : (
                <ChartPlaceholder subtitle="暂无磁盘 I/O 采集数据"/>
            )}
        </ChartContainer>
    );
};
