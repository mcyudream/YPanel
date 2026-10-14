// FtpService FTP 服务管理（M49）：vsftpd 安装/启停/端口/被动范围，agent exec + 回读验证。
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
	"gorm.io/gorm"
)

// FtpService FTP（vsftpd）管理。
type FtpService struct {
	db          *gorm.DB // M69：FTP 账号登记（FtpAccount）
	nodes       *NodeService
	nodeClient  *agentclient.Client // WithNode 绑定的节点客户端（空=本机）
	boundNodeID string              // WithNode 绑定的节点 ID（账号域使用，空=本机）
	tasks       *TaskService        // M65：安装任务化（SetTasks 装配，副本经 WithNode 携带）
}

// NewFtpService 创建。
func NewFtpService(db *gorm.DB, nodes *NodeService) *FtpService {
	return &FtpService{db: db, nodes: nodes}
}

// WithNode 副本绑定节点（沿用现有模式：浅拷贝后换客户端）。
func (s *FtpService) WithNodeDB() *FtpService { return s }

// SetTasks 注入任务服务（main 装配，避免构造签名变更）。
func (s *FtpService) SetTasks(t *TaskService) { s.tasks = t }

// WithNode 返回绑定目标节点的副本（M55 主机域节点化：client 按节点路由，方法签名不变）。
func (s *FtpService) WithNode(nodeId string) (*FtpService, error) {
	node, err := s.nodes.ByID(nodeId)
	if err != nil {
		return nil, err
	}
	cp := *s
	cp.nodeClient = agentclient.New(node.BaseURL, node.Token)
	cp.boundNodeID = nodeId
	return &cp, nil
}

func (s *FtpService) client() (*agentclient.Client, error) {
	if s.nodeClient != nil {
		return s.nodeClient, nil
	}
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *FtpService) exec(ctx context.Context, cmd string, timeout int) (string, error) {
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
	if err != nil {
		return "", err
	}
	return out.Output, nil
}

// portPattern FTP 端口白名单。
var ftpPortPattern = regexp.MustCompile(`^\d{1,5}$`)

// Status FTP 状态（安装/运行/端口/被动范围）。
func (s *FtpService) Status(ctx context.Context) (map[string]any, error) {
	out, err := s.exec(ctx, `command -v vsftpd >/dev/null && echo installed || echo missing; systemctl is-active vsftpd 2>/dev/null || echo inactive; grep -E '^(listen_port|pasv_min_port|pasv_max_port|anonymous_enable)' /etc/vsftpd.conf 2>/dev/null`, 30)
	if err != nil {
		return nil, err
	}
	st := map[string]any{"installed": false, "running": false, "port": 21, "pasvMin": 40000, "pasvMax": 40100, "anonymous": false}
	lines := strings.Split(out, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "installed" {
		st["installed"] = true
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) == "active" {
		st["running"] = true
	}
	for _, line := range lines[2:] {
		kv := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "listen_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["port"] = n
		case "pasv_min_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["pasvMin"] = n
		case "pasv_max_port":
			var n int
			fmt.Sscanf(kv[1], "%d", &n)
			st["pasvMax"] = n
		case "anonymous_enable":
			st["anonymous"] = kv[1] == "YES"
		}
	}
	return st, nil
}

// ftpInstallCmd vsftpd 安装脚本（apt/dnf/yum 多通道；apt 先刷新索引——新机 lists 为空时直接
// install 会报 Unable to locate package；装后以 command -v 回读验证，不依赖管道
// 退出码（`| tail` 会吞掉 apt 的失败退出码）。
const ftpInstallCmd = `export DEBIAN_FRONTEND=noninteractive; ` +
	`if command -v apt-get >/dev/null 2>&1; then apt-get update -qq && apt-get install -y -qq vsftpd; ` +
	`elif command -v dnf >/dev/null 2>&1; then dnf install -y vsftpd; ` +
	`elif command -v yum >/dev/null 2>&1; then yum install -y vsftpd; ` +
	`else echo FTP_NO_PKG_MGR; fi; ` +
	`if command -v vsftpd >/dev/null 2>&1; then systemctl enable --now vsftpd && echo FTP_INSTALL_OK; ` +
	`else echo FTP_INSTALL_MISSING; fi`

// ftpInstallFailMsg 安装失败信息（按标记分类）。
func ftpInstallFailMsg(out string) error {
	msg := "安装失败: " + lastLine(tail(out, 300))
	switch {
	case strings.Contains(out, "FTP_NO_PKG_MGR"):
		msg = "安装失败：未识别到 apt/dnf/yum 包管理器，暂不支持该发行版"
	case strings.Contains(out, "FTP_INSTALL_MISSING"):
		msg = "安装失败（软件源不可用或安装未成功）: " + lastLine(tail(out, 300))
	}
	return errs.Wrapc(errs.CodeFileOpFailed, msg)
}

// InstallTask 安装 vsftpd 任务版（返回任务 ID；apt/yum 输出逐行实时进任务日志）。
// nodeId 仅用于任务标题/防重标识（调用方已 WithNode 绑定执行节点）。
func (s *FtpService) InstallTask(ctx context.Context, nodeId string) (map[string]any, error) {
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	if nodeId == "" {
		nodeId = "local"
	}
	if s.tasks.HasRunning([]string{TaskFtpInstall}, nodeId) {
		return nil, errs.Wrap(errs.ErrBadRequest, "该节点已有 FTP 安装任务运行中，请稍后再试")
	}
	task, err := s.tasks.StartTask(TaskFtpInstall, "安装 FTP 服务（node "+nodeId+"）", nodeId, 12*time.Minute,
		func(ctx context.Context, logf TaskLogf) error {
			ac, err := s.client()
			if err != nil {
				return err
			}
			logf("info", "开始安装 vsftpd（apt/dnf/yum 自动适配）…")
			out, code, err := execStreamOrRun(ctx, ac, ftpInstallCmd, 600, logf)
			if err != nil {
				return err
			}
			if code != 0 || !strings.Contains(out, "FTP_INSTALL_OK") {
				return ftpInstallFailMsg(out)
			}
			logf("info", "安装完成，回读服务状态…")
			st, err := s.Status(ctx)
			if err != nil {
				return err
			}
			if running, ok := st["running"].(bool); ok && running {
				logf("info", "vsftpd 服务运行中")
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// Power 启动/停止/重启。
func (s *FtpService) Power(ctx context.Context, action string) error {
	if action != "start" && action != "stop" && action != "restart" {
		return errs.Wrap(errs.ErrBadRequest, "不支持的动作: "+action)
	}
	_, err := s.exec(ctx, fmt.Sprintf("systemctl %s vsftpd && systemctl is-active vsftpd", action), 30)
	return err
}

// SetPort 设置监听端口 + 被动范围（写 /etc/vsftpd.conf + 重启 + 回读）。
func (s *FtpService) SetPort(ctx context.Context, port, pasvMin, pasvMax int) error {
	if port < 1024 || port > 65535 {
		return errs.Wrap(errs.ErrBadRequest, "监听端口需在 1024-65535")
	}
	if pasvMin < 1024 || pasvMax > 65535 || pasvMin >= pasvMax {
		return errs.Wrap(errs.ErrBadRequest, "被动端口范围不合法")
	}
	script := fmt.Sprintf(`set -e
cp /etc/vsftpd.conf /etc/vsftpd.conf.ypbak 2>/dev/null || true
sed -i '/^listen_port/d;/^pasv_min_port/d;/^pasv_max_port/d;/^listen=/d;/^listen_ipv6/d' /etc/vsftpd.conf
printf '\nlisten=YES\nlisten_port=%d\npasv_enable=YES\npasv_min_port=%d\npasv_max_port=%d\n' %d %d %d >> /etc/vsftpd.conf
systemctl restart vsftpd
grep -E '^(listen_port|pasv_min_port|pasv_max_port)' /etc/vsftpd.conf`, port, pasvMin, pasvMax, port, pasvMin, pasvMax)
	out, err := s.exec(ctx, script, 60)
	if err != nil {
		return err
	}
	if !strings.Contains(out, fmt.Sprintf("listen_port=%d", port)) {
		return errs.Wrapc(errs.CodeFileOpFailed, "端口设置回读验证失败")
	}
	return nil
}

// ftpConfPathProbe 探测配置文件路径（Debian /etc/vsftpd.conf 优先，RHEL /etc/vsftpd/vsftpd.conf 兜底）。
const ftpConfPathProbe = `f=/etc/vsftpd.conf; [ -f "$f" ] || f=/etc/vsftpd/vsftpd.conf; if [ -f "$f" ]; then cat "$f"; else echo __NO_CONF__; fi`

// GetConfig 读 vsftpd.conf 原文（M53：配置编辑器）。
func (s *FtpService) GetConfig(ctx context.Context) (string, error) {
	out, err := s.exec(ctx, ftpConfPathProbe, 15)
	if err != nil {
		return "", err
	}
	if strings.Contains(out, "__NO_CONF__") {
		return "", errs.Wrap(errs.ErrBadRequest, "未找到 vsftpd 配置文件（服务未安装？）")
	}
	return out, nil
}

// PutConfig 写回 vsftpd.conf（base64 传递防注入）+ 重启校验，失败自动回滚备份。
func (s *FtpService) PutConfig(ctx context.Context, content string) error {
	if strings.TrimSpace(content) == "" {
		return errs.Wrap(errs.ErrBadRequest, "配置内容不能为空")
	}
	if len(content) > 512*1024 {
		return errs.Wrap(errs.ErrBadRequest, "配置内容过大（>512KB）")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(content))
	script := `set -e
f=/etc/vsftpd.conf; [ -f "$f" ] || f=/etc/vsftpd/vsftpd.conf
[ -f "$f" ] || { echo __NO_CONF__; exit 1; }
cp "$f" /tmp/yp-vsftpd.bak
printf '%s' '` + b64 + `' | base64 -d > "$f"
if systemctl restart vsftpd 2>/tmp/yp-ftp.err; then
  echo FTP_CONF_OK
else
  cp /tmp/yp-vsftpd.bak "$f"
  systemctl restart vsftpd 2>/dev/null || true
  echo FTP_CONF_ROLLBACK
  cat /tmp/yp-ftp.err
  exit 1
fi`
	out, err := s.exec(ctx, script, 60)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "FTP_CONF_OK") {
		return errs.Wrapc(errs.CodeFileOpFailed, "重启 vsftpd 失败，已回滚: "+lastLine(tail(out, 300)))
	}
	return nil
}

// ---------- 账号管理（M69：系统用户 + vsftpd 白名单/chroot） ----------

// ftpUserRe 账号名白名单（小写字母开头，3-32 位小写字母/数字/下划线/中划线）。
var ftpUserRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,31}$`)

// ftpDirRe FTP 根目录白名单（/ 开头绝对路径，仅字母数字/_-.，不含 shell 元字符与单引号——命令单引号拼接）。
var ftpDirRe = regexp.MustCompile(`^/[A-Za-z0-9_./-]{1,240}$`)

// ftpPwdRe FTP 密码白名单（8-64 位，字母/数字与 !@#%^*_.=- 子集；不含空格/单引号/分号/反引号/$/&/|/;</>）。
var ftpPwdRe = regexp.MustCompile(`^[A-Za-z0-9!@#%^*_.=-]{8,64}$`)

// ftpVsftpdBase 账号模式基线（首个账号创建时幂等写入 vsftpd.conf）。
// userlist_deny=NO：user_list 为白名单——仅面板账号可走 FTP（阻断其余系统用户）；
// chroot_local_user：登录后锁 home；allow_writeable_chroot：vsftpd 要求 home 可写时显式放行。
const ftpVsftpdBase = `local_enable=YES
write_enable=YES
chroot_local_user=YES
allow_writeable_chroot=YES
userlist_enable=YES
userlist_deny=NO
userlist_file=/etc/vsftpd/user_list
user_config_dir=/etc/vsftpd/user_config
`

// ensureVsftpdUserSetup 幂等写入账号模式基线（存在改值、缺失追加）并 reload vsftpd。
func (s *FtpService) ensureVsftpdUserSetup(ctx context.Context) error {
	script := `set -e
C=/etc/vsftpd.conf
touch "$C"
setkv() {
  if grep -qE "^$1=" "$C"; then sed -i "s|^$1=.*|$1=$2|" "$C"; else echo "$1=$2" >> "$C"; fi
}
setkv local_enable YES
setkv write_enable YES
setkv chroot_local_user YES
setkv allow_writeable_chroot YES
setkv userlist_enable YES
setkv userlist_deny NO
setkv userlist_file /etc/vsftpd/user_list
setkv user_config_dir /etc/vsftpd/user_config
touch /etc/vsftpd/user_list
mkdir -p /etc/vsftpd/user_config
chmod 644 /etc/vsftpd/user_list
systemctl reload vsftpd 2>/dev/null || systemctl restart vsftpd
echo FTP_SETUP_OK
`
	out, err := s.exec(ctx, script, 30)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "FTP_SETUP_OK") {
		return errs.Wrapc(errs.CodeFileOpFailed, "vsftpd 账号模式基线写入失败: "+lastLine(tail(out, 300)))
	}
	return nil
}

// SiteDirs 站点目录清单（FTP 根目录「关联站点」下拉数据源）。
// 目录规则与站点 nginx root 一致（/var/www/sites/<name>[+RunDir]）。
func (s *FtpService) SiteDirs(ctx context.Context) ([]map[string]any, error) {
	var sites []model.Site
	if err := s.db.Where("node_id = ? AND type = ?", s.boundNodeID, "static").Order("name ASC").Find(&sites).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(sites))
	for _, st := range sites {
		dir := "/var/www/sites/" + st.Name
		if d := strings.TrimSuffix(st.RunDir, "/"); d != "" && d != "/" {
			dir += d
		}
		out = append(out, map[string]any{
			"name": st.Name, "domain": st.Domain, "dir": dir,
		})
	}
	return out, nil
}

// ListAccounts 节点账号清单（面板库为准，逐条回读系统用户存在性，防面板/系统漂移）。
func (s *FtpService) ListAccounts(ctx context.Context) ([]map[string]any, error) {
	var rows []model.FtpAccount
	if err := s.db.Where("node_id = ?", s.boundNodeID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		exists := false
		if o, err := s.exec(ctx, fmt.Sprintf("id -u '%s' >/dev/null 2>&1 && echo yes || echo no", r.Name), 15); err == nil {
			exists = strings.TrimSpace(lastLine(o)) == "yes"
		}
		out = append(out, map[string]any{
			"id": r.ID, "nodeId": r.NodeID, "name": r.Name, "directory": r.Directory,
			"readOnly": r.ReadOnly, "enabled": r.Enabled, "remark": r.Remark,
			"systemUserExists": exists, "createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// ftpAcctValidate 创建/编辑入参白名单校验（目录/密码均进单引号拼接，白名单排除注入字符）。
func ftpAcctValidate(name, directory, password string) error {
	if !ftpUserRe.MatchString(name) {
		return errs.Wrap(errs.ErrBadRequest, "账号名不合法（小写字母开头，3-32 位小写字母/数字/下划线/中划线）: "+name)
	}
	if !ftpDirRe.MatchString(directory) {
		return errs.Wrap(errs.ErrBadRequest, "根目录不合法（/ 开头绝对路径，仅字母数字/_-.，不含空格与单引号）: "+directory)
	}
	if password != "" && !ftpPwdRe.MatchString(password) {
		return errs.Wrap(errs.ErrBadRequest, "密码不合法（8-64 位，字母/数字与 !@#%^*_.=- 子集；不含空格/单引号/分号/反引号/$/&/|/;</>）")
	}
	return nil
}

// CreateAccount 创建 FTP 账号：系统用户（nologin）+ chpasswd + userlist 白名单 + per-user 只读配置。
// 首个账号触发 vsftpd 基线写入；同名系统用户已存在（非面板创建）明确拒绝。
func (s *FtpService) CreateAccount(ctx context.Context, name, directory, password, remark string, readOnly bool) (*model.FtpAccount, error) {
	if err := ftpAcctValidate(name, directory, password); err != nil {
		return nil, err
	}
	var count int64
	_ = s.db.Model(&model.FtpAccount{}).Where("name = ? AND node_id = ?", name, s.boundNodeID).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "账号已存在: "+name)
	}
	if password == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "创建时密码必填")
	}
	if err := s.ensureVsftpdUserSetup(ctx); err != nil {
		return nil, err
	}
	script := fmt.Sprintf(`set -e
if id -u '%s' >/dev/null 2>&1; then echo FTP_USER_EXISTS; exit 70; fi
useradd -m -d '%s' -s /usr/sbin/nologin '%s'
echo '%s:%s' | chpasswd
chown '%s':'%s' '%s'
grep -qxF '%s' /etc/vsftpd/user_list || echo '%s' >> /etc/vsftpd/user_list
echo FTP_ACCT_OK
`, name, directory, name, name, password, name, name, directory, name, name)
	out, err := s.exec(ctx, script, 30)
	if err != nil {
		return nil, err
	}
	if strings.Contains(out, "FTP_USER_EXISTS") {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "系统已存在同名用户（非面板创建），拒绝接管: "+name)
	}
	if !strings.Contains(out, "FTP_ACCT_OK") {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, "创建账号失败: "+lastLine(tail(out, 300)))
	}
	if readOnly {
		if _, err := s.exec(ctx, fmt.Sprintf("mkdir -p /etc/vsftpd/user_config && echo 'write_enable=NO' > '/etc/vsftpd/user_config/%s'", name), 15); err != nil {
			return nil, err
		}
	}
	if _, err := s.exec(ctx, "systemctl reload vsftpd 2>/dev/null || systemctl restart vsftpd", 30); err != nil {
		return nil, err
	}
	row := &model.FtpAccount{NodeID: s.boundNodeID, Name: name, Directory: directory, ReadOnly: readOnly, Enabled: true, Remark: remark}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateAccount 编辑：改密/根目录/只读/启停/备注（指针 nil = 不改）。目录变更同步 home 并 chown（数据不迁移）。
func (s *FtpService) UpdateAccount(ctx context.Context, id uint, directory, password, remark *string, readOnly, enabled *bool) (*model.FtpAccount, error) {
	var row model.FtpAccount
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	name := row.Name
	if directory != nil && *directory != row.Directory {
		if !ftpDirRe.MatchString(*directory) {
			return nil, errs.Wrap(errs.ErrBadRequest, "根目录不合法: "+*directory)
		}
		if _, err := s.exec(ctx, fmt.Sprintf("mkdir -p '%s' && usermod -d '%s' '%s' && chown '%s':'%s' '%s'", *directory, *directory, name, name, name, *directory), 20); err != nil {
			return nil, err
		}
		row.Directory = *directory
	}
	if password != nil && *password != "" {
		if !ftpPwdRe.MatchString(*password) {
			return nil, errs.Wrap(errs.ErrBadRequest, "密码不合法")
		}
		if _, err := s.exec(ctx, fmt.Sprintf("echo '%s:%s' | chpasswd", name, *password), 15); err != nil {
			return nil, err
		}
	}
	if readOnly != nil && *readOnly != row.ReadOnly {
		uc := fmt.Sprintf("/etc/vsftpd/user_config/%s", name)
		var err error
		if *readOnly {
			_, err = s.exec(ctx, fmt.Sprintf("mkdir -p /etc/vsftpd/user_config && echo 'write_enable=NO' > '%s'", uc), 15)
		} else {
			_, err = s.exec(ctx, fmt.Sprintf("rm -f '%s'", uc), 15)
		}
		if err != nil {
			return nil, err
		}
		row.ReadOnly = *readOnly
	}
	if enabled != nil && *enabled != row.Enabled {
		ul := "/etc/vsftpd/user_list"
		var err error
		if *enabled {
			_, err = s.exec(ctx, fmt.Sprintf("grep -qxF '%s' '%s' || echo '%s' >> '%s'", name, ul, name, ul), 15)
		} else {
			_, err = s.exec(ctx, fmt.Sprintf("sed -i '/^%s$/d' '%s'", name, ul), 15)
		}
		if err != nil {
			return nil, err
		}
		row.Enabled = *enabled
	}
	if remark != nil {
		row.Remark = *remark
	}
	if _, err := s.exec(ctx, "systemctl reload vsftpd 2>/dev/null || systemctl restart vsftpd", 30); err != nil {
		return nil, err
	}
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// DeleteAccount 删除：user_list 摘除 + user_config 清理 + 系统用户删除（purgeDir 级联删目录）。
func (s *FtpService) DeleteAccount(ctx context.Context, id uint, purgeDir bool) error {
	var row model.FtpAccount
	if err := s.db.First(&row, id).Error; err != nil {
		return err
	}
	name := row.Name
	flag := ""
	if purgeDir {
		flag = "-r"
	}
	script := fmt.Sprintf(`sed -i '/^%s$/d' /etc/vsftpd/user_list
rm -f '/etc/vsftpd/user_config/%s'
if id -u '%s' >/dev/null 2>&1; then userdel %s '%s' 2>/dev/null || userdel '%s'; fi
echo FTP_DEL_OK
`, name, name, name, flag, name, name)
	out, err := s.exec(ctx, script, 30)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "FTP_DEL_OK") {
		return errs.Wrapc(errs.CodeFileOpFailed, "系统侧删除失败: "+lastLine(tail(out, 300)))
	}
	if _, err := s.exec(ctx, "systemctl reload vsftpd 2>/dev/null || systemctl restart vsftpd", 30); err != nil {
		return err
	}
	return s.db.Delete(&model.FtpAccount{}, id).Error
}

// TestAccount FTP 登录测试（本机回环 curl ftp:// 列根目录）。密码由前端一次性传入（不落库）。
func (s *FtpService) TestAccount(ctx context.Context, name, password string) (map[string]any, error) {
	if !ftpUserRe.MatchString(name) || !ftpPwdRe.MatchString(password) {
		return nil, errs.Wrap(errs.ErrBadRequest, "账号名/密码格式不合法")
	}
	out, err := s.exec(ctx, fmt.Sprintf("curl -s --connect-timeout 5 -u '%s:%s' ftp://127.0.0.1/ 2>&1 && echo FTP_TEST_OK || echo FTP_TEST_FAIL", name, password), 20)
	if err != nil {
		return nil, err
	}
	ok := strings.Contains(out, "FTP_TEST_OK")
	listing := strings.Replace(strings.Replace(out, "FTP_TEST_OK", "", 1), "FTP_TEST_FAIL", "", 1)
	trimmed := strings.TrimSpace(listing)
	if len(trimmed) > 500 {
		trimmed = trimmed[:500]
	}
	return map[string]any{"ok": ok, "listing": trimmed}, nil
}
