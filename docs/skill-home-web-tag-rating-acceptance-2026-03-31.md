# Skill Home Web 标签与评分闭环自动化验收报告

测试日期：2026-03-31  
测试方式：Playwright CLI 无头浏览器自动化验收  
测试环境：

- Web：`http://127.0.0.1:4173`
- Mock API：`http://127.0.0.1:5005`
- 浏览器：Chrome Headless

## 验收范围

- 技能详情页未登录态是否展示社区标签与评分模块
- 未登录点击评分是否跳转登录，并保留回跳参数
- 登录后是否可以添加社区标签
- 登录后是否可以提交评分和可选短评
- 提交评分后详情页是否回显 `你的评分`、更新均分与评分人数
- 列表页是否直接展示评分信号
- `高评分` 排序是否按评分结果生效
- 详情页控制台是否出现前端错误

## 验收结论

本轮验收通过，没有发现阻塞本次交付的前端问题。

已确认：

- 详情页中，`Community tags` 位于 `Official tags` 之前，符合“社区标签前置”的设计目标。
- 未登录状态下，选择星级后点击 `登录后评分` 会跳转到 `/login?redirect=%2Fskills%2Facme%2Fdeploy-buddy`。
- 登录后可直接新增社区标签，页面原地回显新增标签、`Your tags` 和成功提示。
- 登录后可直接提交评分与短评，页面原地回显 `评分已保存。`、`你的评分：5/5`、`4.7 分` 与 `3 人评分`。
- 列表页结果项直接展示评分分数与人数。
- 在 `sort=rating` 场景下，`4.8 分` 的 `doc-helper` 排在 `4.7 分` 的 `deploy-buddy` 前面。
- 详情页控制台 `error` 数量为 `0`。

## 关键链路

### 1. 未登录评分跳转

- 打开：`/skills/acme/deploy-buddy`
- 详情页可见 `Community tags`、`Community rating`、`Official tags`
- 先选择 `5 星`
- 点击 `登录后评分`
- 页面跳转到：`/login?redirect=%2Fskills%2Facme%2Fdeploy-buddy`

### 2. 登录后添加社区标签

- 使用测试账号登录：
  - 邮箱：`tester@example.com`
  - 密码：`password123`
- 返回详情页后，在 `Add tag` 输入 `deployment`
- 点击 `Add tag`
- 页面提示：`社区标签已更新。`
- 页面新增：
  - 社区标签 `deployment`
  - `Your tags` 区域中的 `deployment`

### 3. 登录后提交评分

- 在详情页选择 `5 星`
- 在 `Add comment` 输入 `Helpful for deployment checks.`
- 点击 `保存评分`
- 页面提示：`评分已保存。`
- 页面回显：
  - `4.7 分`
  - `3 人评分`
  - `你的评分：5/5`
  - 评论输入框保留 `Helpful for deployment checks.`

### 4. 列表页评分信号与排序

- 打开：`/skills?sort=rating`
- 结果项直接显示评分信号
- 排序结果：
  - `doc-helper`：`4.8 分`，`5 人评分`
  - `deploy-buddy`：`4.7 分`，`3 人评分`

## 测试工件

- [详情页未登录态快照](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-17-20-107Z.yml)
- [登录页回跳快照](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-17-51-877Z.yml)
- [社区标签提交后快照](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-18-39-704Z.yml)
- [评分提交后快照](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-19-05-987Z.yml)
- [高评分排序快照](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-21-03-625Z.yml)
- [列表页截图](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-21-24-018Z.png)
- [详情页截图](/mnt/d/code/soul-store/skill-manager/.playwright-cli/page-2026-03-31T13-21-50-744Z.png)
- [详情页控制台检查结果](/mnt/d/code/soul-store/skill-manager/.playwright-cli/console-2026-03-31T13-21-46-474Z.log)

## 备注

- 本轮使用本地 mock API 进行可控验收，避免对真实环境写入标签和评分数据。
- 验收中曾发现第一版测试夹具没有模拟服务端 `sort=rating`；修正 mock 后重新验证，最终排序结果符合预期。
