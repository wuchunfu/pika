import dayjs from 'dayjs';

export const formatTime = (ms: number): string => {
    if (!ms || ms <= 0) return '0 ms';
    if (ms < 1000) return `${ms.toFixed(0)} ms`;
    return `${(ms / 1000).toFixed(2)} s`;
};

export const formatDate = (timestamp: number): string => {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
};

export const formatDateTime = (value: string | number | undefined | null): string => {
    if (value === undefined || value === null || value === '') {
        return '-';
    }
    return dayjs(value).format('YYYY-MM-DD HH:mm:ss');
};

export const formatSpeed = (bytesPerSecond: number): string => {
    if (!bytesPerSecond || bytesPerSecond <= 0) return '0 B/s';
    const k = 1024;
    const sizes = ['B/s', 'K/s', 'M/s', 'G/s', 'T/s'];
    const i = Math.min(Math.floor(Math.log(bytesPerSecond) / Math.log(k)), sizes.length - 1);
    const value = bytesPerSecond / Math.pow(k, i);
    const decimals = value >= 100 ? 0 : value >= 10 ? 1 : 2;
    return `${value.toFixed(decimals)} ${sizes[i]}`;
};

export const formatBytes = (bytes: number | undefined | null): string => {
    if (!bytes || bytes <= 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
};

export const formatPercentValue = (value: number | undefined | null): string => {
    if (value === undefined || value === null || Number.isNaN(value)) return '0.0';
    return value.toFixed(1);
};

export const formatUptime = (seconds: number | undefined | null): string => {
    if (seconds === undefined || seconds === null) return '-';
    if (seconds <= 0) return '0 秒';

    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    const parts: string[] = [];

    // 智能显示：只显示最重要的两个单位，避免文本过长
    if (days > 0) {
        parts.push(`${days} 天`);
        if (hours > 0) parts.push(`${hours} 小时`);
    } else if (hours > 0) {
        parts.push(`${hours} 小时`);
        if (minutes > 0) parts.push(`${minutes} 分钟`);
    } else if (minutes > 0) {
        parts.push(`${minutes} 分钟`);
    }

    return parts.length > 0 ? parts.join(' ') : '不到 1 分钟';
};

/**
 * 根据时间范围格式化图表时间标签
 * @param timestamp - 时间戳（毫秒）
 * @param timeRange - 时间范围（如 '1h', '1d', '3d', '7d'）
 * @returns 格式化后的时间字符串
 */
export const formatChartTime = (timestamp: number, timeRange: string): string => {
    const date = new Date(timestamp);
    
    // 解析时间范围，判断是否超过1天
    const isLongRange = timeRange.endsWith('d') && parseInt(timeRange) > 1;
    
    if (isLongRange) {
        // 超过1天：显示 "月/日 时:分"
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        return `${month}/${day} ${hours}:${minutes}`;
    } else {
        // 1天及以内：只显示 "时:分"
        return date.toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit',
        });
    }
};