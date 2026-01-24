import {AlertCircle} from 'lucide-react';
import {cn} from '@/lib/utils';

interface EmptyStateProps {
    message?: string;
    showBackButton?: boolean;
    className?: string;
}

export const EmptyState = ({message = '监控数据不存在', showBackButton = false, className}: EmptyStateProps) => {
    return (
        <div className={cn(
            "flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-900",
            className
        )}>
            <div className="flex flex-col items-center gap-3 text-center">
                <div className="flex h-16 w-16 items-center justify-center rounded-full bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500">
                    <AlertCircle className="h-8 w-8"/>
                </div>
                <p className="text-sm font-mono text-slate-600 dark:text-slate-400">
                    {message}
                </p>
                {showBackButton && (
                    <button
                        onClick={() => window.history.back()}
                        className="mt-4 text-sm text-blue-600 dark:text-blue-400 hover:underline"
                    >
                        返回监控列表
                    </button>
                )}
            </div>
        </div>
    );
};
