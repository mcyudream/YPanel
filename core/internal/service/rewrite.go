// 伪静态模板库（内置常用 rewrite 模板）。
package service

import (
	"strings"

	"github.com/ypanel/shared/errs"
)

// rewriteTemplates 内置伪静态模板（name → location 内容；插入 server 段）。
var rewriteTemplates = map[string]string{
	"spa": `location / {
    try_files $uri $uri/ /index.html;
}`,
	"wordpress": `location / {
    try_files $uri $uri/ /index.php?$args;
}
location ~ \.php$ {
    deny all;
}`,
	"thinkphp": `location / {
    if (!-e $request_filename) {
        rewrite ^(.*)$ /index.php?s=$1 last;
        break;
    }
}`,
	"laravel": `location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
	"typecho": `location / {
    if (!-e $request_filename) {
        rewrite ^(.*)$ /index.php$1 last;
    }
}`,
	"discuz": `location / {
    rewrite ^([^\.]*)/topic-(.+)\.html$ $1/forum.php?mod=topic&topic=$2 last;
    rewrite ^([^\.]*)/article-([0-9]+)-([0-9]+)\.html$ $1/forum.php?mod=viewthread&tid=$2&page=$3 last;
    rewrite ^([^\.]*)/forum-(\w+)-([0-9]+)\.html$ $1/forum.php?mod=forumdisplay&fid=$2&page=$3 last;
}`,
}

// RewriteTemplateInfo 模板信息。
type RewriteTemplateInfo struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
}

// ListRewriteTemplates 模板清单（排序稳定）。
func ListRewriteTemplates() []RewriteTemplateInfo {
	names := []string{"spa", "wordpress", "thinkphp", "laravel", "typecho", "discuz"}
	out := make([]RewriteTemplateInfo, 0, len(names))
	for _, n := range names {
		out = append(out, RewriteTemplateInfo{Name: n, Content: rewriteTemplates[n]})
	}
	return out
}

// ResolveRewrite 按名称取模板内容；"custom" 或未知返回空（使用 RewriteContent）。
func ResolveRewrite(name string) (string, error) {
	if name == "" || name == "none" || name == "custom" {
		return "", nil
	}
	t, ok := rewriteTemplates[name]
	if !ok {
		return "", errs.Wrap(errs.ErrBadRequest, "未知伪静态模板: "+name)
	}
	return t, nil
}

// validateCustomLocations 自定义 location 基本校验（必须为 location 块；括号配平）。
func validateCustomLocations(items []CustomLocation) error {
	for _, it := range items {
		c := strings.TrimSpace(it.Content)
		if c == "" {
			continue
		}
		if !strings.HasPrefix(c, "location ") && !strings.HasPrefix(c, "if ") && !strings.HasPrefix(c, "#") {
			return errs.Wrap(errs.ErrBadRequest, "自定义 location 须以 location/if 开头: "+it.Comment)
		}
		if strings.Count(c, "{") != strings.Count(c, "}") {
			return errs.Wrap(errs.ErrBadRequest, "自定义 location 花括号不配平: "+it.Comment)
		}
	}
	return nil
}

// CustomLocation 自定义 location。
type CustomLocation struct {
	Comment string `json:"comment"`
	Content string `json:"content"`
}
