import {Area, AreaChart, ResponsiveContainer} from 'recharts';

interface MiniChartProps {
    data: Array<{ time: string; value: number }>;
    lastValue?: number;
    id: string;
}

export const MiniChart = ({data, lastValue, id}: MiniChartProps) => {
    if (!data || data.length === 0) {
        return (
            <div className="h-16 w-full flex items-center justify-center text-xs text-slate-400 dark:text-slate-500">
                暂无数据
            </div>
        );
    }

    const color = lastValue <= 200 ? '#22d3ee' : '#fbbf24';

    return (
        <div className="h-16 w-full -mb-2">
            <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data}>
                    <defs>
                        <linearGradient id={`colorLatency-${id}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={color} stopOpacity={0.3}/>
                            <stop offset="100%" stopColor={color} stopOpacity={0}/>
                        </linearGradient>
                        <filter id="glow" height="300%" width="300%" x="-75%" y="-75%">
                            <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
                            <feMerge>
                                <feMergeNode in="coloredBlur"/>
                                <feMergeNode in="SourceGraphic"/>
                            </feMerge>
                        </filter>
                    </defs>
                    <Area
                        type="monotone"
                        dataKey="value"
                        stroke={color}
                        fill={`url(#colorLatency-${id})`}
                        strokeWidth={2}
                        filter="url(#glow)"
                    />
                </AreaChart>
            </ResponsiveContainer>
        </div>
    );
};
