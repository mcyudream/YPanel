package dto

// M26 P2：源码构建生成 compose 项目（git/源码 → 语言检测 → 模板构建 → compose 项目）。

// Src2ComposeFacts 标记文件内容线索（agent 提取，core 据此给建议）。
type Src2ComposeFacts struct {
	Version   string   `json:"version,omitempty"`  // go 1.23 / java 21 / node 20 等
	StartCmd  string   `json:"startCmd,omitempty"` // package.json scripts.start
	BuildCmd  string   `json:"buildCmd,omitempty"` // package.json scripts.build
	PkgMgr    string   `json:"pkgMgr,omitempty"`   // pnpm/yarn/npm（按 lockfile 识别）
	Packaging string   `json:"packaging,omitempty"` // maven packaging：pom=多模块父/jar=可构建模块
	Modules   []string `json:"modules,omitempty"`  // maven <module> 列表（父 POM 时）
	Boot      bool     `json:"boot,omitempty"`     // spring-boot-maven-plugin 存在
}

// Src2ComposeDetectItem 单目录检测结果：每个目录只报最高优先级命中的标记。
type Src2ComposeDetectItem struct {
	Dir    string           `json:"dir"`              // 相对仓库根；"" 为根目录
	Marker string           `json:"marker"`           // Dockerfile/go.mod/pom.xml/package.json/...
	Facts  Src2ComposeFacts `json:"facts,omitempty"`
}

// Src2ComposeDetectResp 检测结果（preview / detect 共用）。
type Src2ComposeDetectResp struct {
	Commit string                  `json:"commit,omitempty"` // 检出 commit（preview 时返回）
	Items  []Src2ComposeDetectItem `json:"items"`
}

// Src2ComposePreviewReq 预检（现走 SSE 流式接口，参数经 query 传入）。
type Src2ComposePreviewReq struct {
	NodeID       string `json:"nodeId"` // M57 目标节点（空=local）
	GitURL       string `json:"gitUrl" binding:"required"`
	Branch       string `json:"branch"`
	CredentialID uint   `json:"credentialId"`
	Token        string `json:"token"`      // https 密码/token（凭据库匹配后由 core 下发）
	Username     string `json:"username"`   // https 用户名（凭据下发）
	PrivateKey   string `json:"privateKey"` // ssh 私钥（凭据库匹配后由 core 下发）
}

// Src2ComposeCloneTmpReq 临时目录克隆（preview 流式预检用，目录由 agent 创建于 /tmp/yp-src2-*）。
type Src2ComposeCloneTmpReq struct {
	GitURL     string `json:"gitUrl" binding:"required"`
	Branch     string `json:"branch"`
	Token      string `json:"token"`
	Username   string `json:"username"`
	PrivateKey string `json:"privateKey"`
}

// Src2ComposeCloneTmpResp 临时克隆结果（dir 供 detect/cleanup，前缀 /tmp/yp-src2-）。
type Src2ComposeCloneTmpResp struct {
	Dir    string `json:"dir"`
	Commit string `json:"commit"`
}

// Src2ComposeCloneReq 克隆源码到托管项目 src/（重复创建走 fetch+reset 更新）。
type Src2ComposeCloneReq struct {
	Name       string `json:"name" binding:"required"`
	GitURL     string `json:"gitUrl" binding:"required"`
	Branch     string `json:"branch"`
	Token      string `json:"token"`
	Username   string `json:"username"`
	PrivateKey string `json:"privateKey"`
}

// Src2ComposeServiceSpec 用户确认后的单服务定义（monorepo 多选即多服务）。
type Src2ComposeServiceSpec struct {
	Name          string `json:"name" binding:"required"`
	Dir           string `json:"dir"`                // 相对仓库根的构建目录
	Lang          string `json:"lang" binding:"required"` // go/node/node-build/java-maven/java-gradle/python/php/static/dockerfile
	Version       string `json:"version"`            // 语言版本线索（go 1.23 / node 20 / java 21），空=模板默认
	HostPort      int    `json:"hostPort"`           // 0 = 不发布端口
	ContainerPort int    `json:"containerPort"`      // static/php/node-build 固定 80
	StartCmd      string `json:"startCmd"`           // 运行模式启动命令（node/python/go）
	Module        string `json:"module"`             // maven 多模块：从构建目录起的模块相对路径（空=单模块）
	BuildCmd      string `json:"buildCmd"`           // node-build 构建命令（空=按包管理器推导）
	DistDir       string `json:"distDir"`            // node-build 产物目录（空=dist）
}

// Src2ComposeBuildReq 创建构建任务（core 渲染模板 + 编排，agent 执行 clone/build）。
// CredentialID：0=按 host 自动匹配凭据库；凭据命中后 core 解析为 Token/PrivateKey 下发。
type Src2ComposeBuildReq struct {
	NodeID       string                   `json:"nodeId"` // M57 目标节点（空=local）
	Name         string                   `json:"name" binding:"required"`
	GitURL       string                   `json:"gitUrl" binding:"required"`
	Branch       string                   `json:"branch"`
	CredentialID uint                     `json:"credentialId"`
	Services     []Src2ComposeServiceSpec `json:"services" binding:"required,min=1"`
}

// Src2ComposeBuildResp 创建任务响应。
type Src2ComposeBuildResp struct {
	TaskID uint   `json:"taskId"`
	Commit string `json:"commit,omitempty"`
}
