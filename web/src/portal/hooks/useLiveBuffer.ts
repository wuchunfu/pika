import {useEffect, useRef, useState} from 'react';

/**
 * 维护一个"实时滑动窗口"数据缓冲。
 *
 * - 非实时模式：直接返回 initial。
 * - 实时模式：第一次进入并拿到 initial 后用它播种缓冲；之后只追加 livePoint，
 *   不再因为 initial 引用变化（例如父组件重渲染产生新数组）而重置已有数据。
 * - 若 initial 抵达前已经有 livePoint 进来，播种时仅把比已有最早点更早的历史点 prepend，
 *   不丢已经追加的实时点。
 * - resetKey 变化时（如切换网卡 / 切换 agent）会强制清空并等待新 initial 重新播种。
 * - 同一时间戳的 livePoint 被忽略，避免轮询重复触发的重复点。
 * - 早于 ts - windowMs 的点会被丢弃，形成滑动窗口。
 */
export function useLiveBuffer<T extends { timestamp: number }>(
    initial: T[],
    isLive: boolean,
    livePoint: T | null,
    windowMs: number,
    resetKey?: unknown,
): T[] {
    const [buffer, setBuffer] = useState<T[]>([]);
    const lastTsRef = useRef<number>(0);
    const seededRef = useRef<boolean>(false);

    // isLive 切换或 resetKey 变化（数据语义变更）时复位播种状态
    useEffect(() => {
        seededRef.current = false;
        lastTsRef.current = 0;
        setBuffer([]);
    }, [isLive, resetKey]);

    // 进入实时模式后，等历史数据到位再播种（每次复位后只播种一次）
    // 依赖里加 resetKey：万一 RQ 命中旧缓存导致 initial 引用未变也能重新播种
    useEffect(() => {
        if (!isLive) return;
        if (seededRef.current) return;
        if (!initial || initial.length === 0) return;
        seededRef.current = true;

        const initLastTs = initial[initial.length - 1].timestamp;
        setBuffer(prev => {
            if (prev.length === 0) {
                return initial.slice();
            }
            // livePoint 已经先到了：只把比已有最早点更早的历史点 prepend，避免覆盖实时点
            const earliestExisting = prev[0].timestamp;
            const olderHistory = initial.filter(p => p.timestamp < earliestExisting);
            return olderHistory.length > 0 ? [...olderHistory, ...prev] : prev;
        });
        if (initLastTs > lastTsRef.current) {
            lastTsRef.current = initLastTs;
        }
    }, [isLive, initial, resetKey]);

    // 追加实时点
    useEffect(() => {
        if (!isLive || !livePoint) return;
        const ts = livePoint.timestamp;
        if (!Number.isFinite(ts) || ts <= 0) return;
        if (ts <= lastTsRef.current) return;
        lastTsRef.current = ts;

        setBuffer(prev => {
            const next = prev.length === 0 ? [livePoint] : [...prev, livePoint];
            const cutoff = ts - windowMs;
            let drop = 0;
            while (drop < next.length && next[drop].timestamp < cutoff) drop++;
            return drop > 0 ? next.slice(drop) : next;
        });
    }, [livePoint, isLive, windowMs]);

    return isLive ? buffer : initial;
}
