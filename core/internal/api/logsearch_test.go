package api

import (
	"testing"
	"time"

	"github.com/ypanel/shared/dto"
)

func mkResp(tsOffsetsSec []int64, container string, truncated bool) *dto.LogsSearchResp {
	base := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	r := &dto.LogsSearchResp{Items: make([]dto.LogSearchItem, 0, len(tsOffsetsSec)), Truncated: truncated}
	for _, off := range tsOffsetsSec {
		r.Items = append(r.Items, dto.LogSearchItem{
			Ts:        base.Add(time.Duration(off) * time.Second),
			Container: container,
			ID:        "abc123",
			Line:      container,
		})
	}
	return r
}

func TestMergeLogSearchSortAndCap(t *testing.T) {
	results := []nodeSearchResult{
		{Name: "b", Resp: mkResp([]int64{30, 10}, "b-node", false)},
		{Name: "a", Resp: mkResp([]int64{20, 0}, "a-node", true)},
	}
	merged := mergeLogSearch(results, 3)
	if !merged.Truncated {
		t.Fatal("任一节点截断应透出 truncated")
	}
	// 4 条（ts=0a,10b,20a,30b）→ 总量 3 保最新（10b,20a,30b）
	if len(merged.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(merged.Items))
	}
	gotOrder := []string{merged.Items[0].Container, merged.Items[1].Container, merged.Items[2].Container}
	wantOrder := []string{"b-node", "a-node", "b-node"}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("归并顺序 = %v, want %v", gotOrder, wantOrder)
		}
	}
}

func TestMergeLogSearchErrorsAndEmpty(t *testing.T) {
	errResp := &dto.LogsSearchResp{
		Items:  make([]dto.LogSearchItem, 0),
		Errors: []string{"agent HTTP 404"},
	}
	results := []nodeSearchResult{
		{Name: "web", Resp: errResp},
		{Name: "local", Resp: nil}, // 异常防御：nil 结果跳过
	}
	merged := mergeLogSearch(results, 0)
	if len(merged.Items) != 0 || merged.Items == nil {
		t.Fatalf("空结果应返回空切片: %#v", merged.Items)
	}
	if len(merged.Errors) != 1 || merged.Errors[0] != "agent HTTP 404" {
		t.Fatalf("节点错误未透出: %v", merged.Errors)
	}
	if merged.Scanned != 0 || merged.Truncated {
		t.Fatal("空结果字段应为零值")
	}
}

func TestMergeLogSearchScannedAndTotalCap(t *testing.T) {
	a := mkResp([]int64{1, 2, 3}, "a", false)
	a.Scanned = 100000
	b := mkResp([]int64{4}, "b", false)
	b.Scanned = 5
	merged := mergeLogSearch([]nodeSearchResult{{Name: "a", Resp: a}, {Name: "b", Resp: b}}, 2)
	if merged.Scanned != 100005 {
		t.Fatalf("scanned = %d, want 100005", merged.Scanned)
	}
	// 保最新 2 条：ts=3(a), ts=4(b)
	if len(merged.Items) != 2 || merged.Items[0].Line != "a" || merged.Items[1].Line != "b" {
		t.Fatalf("总量截断保最新失败: %+v", merged.Items)
	}
	if !merged.Truncated {
		t.Fatal("core 侧截断应置 truncated")
	}
}
