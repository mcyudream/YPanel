package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/errs"
)

// MiddlewareListItem 的 key 包含来源，数据库 ID 和中间件 ID 不共用命名空间。
type MiddlewareListItem struct {
	Key             string    `json:"key"`
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Label           string    `json:"label"`
	Source          string    `json:"source"` // middleware / database / discovered
	Origin          string    `json:"origin"`
	NodeID          string    `json:"nodeId"` // 外部连接为空，与当前选择的节点无关
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	User            string    `json:"user"`
	Version         string    `json:"version"`
	ContainerName   string    `json:"containerName"`
	ComposeProject  string    `json:"composeProject"`
	ComposeDir      string    `json:"composeDir,omitempty"`
	Image           string    `json:"image"`
	Networks        []string  `json:"networks"`
	State           string    `json:"state"`
	ManagedBy       string    `json:"managedBy"` // middleware / database
	Adoptable       bool      `json:"adoptable"`
	Reusable        bool      `json:"reusable"`
	MinReuseVersion string    `json:"minReuseVersion"`
	OwnerID         uint      `json:"ownerId"`
	Remark          string    `json:"remark"`
	CreatedAt       time.Time `json:"createdAt"`
}

type MiddlewareList struct {
	Items          []MiddlewareListItem `json:"items"`
	Total          int                  `json:"total"`
	DiscoveryError string               `json:"discoveryError,omitempty"`
}

// middlewareContainer 是 Docker 元数据投影；绝不读取 Config.Env。
type middlewareContainer struct {
	ContainerName  string `json:"containerName"`
	Image          string `json:"image"`
	State          string `json:"state"`
	Networks       string `json:"networks"`
	Ports          string `json:"ports"`
	ComposeProject string `json:"composeProject"`
	ComposeDir     string `json:"composeDir"`
}

func middlewareNodeAllowed(ctx context.Context, nodeID string) bool {
	caller, ok := rbac.CallerFrom(ctx)
	if !ok || caller.AllNodes {
		return true
	}
	_, ok = caller.NodeSet[normalizeNodeID(nodeID)]
	return ok
}

func middlewareOwnerAllowed(ctx context.Context, ownerID uint) bool {
	caller, ok := rbac.CallerFrom(ctx)
	return !ok || !caller.Assigned() || ownerID == 0 || ownerID == caller.UserID
}

func middlewareOwnerID(ctx context.Context) uint {
	if caller, ok := rbac.CallerFrom(ctx); ok && caller.Assigned() {
		return caller.UserID
	}
	return 0
}

func middlewareKey(nodeID, container string) string { return normalizeNodeID(nodeID) + "/" + container }

func (s *MiddlewareService) scanContainers(ctx context.Context, nodeID string) ([]middlewareContainer, error) {
	if !middlewareNodeAllowed(ctx, nodeID) {
		return nil, errs.New(errs.CodeForbidden, "error.nodeForbidden", "无该节点的操作权限")
	}
	ac, err := s.clientFor(nodeID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	const cmd = `docker ps -a --no-trunc --format '{"containerName":{{json .Names}},"image":{{json .Image}},"state":{{json .State}},"networks":{{json .Networks}},"ports":{{json .Ports}},"composeProject":{{json (.Label "com.docker.compose.project")}},"composeDir":{{json (.Label "com.docker.compose.project.working_dir")}}}'`
	res, err := execAgentCmd(ac, ctx, cmd, 30)
	if err != nil {
		return nil, err
	}
	out := make([]middlewareContainer, 0)
	for _, line := range strings.Split(strings.TrimSpace(res.Output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item middlewareContainer
		if err := json.Unmarshal([]byte(line), &item); err != nil || item.ContainerName == "" || item.Image == "" {
			return nil, errs.Wrap(errs.ErrBadRequest, "容器发现结果格式异常，请检查节点 Docker 状态")
		}
		out = append(out, item)
	}
	return out, nil
}

func middlewareItem(name, typ, source string, id uint) MiddlewareListItem {
	d, _ := middlewareDiscoveryDescriptor(typ)
	return MiddlewareListItem{Key: fmt.Sprintf("%s/%d", source, id), ID: id, Name: name, Type: typ,
		Label: d.Label, Source: source, ManagedBy: source, State: "external", Networks: make([]string, 0),
		MinReuseVersion: d.MinReuseVersion, Reusable: true}
}

func mergeMiddlewareContainer(item *MiddlewareListItem, c middlewareContainer) {
	item.ContainerName, item.Image, item.State = c.ContainerName, c.Image, c.State
	item.ComposeProject = c.ComposeProject
	item.ComposeDir = c.ComposeDir
	item.Networks = make([]string, 0)
	for _, n := range strings.Split(c.Networks, ",") {
		if n = strings.TrimSpace(n); n != "" {
			item.Networks = append(item.Networks, n)
		}
	}
	sort.Strings(item.Networks)
	if item.Version == "" {
		item.Version = versionFromImage(c.Image)
	}
	item.Reusable = middlewareVersionOK(item.Version, item.MinReuseVersion)
}

var middlewarePublishedPortRE = regexp.MustCompile(`(\d+)->(\d+)/tcp`)

func middlewareHasPort(c middlewareContainer, port int) bool {
	for _, m := range middlewarePublishedPortRE.FindAllStringSubmatch(c.Ports, -1) {
		if m[1] == strconv.Itoa(port) {
			return true
		}
	}
	return false
}

func middlewareLoopback(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1" || host == ""
}

func middlewareDatabaseMatches(r model.DatabaseInstance, dbNode, nodeID, typ string, c middlewareContainer) bool {
	if r.Type != typ || dbNode != nodeID {
		return false
	}
	// 外部远端主机不能因为实例名相同就与本机容器关联。
	if r.Origin == "external" && !middlewareLoopback(r.Host) && r.Host != c.ContainerName {
		return false
	}
	if r.ComposeProject != "" && r.ComposeProject != "-" && r.ComposeProject == c.ComposeProject {
		return true
	}
	return nodeID == "local" && (r.Host == c.ContainerName || r.Origin != "external" && r.Name == c.ContainerName || middlewareLoopback(r.Host) && middlewareHasPort(c, r.Port))
}

// List 合并登记表、数据库域引用和当前节点容器。只读发现不会自动纳管或读取密码。
// 节点不可达时保留登记数据，discoveryError 明示发现失败，不能伪装为空列表。
func (s *MiddlewareService) List(ctx context.Context, nodeID string) (*MiddlewareList, error) {
	nodeID = normalizeNodeID(nodeID)
	if !middlewareNodeAllowed(ctx, nodeID) {
		return nil, errs.New(errs.CodeForbidden, "error.nodeForbidden", "无该节点的操作权限")
	}
	node, err := s.nodes.ByID(nodeID)
	if err != nil {
		return nil, err
	}
	nodeID = node.ID
	var registered []model.MiddlewareInstance
	var databases []model.DatabaseInstance
	var installs []model.AppStoreInstall
	if err := s.db.Order("type, name").Find(&registered).Error; err != nil {
		return nil, err
	}
	if err := s.db.Order("type, name").Find(&databases).Error; err != nil {
		return nil, err
	}
	// 关联安装记录只需要身份与属主，不能读取包含明文参数的 ParamsJSON。
	if err := s.db.Select("compose_project", "node_id", "owner_id").Find(&installs).Error; err != nil {
		return nil, err
	}

	out := &MiddlewareList{Items: make([]MiddlewareListItem, 0)}
	managed := map[string]int{}
	hidden := map[string]bool{}
	hiddenProjects := map[string]bool{}
	projectNodes := map[string][]string{}
	for _, in := range installs {
		projectNodes[in.ComposeProject] = append(projectNodes[in.ComposeProject], normalizeNodeID(in.NodeID))
		if !middlewareOwnerAllowed(ctx, in.OwnerID) {
			hiddenProjects[middlewareKey(in.NodeID, in.ComposeProject)] = true
		}
	}
	for _, r := range registered {
		if r.Origin == "container" && normalizeNodeID(r.NodeID) != nodeID {
			continue
		}
		key := middlewareKey(r.NodeID, r.ContainerName)
		if !middlewareOwnerAllowed(ctx, r.OwnerID) {
			if r.ContainerName != "" {
				hidden[key] = true
			}
			continue
		}
		item := middlewareItem(r.Name, r.Type, "middleware", r.ID)
		item.Origin, item.Host, item.Port, item.User = r.Origin, r.Host, r.Port, r.User
		item.Version, item.ContainerName, item.ComposeProject = r.Version, r.ContainerName, r.ComposeProject
		item.OwnerID, item.Remark, item.CreatedAt = r.OwnerID, r.Remark, r.CreatedAt
		item.Reusable = middlewareVersionOK(r.Version, item.MinReuseVersion)
		if r.Origin == "container" {
			item.NodeID, item.State = nodeID, "missing"
			managed[key] = len(out.Items)
		}
		out.Items = append(out.Items, item)
	}
	// DatabaseInstance 尚无节点字段：仅从唯一的商店项目关联推断，否则存量按 local 处理。
	dbNodes := map[uint]string{}
	dbIndices := map[uint]int{}
	for _, r := range databases {
		if _, ok := middlewareDatabaseRegistry[r.Type]; !ok {
			continue
		}
		node := "local"
		if ns := projectNodes[r.ComposeProject]; len(ns) == 1 {
			node = ns[0]
		}
		dbNodes[r.ID] = node
		if r.Origin != "external" && node != nodeID {
			continue
		}
		if !middlewareOwnerAllowed(ctx, r.OwnerID) {
			continue
		}
		item := middlewareItem(r.Name, r.Type, "database", r.ID)
		item.Origin, item.Host, item.Port, item.User = r.Origin, r.Host, r.Port, r.RootUser
		item.ComposeProject, item.OwnerID, item.Remark, item.CreatedAt = r.ComposeProject, r.OwnerID, r.Remark, r.CreatedAt
		if r.Origin != "external" {
			item.NodeID, item.State = node, "missing"
		}
		dbIndices[r.ID] = len(out.Items)
		out.Items = append(out.Items, item)
	}
	containers, err := s.scanContainers(ctx, nodeID)
	if err != nil {
		out.DiscoveryError = "容器发现失败：" + err.Error()
		for i := range out.Items {
			if out.Items[i].State == "missing" {
				out.Items[i].State = "unknown"
			}
		}
	} else {
		// 一个项目可能包含多个同类服务；有歧义时保留独立发现条目，不能覆盖成一个实例。
		dbMatchCounts := map[uint]int{}
		for _, c := range containers {
			typ, _, ok := middlewareTypeByImage(c.Image)
			if !ok {
				continue
			}
			for _, r := range databases {
				if middlewareDatabaseMatches(r, dbNodes[r.ID], nodeID, typ, c) {
					dbMatchCounts[r.ID]++
				}
			}
		}
		for _, c := range containers {
			key := middlewareKey(nodeID, c.ContainerName)
			if hidden[key] || hiddenProjects[middlewareKey(nodeID, c.ComposeProject)] {
				continue
			}
			if i, ok := managed[key]; ok {
				mergeMiddlewareContainer(&out.Items[i], c)
				continue
			}
			typ, d, ok := middlewareTypeByImage(c.Image)
			if !ok {
				continue
			}
			// 同类型 + 同节点 + 稳定身份（项目/容器/宿主端口），只关联唯一记录。
			matches := make([]uint, 0)
			denied := false
			for _, r := range databases {
				if middlewareDatabaseMatches(r, dbNodes[r.ID], nodeID, typ, c) {
					matches = append(matches, r.ID)
					if !middlewareOwnerAllowed(ctx, r.OwnerID) {
						denied = true
					}
				}
			}
			if denied {
				continue
			}
			if len(matches) == 1 && dbMatchCounts[matches[0]] == 1 {
				if i, visible := dbIndices[matches[0]]; visible {
					mergeMiddlewareContainer(&out.Items[i], c)
					out.Items[i].NodeID = nodeID
				}
				continue // 不把不可见的纳管资源重新暴露为“待纳管”
			}
			item := middlewareItem(c.ContainerName, typ, "discovered", 0)
			item.Key, item.NodeID, item.Origin = "discovered/"+key, nodeID, "container"
			item.Host, item.Port, item.Label = c.ContainerName, d.DefaultPort, d.Label
			item.ManagedBy = "database"
			if _, ok := middlewareRegistry[typ]; ok {
				item.ManagedBy, item.Adoptable = "middleware", true
			}
			mergeMiddlewareContainer(&item, c)
			out.Items = append(out.Items, item)
		}
	}
	sort.Slice(out.Items, func(i, j int) bool {
		return out.Items[i].Type+out.Items[i].Name+out.Items[i].Key < out.Items[j].Type+out.Items[j].Name+out.Items[j].Key
	})
	out.Total = len(out.Items)
	return out, nil
}
