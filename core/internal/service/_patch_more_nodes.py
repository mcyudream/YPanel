import io, re

NODE_SCHEMA = '"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),'
NODE_SCHEMA2 = '"node": schStr("目标节点，默认 local"),'

def patch_tools(p, tools):
    """tools: [(工具名, 是否已有args_struct)]"""
    s = io.open(p, encoding='utf-8').read()
    s = s.replace('Fn: func(_ context.Context,', 'Fn: func(ctx context.Context,')
    blocks = re.split(r'(?=\n\t\t\{\n\t\t\tName: ")', s)
    out = []
    for b in blocks:
        name = None
        for tn, _ in tools:
            if f'Name: "{tn}",' in b and 'Module:' in b:
                name = tn
                break
        if not name:
            out.append(b)
            continue
        b = b.replace('Fn: func(_ context.Context,', 'Fn: func(ctx context.Context,')
        # struct 加 Node（有 parse 匿名 struct 的）
        b2 = re.sub(r'(parseToolArgs\[struct \{\n)', r'\1\t\t\t\t\tNode string `json:"node"`\n', b, count=1)
        if b2 != b:
            b = b2
        # schema 首键前插 node（识别 schObj(map[string]any{ 首次）
        b = b.replace('schObj(map[string]any{\n', 'schObj(map[string]any{\n\t\t\t\t' + NODE_SCHEMA + '\n', 1)
        # 注入行：parse 错误处理后
        b = b.replace(
            '''](input)
				if err != nil {
					return "", err
				}
''',
            '''](input)
				if err != nil {
					return "", err
				}
				ctx = withAINode(ctx, p.Node)
''', 1)
        out.append(b)
        print(f'  patched: {name}')
    s = ''.join(out)
    io.open(p, 'w', encoding='utf-8', newline='').write(s)

# compose 8
patch_tools('aitools_compose.go', [
    ('list_compose_projects', 1), ('compose_logs', 1), ('get_compose_config', 1),
    ('save_compose_config', 1), ('compose_up', 1), ('compose_service_action', 1),
    ('compose_down', 1), ('delete_compose_project', 1),
])

# files 11
patch_tools('aitools_files.go', [
    ('list_files', 1), ('search_files', 1), ('read_file', 1), ('write_file', 1),
    ('make_dir', 1), ('rename_path', 1), ('copy_path', 1), ('compress_files', 1),
    ('decompress_file', 1), ('change_perms', 1), ('delete_paths', 1),
])

# store 3（app_action/read_app_env/save_app_env——struct 已有字段，仅加 Node+注入；store 服务方法带 nodeId）
s = io.open('aitools_store.go', encoding='utf-8').read()
# app_action
old = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Action  string `json:"action"`
				}](input)'''
new = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Action  string `json:"action"`
					Node    string `json:"node"`
				}](input)'''
assert old in s, 'app_action struct'
s = s.replace(old, new, 1)
old = 'if err := s.store.InstalledAction(ctx, p.Project, p.Action, ""); err != nil {'
new = 'if err := s.store.InstalledAction(ctx, p.Project, p.Action, p.Node); err != nil {'
assert old in s; s = s.replace(old, new, 1)
# read_app_env
old = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.InstallEnv(ctx, p.Project)'''
new = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Node    string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				out, err := s.store.InstallEnv(ctx, p.Project, p.Node)'''
assert old in s, 'read_app_env'
s = s.replace(old, new, 1)
# save_app_env
old = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Content string `json:"content"`
				}](input)'''
new = '''				p, err := parseToolArgs[struct {
					Project string `json:"project"`
					Content string `json:"content"`
					Node    string `json:"node"`
				}](input)'''
assert old in s, 'save struct'
s = s.replace(old, new, 1)
old = 'if err := s.store.SaveInstallEnv(ctx, p.Project, p.Content); err != nil {'
new = 'if err := s.store.SaveInstallEnv(ctx, p.Project, p.Node, p.Content); err != nil {'
assert old in s; s = s.replace(old, new, 1)
io.open('aitools_store.go', 'w', encoding='utf-8', newline='').write(s)
print('  store ok')
