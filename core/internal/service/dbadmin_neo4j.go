// DB Admin Neo4j 图工作台服务（M68）：标签/关系元、节点分页、Cypher 控制台、节点编辑与索引/约束管理（P2）。
// 写路径全审计；查询走只读会话。
package service

import (
	"context"

	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// Neo4jGraphMeta 标签与关系类型清单（带元素计数）。
func (s *DBAdminService) Neo4jGraphMeta(ctx context.Context, instanceID uint, database string) (map[string]any, error) {
	var labels, rels []dbdriver.Neo4jLabelInfo
	err := s.withDriver(ctx, instanceID, dbAdminReadTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		nb, ok := drv.(dbdriver.Neo4jBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		var e1, e2 error
		labels, e1 = nb.ListLabels(ctx, database)
		rels, e2 = nb.ListRelTypes(ctx, database)
		if e1 != nil {
			return e1
		}
		return e2
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"labels": labels, "relTypes": rels}, nil
}

// Neo4jCypher Cypher 控制台执行（读语句只读会话、写语句写会话，统一 30s 上限）。
func (s *DBAdminService) Neo4jCypher(ctx context.Context, instanceID uint, username, database, cypher string, limit int) (*dbdriver.QueryResult, error) {
	var out *dbdriver.QueryResult
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		nb, ok := drv.(dbdriver.Neo4jBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		var err error
		out, err = nb.RunCypher(ctx, database, cypher, limit)
		s.audit(username, inst, "neo4j_cypher", database, summaryOf(cypher), err == nil)
		return err
	})
	return out, err
}

// Neo4jNodes 节点分页浏览。
func (s *DBAdminService) Neo4jNodes(ctx context.Context, instanceID uint, database, label string, skip, limit int) (*dbdriver.Neo4jNodePage, error) {
	var out *dbdriver.Neo4jNodePage
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		nb, ok := drv.(dbdriver.Neo4jBrowser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		var err error
		out, err = nb.BrowseNodes(ctx, database, label, skip, limit)
		return err
	})
	return out, err
}

// Neo4jNodeCreate 创建节点。
func (s *DBAdminService) Neo4jNodeCreate(ctx context.Context, instanceID uint, username, database, label, propsJSON string) (int64, error) {
	var id int64
	err := s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		ne, ok := drv.(dbdriver.Neo4jEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		var err error
		id, err = ne.CreateNode(ctx, database, label, propsJSON)
		s.audit(username, inst, "neo4j_node_create", database+"("+label+")", summaryOf(propsJSON), err == nil)
		return err
	})
	return id, err
}

// Neo4jNodeUpdate 合并更新节点属性（SET +=，null 值删属性）。
func (s *DBAdminService) Neo4jNodeUpdate(ctx context.Context, instanceID uint, username, database, idJSON, propsJSON string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		ne, ok := drv.(dbdriver.Neo4jEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		err := ne.UpdateNodeProps(ctx, database, idJSON, propsJSON)
		s.audit(username, inst, "neo4j_node_update", database, "id="+idJSON, err == nil)
		return err
	})
}

// Neo4jNodeDelete 删除节点（detach 级联删关系）。
func (s *DBAdminService) Neo4jNodeDelete(ctx context.Context, instanceID uint, username, database, idJSON string, detach bool) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		ne, ok := drv.(dbdriver.Neo4jEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		err := ne.DeleteNode(ctx, database, idJSON, detach)
		kind := "neo4j_node_delete"
		if detach {
			kind = "neo4j_node_detach_delete"
		}
		s.audit(username, inst, kind, database, "id="+idJSON, err == nil)
		return err
	})
}

// Neo4jIndexCreate 创建索引（unique=true 建唯一性约束）。
func (s *DBAdminService) Neo4jIndexCreate(ctx context.Context, instanceID uint, username, database, name, label string, props []string, unique bool) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		ne, ok := drv.(dbdriver.Neo4jEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		err := ne.CreateIndex(ctx, database, name, label, props, unique)
		s.audit(username, inst, "neo4j_index_create", database+"("+label+")", summaryOf(label+":"+joinStrings(props, ",")), err == nil)
		return err
	})
}

// Neo4jIndexDrop 删除索引/约束（按名）。
func (s *DBAdminService) Neo4jIndexDrop(ctx context.Context, instanceID uint, username, database, name string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		ne, ok := drv.(dbdriver.Neo4jEditor)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 Neo4j")
		}
		err := ne.DropIndex(ctx, database, name)
		s.audit(username, inst, "neo4j_index_drop", database, "name="+name, err == nil)
		return err
	})
}

// joinStrings strings.Join 短别名（审计摘要用）。
func joinStrings(list []string, sep string) string {
	out := ""
	for i, x := range list {
		if i > 0 {
			out += sep
		}
		out += x
	}
	return out
}
