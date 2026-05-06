import type {TimeRangeOption} from '@/api/property.ts';

// 实时模式标识，前端内部值；查询历史时映射到 LIVE_INITIAL_RANGE
export const LIVE_RANGE = 'live';
// 实时模式下首次拉取的历史窗口
export const LIVE_INITIAL_RANGE = '2m';
// 实时模式滑动窗口长度（毫秒）—— 与 LIVE_INITIAL_RANGE 保持一致
export const LIVE_WINDOW_MS = 2 * 60 * 1000;

// 服务器详情页时间范围选项
export const SERVER_TIME_RANGE_OPTIONS: TimeRangeOption[] = [
    {label: '实时', value: LIVE_RANGE},
    {label: '15分钟', value: '15m'},
    {label: '30分钟', value: '30m'},
    {label: '1小时', value: '1h'},
    {label: '3小时', value: '3h'},
    {label: '6小时', value: '6h'},
    {label: '12小时', value: '12h'},
    {label: '1天', value: '1d'},
    {label: '3天', value: '3d'},
    {label: '7天', value: '7d'},
];

// 监控详情页时间范围选项
export const MONITOR_TIME_RANGE_OPTIONS: TimeRangeOption[] = [
    {label: '12小时', value: '12h'},
    {label: '1天', value: '1d'},
    {label: '3天', value: '3d'},
    {label: '7天', value: '7d'},
];
