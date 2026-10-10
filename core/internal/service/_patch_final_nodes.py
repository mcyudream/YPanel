import io

# ============ 1) aitools_sysproc.go：run_command / run_in_workspace 加 node ============
p = 'aitools_sysproc.go'
s = io.open(p, encoding='utf-8').read()

old = '''				p, err := parseToolArgs[struct {
					Command     string `json:"command"`
					TimeoutSecs int    `json:"timeoutSecs"`
				}](input)'''
new = '''				p, err := parseToolArgs[struct {
					Command     string `json:"command"`
					TimeoutSecs int    `json:"timeoutSecs"`
					Node        string `json:"node"`
				}](input)'''
assert old in s, 'run_command struct'
s = s.replace(old, new, 1)
old = '''				if p.TimeoutSecs < 1 || p.TimeoutSecs > 300 {
					p.TimeoutSecs = 120
				}
				return s.hostExecTimeout(ctx, p.Command, p.TimeoutSecs)'''
new = '''				if p.TimeoutSecs < 1 || p.TimeoutSecs > 300 {
					p.TimeoutSecs = 120
				}
				ctx = withAINode(ctx, p.Node)
				return s.hostExecTimeout(ctx, p.Command, p.TimeoutSecs)'''
assert old in s, 'run_command inject'
s = s.replace(old, new, 1)
old = '''			Parameters: schObj(map[string]any{
				"command": schStr("要执行的 shell 命令"), "timeoutSecs": schInt("超时秒数，默认 120 上限 300"),
			}, "command"),'''
new = '''			Parameters: schObj(map[string]any{
				"command": schStr("要执行的 shell 命令"), "timeoutSecs": schInt("超时秒数，默认 120 上限 300"),
				"node": schStr("目标节点，默认 local"),
			}, "command"),'''
assert old in s, 'run_command schema'
s = s.replace(old, new, 1)

old = '''				p, err := parseToolArgs[struct {
					Command string `json:"command"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Command) == "" {
					return "", fmt.Errorf("命令为空")
				}
				return s.hostExec(ctx, fmt.Sprintf("mkdir -p '%s' && cd '%s' && %s", workspaceDir, workspaceDir, p.Command))'''
new = '''				p, err := parseToolArgs[struct {
					Command string `json:"command"`
					Node    string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(p.Command) == "" {
					return "", fmt.Errorf("命令为空")
				}
				ctx = withAINode(ctx, p.Node)
				return s.hostExec(ctx, fmt.Sprintf("mkdir -p '%s' && cd '%s' && %s", workspaceDir, workspaceDir, p.Command))'''
assert old in s, 'workspace struct'
s = s.replace(old, new, 1)
old = '''			Parameters: schObj(map[string]any{"command": schStr("shell 命令")}, "command"),'''
assert old in s, 'workspace schema'
s = s.replace(old, '''			Parameters: schObj(map[string]any{"command": schStr("shell 命令"), "node": schStr("目标节点，默认 local")}, "command"),''', 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('exec ok')

# ============ 2) aitools_netsec.go：firewall 三工具 WithNode + nat_rule_save 透传 nodeId ============
p = 'aitools_netsec.go'
s = io.open(p, encoding='utf-8').read()

# firewall_status
old = '''			Fn: func(ctx context.Context, _ string) (string, error) {
				out := map[string]any{}
				if st, err := s.fw.Status(ctx); err == nil {'''
new = '''			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				fw, ferr := s.fw.WithNode(p.Node)
				if ferr != nil {
					return "", ferr
				}
				out := map[string]any{}
				if st, err := fw.Status(ctx); err == nil {'''
assert old in s, 'fw status'
s = s.replace(old, new, 1)
old = '''			Parameters: schObj(map[string]any{}),
			Fn: func(ctx context.Context, _ string) (string, error) {
				out := map[string]any{}'''
assert old in s; s = s.replace(old, '''			Parameters: schObj(map[string]any{"node": schStr("目标节点，默认 local")}),
			Fn: func(ctx context.Context, input string) (string, error) {
				out := map[string]any{}''', 1)

# firewall_rule_add / remove：WithNode
old = '''			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Port  string `json:"port"`
					Proto string `json:"proto"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Proto != "tcp" && p.Proto != "udp" {
					p.Proto = "tcp"
				}
				if err := s.fw.Allow(ctx, p.Port, p.Proto); err != nil {'''
new = '''			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Port  string `json:"port"`
					Proto string `json:"proto"`
					Node  string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				if p.Proto != "tcp" && p.Proto != "udp" {
					p.Proto = "tcp"
				}
				fw, nerr := s.fw.WithNode(p.Node)
				if nerr != nil {
					return "", nerr
				}
				if err := fw.Allow(ctx, p.Port, p.Proto); err != nil {'''
assert old in s, 'fw add'
s = s.replace(old, new, 1)

old = '''			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Number int `json:"number"`
				}](input)
				if err != nil {
					return "", err
				}
				if err := s.fw.DeleteRule(ctx, p.Number); err != nil {'''
new = '''			Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Number int    `json:"number"`
					Node   string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				fw, nerr := s.fw.WithNode(p.Node)
				if nerr != nil {
					return "", nerr
				}
				if err := fw.DeleteRule(ctx, p.Number); err != nil {'''
assert old in s, 'fw remove'
s = s.replace(old, new, 1)
for old_s, new_s in [
    ('''			Parameters: schObj(map[string]any{
				"port": schStr("端口或端口段"), "proto": schEnum("协议", "tcp", "udp"),
			}, "port", "proto"),''',
     '''			Parameters: schObj(map[string]any{
				"port": schStr("端口或端口段"), "proto": schEnum("协议", "tcp", "udp"), "node": schStr("目标节点，默认 local"),
			}, "port", "proto", "node"),'''),
    ('''			Parameters: schObj(map[string]any{"number": schInt("规则编号")}, "number"),''',
     '''			Parameters: schObj(map[string]any{"number": schInt("规则编号"), "node": schStr("目标节点，默认 local")}, "number", "node"),'''),
]:
    assert old_s in s, old_s[:50]
    s = s.replace(old_s, new_s, 1)

# nat_rule_save：NodeID 透传
old = '''				rule := &model.NatForwardRule{
					NodeID: "local", Name: p.Name, Protocol: p.Protocol,'''
new = '''				nodeId := p.Node
				if nodeId == "" {
					nodeId = "local"
				}
				rule := &model.NatForwardRule{
					NodeID: nodeId, Name: p.Name, Protocol: p.Protocol,'''
assert old in s, 'nat rule'
s = s.replace(old, new, 1)
old = '''			Parameters: schObj(map[string]any{
				"id": schInt("更新已有规则时传"), "name": schStr("规则名"), "protocol": schEnum("协议", "tcp", "udp"),
				"listenPort": schInt("监听端口"), "targetIp": schStr("目标 IP"), "targetPort": schInt("目标端口"),
				"enabled": schBool("启用"),
			}, "name", "protocol", "listenPort", "targetIp", "targetPort"),'''
new = '''			Parameters: schObj(map[string]any{
				"id": schInt("更新已有规则时传"), "name": schStr("规则名"), "protocol": schEnum("协议", "tcp", "udp"),
				"listenPort": schInt("监听端口"), "targetIp": schStr("目标 IP"), "targetPort": schInt("目标端口"),
				"enabled": schBool("启用"), "node": schStr("目标节点 ID，默认 local"),
			}, "name", "protocol", "listenPort", "targetIp", "targetPort", "node"),'''
assert old in s; s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('netsec ok')

# ============ 3) aitools_store.go：install schema 暴露 nodeId + uninstall 透传 ============
p = 'aitools_store.go'
s = io.open(p, encoding='utf-8').read()
old = '''			Parameters: schObj(map[string]any{
				"sourceId": schInt("应用来源 ID"), "key": schStr("应用 key"),
				"name": schStr("应用实例名（小写字母/数字/中划线，如 app-blog）"),'''
new = '''			Parameters: schObj(map[string]any{
				"sourceId": schInt("应用来源 ID"), "key": schStr("应用 key"),
				"name": schStr("应用实例名（小写字母/数字/中划线，如 app-blog）"),
				"nodeId": schStr("目标节点 ID，默认本机"),'''
assert old in s, 'install schema'
s = s.replace(old, new, 1)

old = '''				out, err := s.store.Uninstall(ctx, p.Project, StoreUninstallOptions{
					PurgeData: p.PurgeData, RemoveImage: p.RemoveImage, CascadeDB: p.CascadeDB,
				})'''
new = '''				out, err := s.store.Uninstall(ctx, p.Project, StoreUninstallOptions{
					NodeID: p.Node, PurgeData: p.PurgeData, RemoveImage: p.RemoveImage, CascadeDB: p.CascadeDB,
				})'''
assert old in s, 'uninstall'
s = s.replace(old, new, 1)
old = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					PurgeData   bool   `json:"purgeData"`
					RemoveImage bool   `json:"removeImage"`
					CascadeDB   bool   `json:"cascadeDB"`
				}](input)'''
if old not in s:
    # 字段顺序可能不同，宽松匹配
    import re
    m = re.search(r'parseToolArgs\[struct \{([^}]*Project string[^}]*)\]\(input\)', s)
    print('uninstall struct block:', m.group(0) if m else '?')
    old = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					PurgeData   bool   `json:"purgeData"`
					RemoveImage bool   `json:"removeImage"`
					CascadeDB   bool   `json:"cascadeDB"`
				}](input)'''
new = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Node    string `json:"node"`
					PurgeData   bool `json:"purgeData"`
					RemoveImage bool `json:"removeImage"`
					CascadeDB   bool `json:"cascadeDB"`
				}](input)'''
assert old in s, 'uninstall struct'
s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('store ok')
