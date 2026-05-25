# Skill Home 社交化与用户统计开发计划

**Goal:** 完成平台社交化第一批需求：注册中文名必填、skill 元信息带作者用户信息、点赞/分享入口、安装数量上报、用户贡献统计。

**影响面:** `skill-home-server`、`skill-home-cli`、`skill-home-web`、`API.md`、`README.md`、`skill-home-server/README.md`。

## 范围

- 注册新增 `display_name_zh`，服务端校验必填，Web 注册页必填。
- skill 列表、搜索、详情、用户 skill 列表返回 owner 用户信息，包含 `username` 和 `display_name_zh`。
- 新增 skill 点赞/取消点赞，返回 `like_count` 和 `viewer_liked`。
- 新增安装成功事件上报，维护 `install_count`。
- Web 详情页提供点赞按钮和分享按钮；分享优先使用浏览器分享能力，不可用时复制链接。
- 用户 Profile 展示服务端统计：产出 skill、获赞、下载、安装、评分。
- CLI `install` 成功后上报安装事件。

## 非目标

- 不做复杂推荐算法。
- 不做分享排行榜；分享事件可后续扩展，当前以分享入口为主。
- 不改生产发布版本号，不执行线上部署。

## 验收标准

- 注册接口缺少中文名时返回 400，包含中文名时正常注册并返回字段。
- skill 列表/搜索/详情能返回 owner 信息、点赞数、安装数。
- 登录用户可点赞和取消点赞，重复请求幂等，计数字段正确。
- CLI `install` 在同步成功后调用安装事件接口；上报失败不阻断本地安装。
- Profile 页面展示服务端用户统计，接口失败时退回本地基础统计。
- API 文档包含新增字段和接口。

## 验证计划

- `cd skill-home-server && go test ./...`
- `cd skill-home-cli && go test ./...`
- `cd skill-home-web && npm test -- --runInBand` 或 `npm test`
- `cd skill-home-web && npm run build`

## 风险与处理

- `display_name_zh` 对已有用户不能直接强制数据库非空；本轮通过注册 API 与 UI 强制，数据库字段允许历史空值。
- 安装事件上报属于运营统计，CLI 上报失败只提示 warning，不影响本地安装成功。
- 现有工作区已有用户改动；本轮只在必要文件上叠加，不回滚无关改动。
