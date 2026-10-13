// Neo4j 驱动（M68）：Bolt 直连，社区版单库语义。
// 管理类操作（建库/建号/远程开关）Neo4j 无对应概念，明确拒绝而非静默 no-op；
// 图能力（标签/节点/Cypher/索引约束）经 Neo4jBrowser / Neo4jEditor 能力接口供 service 层断言取用。
package dbdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	neo4jdb "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"

	"github.com/ypanel/shared/errs"
)

type neo4jDriver struct {
	driver neo4jdb.DriverWithContext
	auth   neo4jdb.AuthToken
	target string
}

func newNeo4j(host string, port int, user, password string) (Driver, error) {
	target := fmt.Sprintf("bolt://%s:%d", host, port)
	auth := neo4jdb.BasicAuth(user, password, "")
	drv, err := neo4jdb.NewDriverWithContext(target, auth, func(c *config.Config) {
		c.ConnectionAcquisitionTimeout = 10 * time.Second
	})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return &neo4jDriver{driver: drv, auth: auth, target: target}, nil
}

func (d *neo4jDriver) Close() {
	_ = d.driver.Close(context.Background())
}

// session 短会话助手。
func (d *neo4jDriver) session(ctx context.Context, database, accessMode string) (neo4jdb.SessionWithContext, func()) {
	mode := neo4jdb.AccessModeWrite
	if accessMode == "read" {
		mode = neo4jdb.AccessModeRead
	}
	dbName := database
	if dbName == "" {
		dbName = "neo4j"
	}
	s := d.driver.NewSession(ctx, neo4jdb.SessionConfig{DatabaseName: dbName, AccessMode: mode, BoltLogger: nil})
	return s, func() { _ = s.Close(ctx) }
}

func (d *neo4jDriver) Ping(ctx context.Context) error {
	// 默认用户库执行（system 库只收系统命令，RETURN 会被拒）
	s, done := d.session(ctx, "", "read")
	defer done()
	res, err := s.Run(ctx, "RETURN 1", nil)
	if err != nil {
		return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	_, err = res.Collect(ctx)
	return err
}

// ListDatabases 返回可用图数据库清单（社区版通常仅 neo4j；system 不列）。
func (d *neo4jDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	s, done := d.session(ctx, "system", "read")
	defer done()
	res, err := s.Run(ctx, "SHOW DATABASES", nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DatabaseInfo, 0, len(recs))
	for _, r := range recs {
		name, _ := r.Get("name")
		if strings.EqualFold(toString(name), "system") {
			continue
		}
		out = append(out, DatabaseInfo{Name: toString(name)})
	}
	return out, nil
}

func (d *neo4jDriver) CreateDatabase(ctx context.Context, name, charset string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 社区版为单库，不支持创建图数据库")
}

func (d *neo4jDriver) DropDatabase(ctx context.Context, name string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 社区版为单库，不支持删除图数据库")
}

// ListUsers 数据库账号（SHOW USERS；仅展示，管理走实例自身机制）。
func (d *neo4jDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	s, done := d.session(ctx, "system", "read")
	defer done()
	res, err := s.Run(ctx, "SHOW USERS", nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserInfo, 0, len(recs))
	for _, r := range recs {
		u, _ := r.Get("user")
		out = append(out, UserInfo{Name: toString(u), Extra: "neo4j user"})
	}
	return out, nil
}

func (d *neo4jDriver) CreateUser(ctx context.Context, name, host, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 用户管理请使用实例自身机制，面板不代管")
}

func (d *neo4jDriver) DropUser(ctx context.Context, name, host string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 用户管理请使用实例自身机制，面板不代管")
}

func (d *neo4jDriver) ChangePassword(ctx context.Context, name, host, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 用户管理请使用实例自身机制，面板不代管")
}

func (d *neo4jDriver) EnableRemote(ctx context.Context, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 无 SQL 远程授权语义")
}

func (d *neo4jDriver) DisableRemote(ctx context.Context) error {
	return errs.Wrapc(errs.CodeBadRequest, "Neo4j 无 SQL 远程授权语义")
}

// ---- Neo4jBrowser 能力 ----

func (d *neo4jDriver) ListLabels(ctx context.Context, database string) ([]Neo4jLabelInfo, error) {
	// db.labels 不带计数；节点数按标签聚合一并给出（空图零开销）
	cy := "MATCH (n) RETURN labels(n) AS labels, count(*) AS cnt"
	s, done := d.session(ctx, database, "read")
	defer done()
	res, err := s.Run(ctx, cy, nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	counts := map[string]int64{}
	order := make([]string, 0)
	for _, r := range recs {
		labels, _ := r.Get("labels")
		cnt, _ := r.Get("cnt")
		list, _ := labels.([]any)
		for _, l := range list {
			name := toString(l)
			if name == "" {
				continue
			}
			if _, ok := counts[name]; !ok {
				order = append(order, name)
			}
			counts[name] += toInt64(cnt)
		}
	}
	out := make([]Neo4jLabelInfo, 0, len(order))
	for _, name := range order {
		out = append(out, Neo4jLabelInfo{Name: name, Nodes: counts[name]})
	}
	return out, nil
}

func (d *neo4jDriver) ListRelTypes(ctx context.Context, database string) ([]Neo4jLabelInfo, error) {
	cy := "MATCH ()-[r]->() RETURN type(r) AS type, count(*) AS cnt ORDER BY cnt DESC"
	s, done := d.session(ctx, database, "read")
	defer done()
	res, err := s.Run(ctx, cy, nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Neo4jLabelInfo, 0, len(recs))
	for _, r := range recs {
		t, _ := r.Get("type")
		cnt, _ := r.Get("cnt")
		out = append(out, Neo4jLabelInfo{Name: toString(t), Nodes: toInt64(cnt)})
	}
	return out, nil
}

func (d *neo4jDriver) RunCypher(ctx context.Context, database, cypher string, limit int) (*QueryResult, error) {
	if strings.TrimSpace(cypher) == "" {
		return nil, errs.Wrapc(errs.CodeBadRequest, "Cypher 语句为空")
	}
	if limit <= 0 {
		limit = 200
	}
	s, done := d.session(ctx, database, "write")
	defer done()
	start := time.Now()
	res, err := s.Run(ctx, cypher, nil)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	if len(recs) > limit {
		recs = recs[:limit]
	}
	out := &QueryResult{Columns: make([]string, 0), Rows: make([][]any, 0, len(recs))}
	if len(recs) > 0 {
		out.Columns = recs[0].Keys
	}
	for _, r := range recs {
		row := make([]any, 0, len(r.Values))
		for _, v := range r.Values {
			row = append(row, neo4jToJSON(v))
		}
		out.Rows = append(out.Rows, row)
	}
	out.Elapsed = time.Since(start).Round(time.Millisecond).String()
	// 写语句（CREATE/SET/DELETE 等）无行返回：补计数器摘要提示；读语句空结果保持空展示
	if len(out.Rows) == 0 && len(out.Columns) == 0 {
		summary, _ := res.Consume(ctx)
		if summary.Counters().ContainsUpdates() || summary.Counters().ContainsSystemUpdates() {
			out.Columns = []string{"result"}
			out.Rows = append(out.Rows, []any{fmt.Sprintf("OK (counters: %s)", neo4jCounters(summary))})
		}
	}
	return out, nil
}

func (d *neo4jDriver) BrowseNodes(ctx context.Context, database, label string, skip, limit int) (*Neo4jNodePage, error) {
	if err := ValidateIdent(label); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	s, done := d.session(ctx, database, "read")
	defer done()
	cntRes, err := s.Run(ctx, fmt.Sprintf("MATCH (n:`%s`) RETURN count(n) AS c", label), nil)
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	recs, err := cntRes.Collect(ctx)
	if err != nil {
		return nil, err
	}
	total := int64(0)
	if len(recs) > 0 {
		if c, ok := recs[0].Get("c"); ok {
			total = toInt64(c)
		}
	}
	res, err := s.Run(ctx, fmt.Sprintf("MATCH (n:`%s`) RETURN elementId(n) AS eid, id(n) AS nid, properties(n) AS props ORDER BY id(n) SKIP $skip LIMIT $limit", label),
		map[string]any{"skip": skip, "limit": limit})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	recs2, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	page := &Neo4jNodePage{Total: total, Skip: skip, Limit: limit, Nodes: make([]string, 0, len(recs2))}
	for _, r := range recs2 {
		eid, _ := r.Get("eid")
		nid, _ := r.Get("nid")
		props, _ := r.Get("props")
		m := map[string]any{}
		if pm, ok := props.(map[string]any); ok {
			m = pm
		}
		m["_elementId"] = toString(eid)
		m["_id"] = toInt64(nid)
		b, _ := json.Marshal(m)
		page.Nodes = append(page.Nodes, string(b))
	}
	return page, nil
}

// ---- Neo4jEditor 能力 ----

func (d *neo4jDriver) CreateNode(ctx context.Context, database, label, propsJSON string) (int64, error) {
	if err := ValidateIdent(label); err != nil {
		return 0, err
	}
	props, err := parsePropsJSON(propsJSON)
	if err != nil {
		return 0, err
	}
	s, done := d.session(ctx, database, "write")
	defer done()
	res, err := s.Run(ctx, fmt.Sprintf("CREATE (n:`%s`) SET n = $props RETURN id(n) AS nid", label), map[string]any{"props": props})
	if err != nil {
		return 0, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return 0, err
	}
	if len(recs) == 0 {
		return 0, errs.Wrapc(errs.CodeBadRequest, "创建节点无返回")
	}
	nid, _ := recs[0].Get("nid")
	return toInt64(nid), nil
}

func (d *neo4jDriver) UpdateNodeProps(ctx context.Context, database, idJSON, propsJSON string) error {
	id, err := parseNodeID(idJSON)
	if err != nil {
		return err
	}
	props, err := parsePropsJSON(propsJSON)
	if err != nil {
		return err
	}
	s, done := d.session(ctx, database, "write")
	defer done()
	// SET += 合并语义：键值覆盖、新键追加；显式置 null 删除属性
	_, err = s.Run(ctx, "MATCH (n) WHERE id(n) = $id SET n += $props", map[string]any{"id": id, "props": props})
	return neo4jWrap(err)
}

func (d *neo4jDriver) DeleteNode(ctx context.Context, database, idJSON string, detach bool) error {
	id, err := parseNodeID(idJSON)
	if err != nil {
		return err
	}
	cy := "MATCH (n) WHERE id(n) = $id DELETE n"
	if detach {
		cy = "MATCH (n) WHERE id(n) = $id DETACH DELETE n"
	}
	s, done := d.session(ctx, database, "write")
	defer done()
	_, err = s.Run(ctx, cy, map[string]any{"id": id})
	return neo4jWrap(err)
}

func (d *neo4jDriver) ListIndexes(ctx context.Context, database, label string) ([]Neo4jIndexInfo, error) {
	s, done := d.session(ctx, database, "read")
	defer done()
	res, err := s.Run(ctx, "SHOW INDEXES YIELD name, type, entityType, labelsOrTypes, properties, state WHERE name <> '' RETURN name, type, entityType, labelsOrTypes, properties, state ORDER BY name", nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Neo4jIndexInfo, 0, len(recs))
	for _, r := range recs {
		info := Neo4jIndexInfo{}
		if v, ok := r.Get("name"); ok {
			info.Name = toString(v)
		}
		if v, ok := r.Get("type"); ok {
			info.Type = toString(v)
		}
		if v, ok := r.Get("entityType"); ok {
			info.EntityType = toString(v)
		}
		if v, ok := r.Get("labelsOrTypes"); ok {
			info.LabelsOrTypes = toStringSlice(v)
		}
		if v, ok := r.Get("properties"); ok {
			info.Properties = toStringSlice(v)
		}
		if v, ok := r.Get("state"); ok {
			info.State = toString(v)
		}
		if label == "" || containsStr(info.LabelsOrTypes, label) {
			out = append(out, info)
		}
	}
	return out, nil
}

func (d *neo4jDriver) CreateIndex(ctx context.Context, database, name, label string, props []string, unique bool) error {
	if err := ValidateIdent(label); err != nil {
		return err
	}
	if len(props) == 0 {
		return errs.Wrapc(errs.CodeBadRequest, "索引至少需要一个属性")
	}
	if name != "" {
		if err := ValidateIdent(name); err != nil {
			return err
		}
	}
	quoted := make([]string, 0, len(props))
	for _, p := range props {
		if err := ValidateIdent(p); err != nil {
			return err
		}
		quoted = append(quoted, fmt.Sprintf("n.`%s`", p))
	}
	var cy string
	if unique {
		// Neo4j 5 语法：FOR ... REQUIRE（ON/ASSERT 是 v4 语法，v5 拒绝）
		cy = fmt.Sprintf("CREATE CONSTRAINT %s FOR (n:`%s`) REQUIRE (%s) IS UNIQUE",
			constraintName(name, label, props), label, strings.Join(quoted, ", "))
	} else {
		cy = fmt.Sprintf("CREATE INDEX %s FOR (n:`%s`) ON (%s)",
			indexName(name, label, props), label, strings.Join(quoted, ", "))
	}
	s, done := d.session(ctx, database, "write")
	defer done()
	_, err := s.Run(ctx, cy, nil)
	return neo4jWrap(err)
}

func (d *neo4jDriver) DropIndex(ctx context.Context, database, name string) error {
	if err := ValidateIdent(name); err != nil {
		return err
	}
	// 先试索引后试约束（唯一约束以索引形式呈现，前端不区分来源）
	s, done := d.session(ctx, database, "write")
	defer done()
	if _, err := s.Run(ctx, fmt.Sprintf("DROP INDEX `%s` IF EXISTS", name), nil); err != nil {
		if _, err2 := s.Run(ctx, fmt.Sprintf("DROP CONSTRAINT `%s` IF EXISTS", name), nil); err2 != nil {
			return neo4jWrap(err)
		}
	}
	return nil
}

func (d *neo4jDriver) ListConstraints(ctx context.Context, database, label string) ([]Neo4jConstraintInfo, error) {
	s, done := d.session(ctx, database, "read")
	defer done()
	res, err := s.Run(ctx, "SHOW CONSTRAINTS YIELD name, type, entityType, labelsOrTypes, properties RETURN name, type, entityType, labelsOrTypes, properties ORDER BY name", nil)
	if err != nil {
		return nil, err
	}
	recs, err := res.Collect(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Neo4jConstraintInfo, 0, len(recs))
	for _, r := range recs {
		info := Neo4jConstraintInfo{}
		if v, ok := r.Get("name"); ok {
			info.Name = toString(v)
		}
		if v, ok := r.Get("type"); ok {
			info.Type = toString(v)
		}
		if v, ok := r.Get("entityType"); ok {
			info.EntityType = toString(v)
		}
		if v, ok := r.Get("labelsOrTypes"); ok {
			info.LabelsOrTypes = toStringSlice(v)
		}
		if v, ok := r.Get("properties"); ok {
			info.Properties = toStringSlice(v)
		}
		if label == "" || containsStr(info.LabelsOrTypes, label) {
			out = append(out, info)
		}
	}
	return out, nil
}

// ---- 助手 ----

func neo4jWrap(err error) error {
	if err == nil {
		return nil
	}
	return errs.Wrapc(errs.CodeFileOpFailed, err.Error())
}

// neo4jCounters ResultSummary 计数摘要（写语句反馈）。
func neo4jCounters(summary neo4jdb.ResultSummary) string {
	c := summary.Counters()
	parts := make([]string, 0, 6)
	add := func(name string, v int) {
		if v > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", name, v))
		}
	}
	add("nodesCreated", c.NodesCreated())
	add("nodesDeleted", c.NodesDeleted())
	add("relsCreated", c.RelationshipsCreated())
	add("relsDeleted", c.RelationshipsDeleted())
	add("propertiesSet", c.PropertiesSet())
	add("indexesAdded", c.IndexesAdded())
	add("constraintsAdded", c.ConstraintsAdded())
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// neo4jToJSON 把 Neo4j 值转 JSON 可序列化形态（Node/Relationship/Path 展开，时间类型字符串化）。
func neo4jToJSON(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case neo4jdb.Node:
		m := map[string]any{"_id": t.Id, "_elementId": t.ElementId, "_labels": t.Labels}
		for k, pv := range t.Props {
			m[k] = neo4jToJSON(pv)
		}
		return m
	case neo4jdb.Relationship:
		m := map[string]any{"_id": t.Id, "_elementId": t.ElementId, "_type": t.Type, "_start": t.StartId, "_end": t.EndId}
		for k, pv := range t.Props {
			m[k] = neo4jToJSON(pv)
		}
		return m
	case neo4jdb.Path:
		nodes := make([]any, 0)
		rels := make([]any, 0)
		for _, n := range t.Nodes {
			nodes = append(nodes, neo4jToJSON(n))
		}
		for _, r := range t.Relationships {
			rels = append(rels, neo4jToJSON(r))
		}
		return map[string]any{"_path": true, "nodes": nodes, "relationships": rels}
	case neo4jdb.Date, neo4jdb.LocalTime, neo4jdb.Time, neo4jdb.LocalDateTime, neo4jdb.Duration:
		return fmt.Sprintf("%v", v)
	case neo4jdb.Point2D, neo4jdb.Point3D:
		return fmt.Sprintf("%v", v)
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, neo4jToJSON(item))
		}
		return out
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, item := range t {
			m[k] = neo4jToJSON(item)
		}
		return m
	default:
		return v
	}
}

// parsePropsJSON 属性 JSON → map（Cypher 参数注入，天然参数化）。
func parsePropsJSON(s string) (map[string]any, error) {
	if strings.TrimSpace(s) == "" {
		return map[string]any{}, nil
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "属性 JSON 不合法: "+err.Error())
	}
	return m, nil
}

// parseNodeID 节点定位 JSON：接受 {"id":123} / {"elementId":"4:xx:0"} / 纯数字字符串。
func parseNodeID(s string) (map[string]any, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil, errs.Wrapc(errs.CodeBadRequest, "节点定位为空")
	}
	if strings.HasPrefix(t, "{") {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(t), &m); err == nil {
			if _, ok := m["id"]; ok {
				return map[string]any{"id": toInt64(m["id"])}, nil
			}
			if eid, ok := m["elementId"]; ok {
				return map[string]any{"elementId": toString(eid)}, nil
			}
		}
		return nil, errs.Wrap(errs.ErrBadRequest, "节点定位 JSON 不合法（需 id 或 elementId）")
	}
	return map[string]any{"id": toInt64(t)}, nil
}

// indexName/constraintName 自动命名（未指定时）。
func indexName(name, label string, props []string) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("yp_idx_%s_%s", strings.ToLower(label), strings.Join(props, "_"))
}

func constraintName(name, label string, props []string) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("yp_uniq_%s_%s", strings.ToLower(label), strings.Join(props, "_"))
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		var n int64
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}

func toStringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, toString(item))
	}
	return out
}
