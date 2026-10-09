package dto

// 磁盘空间保护（Disk Guard）core↔agent 协议。
// 语义：agent 侧执行"批量停止+策略改写"与"按快照恢复"；触发判断与事件状态由 core 维护。

// GuardStoppedItem 被磁盘保护停掉的容器快照条目（含原重启策略，恢复时还原）。
type GuardStoppedItem struct {
	ID            string `json:"id"` // 容器短 ID（12 位）
	Name          string `json:"name"`
	RestartPolicy string `json:"restartPolicy"` // no/always/unless-stopped/on-failure；"" 视为 no
}

// GuardStopAllReq 批量停止请求。
type GuardStopAllReq struct {
	Exclude []string `json:"exclude"` // 豁免容器名（触发时不停）
}

// GuardStopAllResp 批量停止结果。
type GuardStopAllResp struct {
	Stopped []GuardStoppedItem `json:"stopped"`
	Skipped []string           `json:"skipped"` // 命中豁免名单未停的容器名
	Failed  []string           `json:"failed"`  // "name: 原因"（停止失败，下轮压制重试）
}

// GuardRestoreReq 一键恢复请求（按事件快照）。
type GuardRestoreReq struct {
	Containers []GuardStoppedItem `json:"containers"`
}

// GuardRestoreResp 一键恢复结果。
type GuardRestoreResp struct {
	Started []string `json:"started"` // 已启动/本就在运行的容器名
	Failed  []string `json:"failed"`  // "name: 原因"
}
