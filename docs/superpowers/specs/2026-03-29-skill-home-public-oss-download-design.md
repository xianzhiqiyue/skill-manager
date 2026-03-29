# Skill Home 公共 Skill 包 OSS 直链分发设计

## 1. 目标

将公共 skill 包的分发链路从 Skill Home 应用服务器迁移到阿里云 OSS，同时保留 Skill Home 作为管理、搜索和发布系统。

期望的用户可见结果是：

- 公共 skill 的元数据仍然由 Skill Home 提供
- 新发布的公共 skill 会返回基于 OSS 的 `download_url`
- Web UI 下载公共 skill 时不再经过应用服务器
- CLI 也会优先使用服务端返回的 `download_url`
- 旧的 `GET /api/v1/download/:namespace/:name/:version` 路由在兼容期内继续可用

这样可以先完成对象存储承载的包分发，而不必等私有 skill 鉴权方案落地后再推进。

## 2. 现状

当前 Skill Home 同时承担了元数据注册中心和二进制分发网关两种职责：

- skill 归档文件通过 Skill Home 上传，并由对象存储抽象层负责保存
- 数据库为每个版本保存元数据以及对应的 `storage_path`
- 公共 API 响应里的 `download_url` 目前返回 `/api/v1/download/...`
- 下载路由会读取对象内容、按需转换归档格式、递增下载计数，然后把字节流返回给客户端
- Web UI 当前是通过推导 `/api/v1/download/...` 来生成下载链接
- CLI 也把 `/api/v1/download/...` 写死在实现里，不信任服务端返回的 `download_url`

这意味着：

- skill 包下载带宽仍然压在应用服务器上
- 仅仅切换对象存储，并不能把下载流量从 Skill Home 上真正卸掉
- 客户端部分依赖了旧下载路由的路径形状

## 3. 设计决策

采用 OSS 作为公共 skill 包的下载源，并把 `download_url` 设为公共 skill 包分发的权威地址。

### 3.1 核心决策

对于公共 skill：

- 归档对象存放在 OSS
- API 响应中的 `download_url` 返回 OSS 绝对地址
- 客户端优先使用这个地址下载

为了兼容现有客户端：

- `GET /api/v1/download/:namespace/:name/:version` 继续保留
- 当 skill 为公共 skill，且请求的输出格式与存储归档格式一致时，该路由返回 `302` 跳转到同一个 OSS 对象
- 只有仍然需要服务端归档转换的边缘场景，才继续由 Skill Home 读取并回传

对于私有或团队内 skill：

- 本阶段不引入新的直链下载行为
- 后续可以在相同元数据模型下，把公共直链替换为 OSS 签名地址，或恢复由网关控制的下载方式

### 3.2 为什么这样设计

这个方案有意把 Skill Home 定位成元数据和控制平面，而不是主要的包传输平面。

与“仅把 MinIO 换成 OSS，但仍走旧下载网关”相比，这个方案真正把公共 skill 包流量从应用服务器上卸掉了。

与“直接删除旧下载路由”相比，这个方案不会在迁移过程中打断已经安装出去的旧版 CLI，也不会让旧页面逻辑立刻失效。

## 4. 范围

### 4.1 本阶段包含

- 将公共 skill 归档文件存放到 OSS
- 增加公共对象访问基址的配置
- 在创建和发布 API 中返回 OSS `download_url`
- 让 Web UI 使用 API 返回的 `download_url`
- 让 CLI 优先使用 API 返回的 `download_url`，并在需要时回退到旧下载路由
- 保留旧下载路由，并将其作为公共 skill 的兼容跳转入口
- 支持把已有公共 skill 归档分阶段迁移到 OSS

### 4.2 本阶段不包含

- 私有 skill 的签名下载地址
- 团队内 skill 的鉴权重构
- CDN 加速策略
- 除了生成直链所需内容之外的元数据模型变更
- 在本阶段移除旧下载路由

## 5. 约束与假设

- 第一阶段允许公共 skill 包公开下载。
- 私有和团队内 skill 的处理留到下一阶段。
- OSS 会提供适合直接下载的公网域名或自定义域名。
- 客户端在迁移期间需要同时兼容绝对下载地址和旧的相对路径。
- 现有 `storage_path` 仍然是定位对象的唯一真实来源。

## 6. 架构

### 6.1 存储模型

保留当前的对象 key 结构：

- `skills/<namespace>/<name>/<uuid>.<ext>`

Skill Home 继续像现在一样生成并保存 `storage_path`。变化不在对象命名，而在于如何根据 `storage_path` 推导公共下载地址。

存储配置需要新增公共 URL 概念，例如：

- `storage.public_base_url`
- 可选的 `storage.download_strategy`

对象存储抽象层应当能够在启用公共直链分发时，根据对象 key 生成对应的公共 URL。

### 6.2 API 模型

`download_url` 将成为公共 skill 包分发的权威服务端契约。

创建和发布响应对公共 skill 应返回 OSS 绝对地址，而不是相对的 `/api/v1/download/...` 路径。

技能详情和列表类响应也应保持一致语义，以便 Web 和 CLI 都能依赖同一个字段。

### 6.3 兼容路由

旧下载路由继续保留：

- 如果 skill 是公共 skill，且请求输出格式与存储格式一致，则返回 `302 Found` 到 OSS URL
- 如果仍然需要旧的服务端格式转换路径，则继续由 Skill Home 返回转换后的归档
- 如果将来 skill 为私有 skill，这个路由仍然是重新引入鉴权和签名 URL 的天然落点

这样一来，旧下载路由就从主下载通道变成了兼容桥接层。

## 7. 客户端行为

### 7.1 Web UI

Web UI 不应再通过拼接 `/api/v1/download/...` 来推导公共下载链接。

应改为：

- 如果 API 数据里有 `download_url`，直接使用
- 只有旧接口响应里没有该字段时，才回退到旧下载路由

这样可以让浏览器下载逻辑和后端契约保持一致，避免在前端重复维护下载 URL 规则。

### 7.2 CLI

CLI 当前在安装和拉取流程中把 `/api/v1/download/...` 写死了。

应改为：

1. 先获取包含 `download_url` 的 skill 元数据或版本元数据
2. 有可用地址时直接使用
3. 只有遇到还未支持该字段的旧服务端时，才回退到 `/api/v1/download/...`

这样可以让新版本 CLI 与直链模式对齐，同时保留对混合环境的兼容能力。

### 7.3 现有客户端

已经安装出去的旧版 CLI，以及仍然依赖旧逻辑的页面，在兼容期内都可以继续通过 `/api/v1/download/...` 正常工作。

## 8. 推出步骤

### 8.1 第一阶段：服务端能力到位

- 增加公共 OSS URL 配置
- 让存储层或辅助函数能够根据 `storage_path` 构造公共 URL
- 更新创建和发布响应，返回 OSS `download_url`
- 修改旧下载路由，使其在公共直链场景下返回重定向

完成后：

- 新发布的 skill 已经可以指向 OSS
- 旧客户端依然可用

### 8.2 第二阶段：客户端切换

- 更新 Web 下载逻辑，信任 `download_url`
- 更新 CLI 下载逻辑，信任 `download_url`
- 发布新的 CLI 版本

完成后：

- 大部分公共 skill 包流量应当已经从应用服务器迁到 OSS

### 8.3 第三阶段：历史数据回填

- 如果已有公共 skill 包尚未在 OSS 中存在，则将其迁移过去
- 校验数据库里的 `storage_path` 是否都能映射到可访问对象
- 从 Web 和 CLI 角度抽样验证历史公共 skill

### 8.4 第四阶段：私有 skill 后续

- 为团队内 skill 引入 OSS 签名 URL 或网关控制下载
- 决定私有对象的 `download_url` 是否变成时效性地址，还是继续只允许旧下载路由承载私有访问

## 9. 异常处理

预期行为如下：

- 如果缺少 OSS 公共 URL 配置，公共 skill 应继续使用旧下载路由，而不是返回失效绝对地址
- 如果 OSS 上缺少某个公共对象，旧下载路由应返回清晰的下载失败，而不是重定向到死链
- 如果请求格式需要服务端转换，不能盲目重定向，必须继续使用现有转换链路
- 如果客户端拿到的是绝对 `download_url`，就不能再把它相对到 registry base URL 上

## 10. 测试策略

### 10.1 服务端测试

- 验证公共创建和发布响应会返回 OSS 绝对地址
- 验证公共 skill 的旧下载请求在无需转换时会重定向到 OSS
- 验证格式转换请求仍然返回转换后的归档内容
- 验证未配置公共 URL 时能安全回退

### 10.2 Web 测试

- 验证 skill 详情页下载按钮使用 API 返回的 `download_url`
- 验证发布成功页使用返回的 `download_url`
- 验证面对旧载荷时，回退逻辑仍然正常

### 10.3 CLI 测试

- 验证下载逻辑在存在 `download_url` 时会优先使用它
- 验证元数据缺少 `download_url` 时会回退到 `/api/v1/download/...`
- 验证绝对 OSS URL 不会被错误地按 registry base URL 重写

### 10.4 运行验证

- 发布一个公共测试 skill
- 确认 API 载荷返回 OSS 地址
- 确认浏览器下载直接命中 OSS
- 确认当前 CLI 能通过新元数据路径安装该 skill
- 确认旧版 CLI 仍能通过 `/api/v1/download/...` 成功下载

## 11. 受影响代码区域

主要涉及文件：

- `skill-home-server/internal/config/config.go`
- `skill-home-server/internal/storage/object.go`
- `skill-home-server/internal/api/handlers/skill.go`
- `skill-home-server/internal/api/handlers/version.go`
- `skill-home-server/internal/api/handlers/handler_integration_test.go`
- `skill-home-web/src/api.ts`
- `skill-home-web/src/pages/PublishNewPage.tsx`
- `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- `skill-home-cli/internal/registry/client.go`
- `skill-home-cli/internal/registry/types.go`
- `skill-home-cli/internal/registry/client_test.go`
- 描述下载语义的 API 文档和 README

## 12. 取舍

优点：

- 公共 skill 包带宽迁移到 OSS
- Web 和 CLI 对齐到统一的后端契约
- 因为旧下载路由仍在，迁移过程具备向后兼容性
- 后续私有 skill 支持仍保留清晰的演进路径

代价：

- 直链模式会提高系统对对象托管配置的依赖
- 客户端需要学会处理绝对下载地址
- 将来做私有 skill 时，需要在签名 URL 和混合网关模式之间做进一步设计

这个取舍是合理的，因为当前目标就是尽快把公共 skill 包分发从应用服务器迁移出去，同时保留后续私有下载方案的安全演进空间。
