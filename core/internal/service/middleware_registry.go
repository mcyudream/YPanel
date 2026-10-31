// 可复用中间件类型注册表（M68 通用规范核心）。
//
// 新增一种可复用非数据库中间件类型（如 nats、kafka、elasticsearch），只需在 middlewareRegistry
// 注册一项 descriptor，扫描发现、版本校验、连接测试、商店复用注入即全链生效——不改任何调用点。
//
// 字段语义：
//   - ImagePatterns：镜像仓库名/路径的完整后缀匹配（不含 tag；兼容 registry 与代理路径）
//   - DefaultPort：默认服务端口（扫描/手动添加的缺省值）
//   - CredentialEnv：显式纳管时从容器 env 提取凭据的 [用户键, 密码键]（可空串=无该项）
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
	Type              string
	Label             string
	ImagePatterns     []string
	DefaultPort       int
	CredentialEnv     [2]string
	CredentialAliases [2][]string
	VersionProbe      string
	MinReuseVersion   string
	InjectPrefix      string
	TestConn          func(ctx context.Context, host string, port int, user, pass string) error
}

// middlewareRegistry 类型描述符注册表（Neo4j/S3 走数据库纳管域，不在此列；需要复用校验时可补注册）。
var middlewareRegistry = map[string]MiddlewareDescriptor{
	"rabbitmq": {
		Type: "rabbitmq", Label: "RabbitMQ",
		ImagePatterns:     []string{"rabbitmq"},
		DefaultPort:       5672,
		CredentialEnv:     [2]string{"RABBITMQ_DEFAULT_USER", "RABBITMQ_DEFAULT_PASS"},
		CredentialAliases: [2][]string{{"RABBITMQ_USERNAME"}, {"RABBITMQ_PASSWORD"}},
		MinReuseVersion:   "3.10",
		InjectPrefix:      "RABBITMQ",
		TestConn:          tcpTestConn,
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

// 数据库域只参与发现与关联，登记/凭据/操作仍由 DatabaseService 管理。
var middlewareDatabaseRegistry = map[string]MiddlewareDescriptor{
	"s3":       {Type: "s3", Label: "对象存储 (S3)", ImagePatterns: []string{"minio/minio", "rustfs/rustfs"}, DefaultPort: 9000},
	"neo4j":    {Type: "neo4j", Label: "Neo4j", ImagePatterns: []string{"neo4j"}, DefaultPort: 7687},
	"redis":    {Type: "redis", Label: "Redis", ImagePatterns: []string{"redis", "redis-stack", "redis-stack-server"}, DefaultPort: 6379},
	"mongo":    {Type: "mongo", Label: "MongoDB", ImagePatterns: []string{"mongo", "mongodb"}, DefaultPort: 27017},
	"mysql":    {Type: "mysql", Label: "MySQL", ImagePatterns: []string{"mysql", "mariadb"}, DefaultPort: 3306},
	"postgres": {Type: "postgres", Label: "PostgreSQL", ImagePatterns: []string{"postgres", "postgresql"}, DefaultPort: 5432},
}

func middlewareDiscoveryDescriptor(t string) (MiddlewareDescriptor, bool) {
	if d, ok := middlewareDescriptorOf(t); ok {
		return d, true
	}
	d, ok := middlewareDatabaseRegistry[t]
	return d, ok
}

// middlewareDescriptorOf 按类型取描述符。
func middlewareDescriptorOf(t string) (MiddlewareDescriptor, bool) {
	d, ok := middlewareRegistry[strings.ToLower(strings.TrimSpace(t))]
	return d, ok
}

// middlewareTypeByImage 先去 digest/tag，再按完整路径段匹配；registry 端口不作为 tag。
func middlewareTypeByImage(image string) (string, MiddlewareDescriptor, bool) {
	img, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(image)), "@")
	if i := strings.LastIndex(img, ":"); i > strings.LastIndex(img, "/") {
		img = img[:i]
	}
	best := ""
	var bestDesc MiddlewareDescriptor
	match := func(d MiddlewareDescriptor) {
		for _, p := range d.ImagePatterns {
			if img == p || strings.HasSuffix(img, "/"+p) {
				if len(p) > len(best) {
					best = p
					bestDesc = d
				}
			}
		}
	}
	for _, d := range middlewareRegistry {
		match(d)
	}
	for _, d := range middlewareDatabaseRegistry {
		match(d)
	}
	if best == "" {
		return "", MiddlewareDescriptor{}, false
	}
	return bestDesc.Type, bestDesc, true
}

// tcpTestConn 通用 TCP 连通测试。
func tcpTestConn(ctx context.Context, host string, port int, _, _ string) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return errs.Wrap(errs.ErrAgentUnreach, "TCP 连接失败 "+addr+": "+err.Error())
	}
	_ = conn.Close()
	return nil
}

// versionFromImage 从镜像 tag 提取语义版本（"rabbitmq:4.3.2-management-alpine" → "4.3.2"；
// "neo4j:5.26-community" → "5.26"；无版本 tag（latest 等）返回空）。
func versionFromImage(image string) string {
	image, _, _ = strings.Cut(image, "@")
	i := strings.LastIndex(image, ":")
	if i <= strings.LastIndex(image, "/") {
		return ""
	}
	m := semverInTagRe.FindString(image[i+1:])
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

// middlewareVersionProbe 只在显式操作时执行，失败向上传递，避免把命令报错当版本。
func (s *MiddlewareService) middlewareVersionProbe(ctx context.Context, nodeId, container, probe string) (string, error) {
	ac, err := s.clientFor(nodeId)
	if err != nil {
		return "", err
	}
	cmd := "docker inspect " + shellQuote(container) + " --format '{{.Config.Image}}'"
	if strings.TrimSpace(probe) != "" {
		cmd = fmt.Sprintf("docker exec %s sh -c %s", shellQuote(container), shellQuote(probe))
	}
	res, err := execAgentCmd(ac, ctx, cmd, 15)
	if err != nil {
		return "", err
	}
	if probe == "" {
		return versionFromImage(strings.TrimSpace(res.Output)), nil
	}
	return strings.TrimSpace(firstLine(res.Output)), nil
}
