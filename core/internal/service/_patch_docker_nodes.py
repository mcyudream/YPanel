import io, re

p = 'aitools_docker.go'
s = io.open(p, encoding='utf-8').read()

# 0) Fn 首参具名 ctx（闭包捕获外层会导致 withAINode 语义错误）
s = s.replace('Fn: func(_ context.Context,', 'Fn: func(ctx context.Context,')

NODE_SCHEMA = '"node": schStr("目标节点 ID（list_nodes 可查），默认 local"),'

def add_node_to_struct(block):
    # 有 parse struct 的块：struct 加 Node 字段
    return block.replace('parseToolArgs[struct {\n', 'parseToolArgs[struct {\n\t\t\t\t\tNode string `json:"node"`\n', 1)

def add_node_to_schema(block):
    # schema 首键前插 node
    return block.replace('schObj(map[string]any{\n', 'schObj(map[string]any{\n\t\t\t\t\t' + NODE_SCHEMA + '\n', 1)

def add_inject_and_dx(block):
    # parse 错误处理后插 withAINode + WithNode 获取
    return block.replace(
        '''](input)
				if err != nil {
					return "", err
				}
''',
        '''](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
''', 1)

TOOLS = [
    # [工具名, 有args?, 用dockerX?]
    ('inspect_container', True, False),
    ('container_stats', True, False),
    ('create_container', True, True),
    ('containers_prune', False, True),
    ('list_images', True, False),
    ('pull_image', True, True),
    ('remove_image', True, True),
    ('images_prune', False, True),
    ('list_networks', False, False),
    ('create_network', True, True),
    ('remove_network', True, True),
    ('list_volumes', False, False),
    ('create_volume', True, True),
    ('remove_volume', True, True),
    ('volumes_prune', False, True),
]

# 按工具名切块
blocks = re.split(r'(?=\n\t\t\{\n\t\t\tName: ")', s)
out_blocks = []
for b in blocks:
    matched = None
    for name, has_args, uses_dx in TOOLS:
        if f'Name: "{name}",' in b and 'Module:' in b:
            matched = (name, has_args, uses_dx)
            break
    if not matched:
        out_blocks.append(b)
        continue
    name, has_args, uses_dx = matched
    # Fn 具名
    b = b.replace('Fn: func(_ context.Context,', 'Fn: func(ctx context.Context,')
    if has_args:
        b = add_node_to_struct(b)
        b = add_node_to_schema(b)
        b = add_inject_and_dx(b)
    else:
        # 无参工具：schema 加 node、加 args 解析与 dx
        b = b.replace('Parameters: schObj(map[string]any{}),',
                      'Parameters: schObj(map[string]any{' + NODE_SCHEMA + '}),', 1)
        b = b.replace('Fn: func(ctx context.Context, _ string) (string, error) {',
                      '''Fn: func(ctx context.Context, input string) (string, error) {
				p, err := parseToolArgs[struct {
					Node string `json:"node"`
				}](input)
				if err != nil {
					return "", err
				}
				dx, derr := s.dockerX.WithNode(p.Node)
				if derr != nil {
					return "", derr
				}
''', 1)
    if uses_dx:
        b = b.replace('s.dockerX.', 'dx.')
    out_blocks.append(b)
    print(f'  patched: {name}')

s = ''.join(out_blocks)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('docker file done')
