// 可复用中间件类型注册表（M68 通用规范核心）。
//
// 新增一种可复用非数据库中间件类型（如 nats、kafka、elasticsearch），只需在 middlewareRegistry
// 注册一项 descriptor，扫描发现、版本校验、连接测试、商店复用注入即全链生效——不改任何调用点。
//
// 字段语义：
//   - ImagePatterns：容器扫描的镜像名前缀匹配（小写；不含 tag 也匹配，"rabbitmq" 命中 rabbitmq:4.x）
//   - DefaultPort：默认服务端口（扫描/手动添加的缺省值）
//   - CredentialEnv：扫描时从容器 env 提取凭据的 [用户键, 密码键]（可空串=无该项）
//   - VersionProbe：容器内版本探测命令（空 = 从镜像 tag 提取语义版本，见 versionFromImage）
//   - MinReuseVersion：面板认可的最低可复用版本基线（semver 逐段比较；低于则下拉禁选）
//   - TestConn：连接测试（TCP/协议层；密码错不在此兜底，以 descriptor 能力为准）
//   - InjectPrefix：商店复用注入的默认 env 键前缀（应用声明未指定时回退）
package service

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ypanel/shared/errs"
)

// MiddlewareDescriptor 中间件类型描述符。
type MiddlewareDescriptor struct {
	Type            string
	Label           string
	ImagePatterns   []string
	DefaultPort     int
	CredentialEnv   [2]string
	VersionProbe    string
	MinReuseVersion string
	InjectPrefix    string
	TestConn        func(ctx context.Context, host string, port int, user, pass string) error
}

// middlewareRegistry 类型描述符注册表（Neo4j/S3 走数据库纳管域，不在此列；需要复用校验时可补注册）。
var middlewareRegistry = map[string]MiddlewareDescriptor{
	"rabbitmq": {
		Type: "rabbitmq", Label: "RabbitMQ",
		ImagePatterns:   []string{"rabbitmq"},
		DefaultPort:     5672,
		CredentialEnv:   [2]string{"RABBITMQ_DEFAULT_USER", "RABBITMQ_DEFAULT_PASS"},
		MinReuseVersion: "3.10",
		InjectPrefix:    "RABBITMQ",
		TestConn:        tcpTestConn,
	},
	"memcached": {
		Type: "memcached", Label: "Memcached",
		ImagePatterns: []string{"memcached"},
		DefaultPort:   11211,
		VersionProbe:  "", // 镜像 tag 推断
		InjectPrefix:  "MEMCACHED",
		TestConn:      tcpTestConn,
	},
}

// middlewareDescriptorOf 按类型取描述符。
func middlewareDescriptorOf(t string) (MiddlewareDescriptor, bool) {
	d, ok := middlewareRegistry[strings.ToLower(strings.TrimSpace(t))]
	return d, ok
}

// middlewareTypeByImage 镜像名 → 类型（扫描发现；取最长前缀命中）。
func middlewareTypeByImage(image string) (string, MiddlewareDescriptor, bool) {
	img := strings.ToLower(image)
	if i := strings.Index(img, ":"); i > 0 {
		img = img[:i]
	}
	best := ""
	var bestDesc MiddlewareDescriptor
	for _, d := range middlewareRegistry {
		for _, p := range d.ImagePatterns {
			if img == p || strings.HasPrefix(img, p+"/") || strings.HasPrefix(img, p) {
				if len(p) > len(best) {
					best = p
					bestDesc = d
				}
			}
		}
	}
	if best == "" {
		return "", MiddlewareDescriptor{}, false
	}
	return bestDesc.Type, bestDesc, true
}

// tcpTestConn 通用 TCP 连通测试。
func tcpTestConn(_ context.Context, host string, port int, _, _ string) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return errs.Wrap(errs.ErrAgentUnreach, "TCP 连接失败 "+addr+": "+err.Error())
	}
	_ = conn.Close()
	return nil
}

// versionFromImage 从镜像 tag 提取语义版本（"rabbitmq:4.3.2-management-alpine" → "4.3.2"；
// "neo4j:5.26-community" → "5.26"；无版本 tag（latest 等）返回空）。
func versionFromImage(image string) string {
	tag := image
	if i := strings.LastIndex(image, ":"); i > 0 && !strings.Contains(image[i:], "/") {
		tag = image[i+1:]
	} else if i < 0 {
		return "" // 无 tag（隐式 latest）
	}
	m := semverInTagRe.FindString(tag)
	return m
}

var semverInTagRe = regexp.MustCompile(`\d+(\.\d+){1,3}`)

// CompareSemver 语义化版本逐段比较：a<b 返回 -1，a>b 返回 1，无法解析或相等返回 0。
// 宽容口径："5.26" vs "5.26.7" 短段补 0 比较；非数字段（rc1 等）忽略后续。
func CompareSemver(a, b string) int {
	as := semverSegments(a)
	bs := semverSegments(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func semverSegments(v string) []int {
	m := semverInTagRe.FindString(strings.TrimSpace(v))
	if m == "" {
		return nil
	}
	parts := strings.Split(m, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}

// middlewareVersionOK 复用版本校验：声明基线 > 实例版本 → false（禁选）。
// 任一版本未知（空/不可解析）→ true（跳过校验，不阻塞复用）。
func middlewareVersionOK(instanceVersion, minVersion string) bool {
	if strings.TrimSpace(instanceVersion) == "" || strings.TrimSpace(minVersion) == "" {
		return true
	}
	return CompareSemver(instanceVersion, minVersion) >= 0
}

// middlewarePortOf 实例端口兜底：未设置时取 descriptor 默认。
func middlewarePortOf(t string, port int) int {
	if port > 0 {
		return port
	}
	if d, ok := middlewareDescriptorOf(t); ok {
		return d.DefaultPort
	}
	return port
}

// middlewareVersionProbe 容器内版本探测（VersionProbe 非空时经 agent docker exec；失败返回空不阻塞）。
func (s *MiddlewareService) middlewareVersionProbe(ctx context.Context, nodeId, container, probe string) string {
	if strings.TrimSpace(probe) == "" {
		return ""
	}
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return ""
	}
	res, err := execAgentCmd(ac, ctx, fmt.Sprintf("docker exec %s sh -c %q 2>/dev/null || true", container, probe), 15)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(firstLine(res.Output))
}
