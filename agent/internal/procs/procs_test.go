package procs

import (
	"testing"

	"github.com/ypanel/shared/dto"
)

func mkList() []dto.ProcessItem {
	return []dto.ProcessItem{
		{Pid: 3, Name: "nginx", Cpu: 5.0, Mem: 2.0, MemRSS: 300 << 20, User: "www"},
		{Pid: 1, Name: "systemd", Cpu: 0.1, Mem: 0.2, MemRSS: 10 << 20, User: "root"},
		{Pid: 2, Name: "ypanel", Cpu: 12.5, Mem: 8.0, MemRSS: 800 << 20, User: "root"},
	}
}

func TestSortByCPUDefaultDesc(t *testing.T) {
	list := mkList()
	sortBy(list, "", "")
	if list[0].Name != "ypanel" || list[2].Name != "systemd" {
		t.Fatalf("期望按 CPU 降序 ypanel 在首位，得到 %v", list)
	}
}

func TestSortByMemAsc(t *testing.T) {
	list := mkList()
	sortBy(list, "mem", "asc")
	if list[0].Name != "systemd" || list[2].Name != "ypanel" {
		t.Fatalf("期望按内存升序 systemd 在首位，得到 %v", list)
	}
}

func TestSortByRssDesc(t *testing.T) {
	list := mkList()
	sortBy(list, "rss", "desc")
	if list[0].Name != "ypanel" {
		t.Fatalf("期望按 RSS 降序 ypanel 在首位，得到 %v", list)
	}
}

func TestSortByPidAsc(t *testing.T) {
	list := mkList()
	sortBy(list, "pid", "asc")
	if list[0].Pid != 1 || list[2].Pid != 3 {
		t.Fatalf("期望按 PID 升序，得到 %v", list)
	}
}

func TestSortByNameDesc(t *testing.T) {
	list := mkList()
	sortBy(list, "name", "desc")
	if list[0].Name != "ypanel" || list[2].Name != "nginx" {
		t.Fatalf("期望按名称降序 ypanel 在首位，得到 %v", list)
	}
}

func TestSortByInvalidFieldFallsBackToCPU(t *testing.T) {
	list := mkList()
	sortBy(list, "bogus", "desc")
	if list[0].Name != "ypanel" {
		t.Fatalf("非法字段应回退 CPU 排序，得到 %v", list)
	}
}
