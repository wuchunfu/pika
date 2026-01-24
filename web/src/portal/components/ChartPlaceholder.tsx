import {TrendingUp} from 'lucide-react';
import {cn} from '@/lib/utils';

interface ChartPlaceholderProps {
    icon?: typeof TrendingUp;
    title?: string;
    subtitle?: string;
    heightClass?: string;
    className?: string;
}

export const ChartPlaceholder = ({
                                     icon: Icon = TrendingUp,
                                     title = '暂无数据',
                                     subtitle = '等待采集新数据后展示图表',
                                     heightClass = 'h-52',
                                     className,
                                 }: ChartPlaceholderProps) => {
    return (
        <div className={cn(
            "flex items-center justify-center rounded-lg border border-dashed border-slate-200 dark:border-slate-700 text-sm text-slate-500 dark:text-slate-400",
            heightClass,
            className
        )}>
            <div className="text-center">
                <Icon className="mx-auto mb-3 h-10 w-10 text-slate-300 dark:text-slate-600"/>
                <p className="font-medium">{title}</p>
                {subtitle && (
                    <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
                        {subtitle}
                    </p>
                )}
            </div>
        </div>
    );
};
