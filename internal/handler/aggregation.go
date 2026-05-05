package handler

import "strings"

// normalizeAggregation 归一化聚合参数：
//   - "" / 未识别值 → "" 表示 band 模式（同时返回 avg + max 两条 series）
//   - "avg" / "max"     → 单条对应窗口聚合
//   - "raw"             → 不做窗口聚合，按 step 取原始样本
func normalizeAggregation(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "avg", "max", "raw":
		return value
	default:
		return ""
	}
}
