import dayjs from 'dayjs';
import {cn} from '@/lib/utils';

type CustomTooltipProps = {
    active?: boolean;
    payload?: Array<{
        name?: string;
        value?: number;
        color?: string;
        dataKey?: string;
        payload?: {
            timestamp?: number | string;
            [key: string]: unknown;
        };
    }>;
    label?: string | number;
    unit?: string;
    className?: string;
};

export const CustomTooltip = ({active, payload, label, unit = '%', className}: CustomTooltipProps) => {
    if (!active || !payload || payload.length === 0) {
        return null;
    }

    // 从 payload 中获取完整的时间戳信息（如果有的话）
    const fullTimestamp = payload[0]?.payload?.timestamp;
    const displayLabel = fullTimestamp
        ? dayjs(fullTimestamp).format('MM-DD HH:mm')
        : label;

    return (
        <div className={cn(
            "rounded-md border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-lg px-3 py-2 text-xs",
            className
        )}>
            <p className="font-semibold text-slate-700 dark:text-white mb-2">
                {displayLabel}
            </p>
            <div className="space-y-1">
                {payload.map((entry, index) => {
                    if (!entry) {
                        return null;
                    }

                    const dotColor = entry.color ?? '#6366f1';
                    const title = entry.name ?? entry.dataKey ?? `系列 ${index + 1}`;
                    const value = typeof entry.value === 'number'
                        ? Number.isFinite(entry.value)
                            ? entry.value.toFixed(2)
                            : '-'
                        : entry.value;

                    return (
                        <p key={`${entry.dataKey ?? index}`} className="flex items-center gap-2 text-xs">
                            <span
                                className="inline-block h-2 w-2 rounded-full"
                                style={{backgroundColor: dotColor}}
                            />
                            <span className="text-slate-600 dark:text-slate-400">
                                {title}: <span className="font-semibold text-slate-900 dark:text-white">{value}{unit}</span>
                            </span>
                        </p>
                    );
                })}
            </div>
        </div>
    );
};
