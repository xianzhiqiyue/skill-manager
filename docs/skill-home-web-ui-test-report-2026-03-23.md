# Skill Home Web UI 测试报告

测试日期：2026-03-23（Playwright 运行日志为 UTC 2026-03-22 16:03-16:11，对应北京时间 2026-03-23 00:03-00:11）  
测试对象：[http://47.122.112.210:8080](http://47.122.112.210:8080)  
测试方式：真实浏览器走查 + 桌面/移动端截图 + DOM 快照 + 控制台检查

## 测试范围

- 页面范围：`/`、`/skills`、`/skills/testuser/skill-home-manager`
- 视口范围：
  - 桌面：默认桌面视口、`1440 x 1024`
  - 移动端：`390 x 844`
- 本轮重点：信息架构、视觉层级、移动端首屏、技能查找链路、详情页安装链路、复制反馈
- 未深测范围：登录后发布页、控制台、注册/登录表单的完整业务流

## 总体结论

当前版本不适合直接做 UI 验收通过，主要问题不是“零散样式瑕疵”，而是页面结构仍然偏重、重复信息太多、核心任务路径不够干净。

本轮确认的主要阻塞项：

1. 首页信息层级过载，`Highlights` 等模块与 Hero/Workflow/FAQ 重复。
2. 技能中心同时存在“筛选器 + 列表/卡片 + 即时预览侧栏”，主任务不够聚焦。
3. 移动端头部与技能中心首屏占位过大，结果列表进入过慢。
4. 详情页上半区空间利用率较低，版本表还出现错误发布时间。

本轮未发现的基础问题：

- 测试过的首页与技能中心移动端无横向溢出。
- 技能详情页控制台告警为 `0`，未发现前端报错或 warning。

## 问题清单

### P1 首页信息架构仍然过载，`Highlights` 应从首页移除

- 现象：
  - Hero 后面立刻进入 `Highlights`，文案内容与 Hero、Workflow、FAQ 高度重复。
  - 首页连续堆叠 `Highlights`、`Workflow`、`Recommended`、`首页快速安装指引`、`FAQ`，首屏之外仍然是连续解释型模块，缺少更强的“发现技能”主路径。
- 影响：
  - 首页更像“重构说明页”，而不是产品入口页。
  - 用户在真正进入技能中心之前，要先消费大量重复说明，认知负担偏高。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L529)
  - [App.tsx](../skill-home-web/src/App.tsx#L559)
  - [App.tsx](../skill-home-web/src/App.tsx#L575)
  - [App.tsx](../skill-home-web/src/App.tsx#L619)
  - [App.tsx](../skill-home-web/src/App.tsx#L625)
- 证据截图：
  - [首页桌面截图](../.playwright-cli/page-2026-03-22T16-03-32-830Z.png)
  - [首页移动端截图](../.playwright-cli/page-2026-03-22T16-04-56-719Z.png)
- 建议：
  - 删除 `Highlights` 模块，保留 Hero。
  - 把首页压缩成 `Hero + 推荐技能/最新技能 + 进入技能中心 CTA + 精简 FAQ`。
  - `Workflow` 只保留 3 步以内的简版，避免再次解释整套产品逻辑。

### P1 技能中心存在重复链路，`即时预览` 按钮和右侧预览侧栏应一起去掉

- 现象：
  - 当前技能中心同时包含左侧筛选器、中间列表/卡片、右侧 `即时预览` 侧栏。
  - 每张卡片又额外提供一个 `即时预览` 按钮，导致“卡片内容”和“右侧预览内容”重复展示同一 skill 的描述、版本、下载量、License。
  - 在移动端，虽然预览栏会折到下方，但它仍然是重复内容，只会拉长页面。
- 影响：
  - 用户的主任务应是“找到 skill 并进入详情页”，现在却被分散成“看卡片、点即时预览、再点详情页”三段。
  - 技能中心的信息密度虽然提高了，但有效信息密度没有提高，重复信息反而挤压了浏览面积。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L659)
  - [App.tsx](../skill-home-web/src/App.tsx#L801)
  - [App.tsx](../skill-home-web/src/App.tsx#L840)
  - [styles.css](../skill-home-web/src/styles.css#L593)
  - [styles.css](../skill-home-web/src/styles.css#L669)
  - [styles.css](../skill-home-web/src/styles.css#L730)
- 证据截图：
  - [技能中心桌面截图](../.playwright-cli/page-2026-03-22T16-04-15-781Z.png)
  - [技能中心移动端首屏](../.playwright-cli/page-2026-03-22T16-06-52-838Z.png)
  - [技能中心移动端下滚截图](../.playwright-cli/page-2026-03-22T16-07-20-374Z.png)
- 建议：
  - 删除所有 `即时预览` 按钮。
  - 删除 `PreviewPanel`。
  - 技能中心只保留两种主动作：`查看详情` 和 `快速安装/复制引用`。

### P1 版本表发布时间展示错误，当前 UI 已经影响可信度

- 现象：
  - 技能详情页版本表中的发布时间显示为 `1年1月1日 08:06`。
  - 这不是正常业务时间格式，用户会直接怀疑数据可靠性。
- 影响：
  - 这是比视觉更严重的 UI 展示缺陷，会直接伤害“可安装、可信”的产品定位。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L968)
  - [App.tsx](../skill-home-web/src/App.tsx#L973)
- 证据截图：
  - [版本表截图](../.playwright-cli/page-2026-03-22T16-11-11-714Z.png)
- 建议：
  - 优先核对 `published_at` / `created_at` 的真实值与格式化逻辑。
  - 遇到空值时不要输出时间零值，应回退为 `未发布` 或 `未知时间`。

### P2 移动端头部占位偏大，首屏太早进入“导航工具区”

- 现象：
  - 移动端头部同时放了品牌、搜索、登录、注册、菜单按钮。
  - 菜单展开后又追加一整块导航按钮区，导致首屏上半部几乎都在处理导航和账号入口。
- 影响：
  - 首页和技能中心都要先穿过一层很厚的顶栏，内容触达速度偏慢。
  - 对移动端来说，当前头部不是“导航”，而是“占位”。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L1640)
  - [App.tsx](../skill-home-web/src/App.tsx#L1702)
  - [styles.css](../skill-home-web/src/styles.css#L89)
  - [styles.css](../skill-home-web/src/styles.css#L1026)
  - [styles.css](../skill-home-web/src/styles.css#L1081)
- 证据截图：
  - [首页移动端截图](../.playwright-cli/page-2026-03-22T16-04-56-719Z.png)
  - [首页移动端菜单展开](../.playwright-cli/page-2026-03-22T16-05-43-466Z.png)
  - [技能中心移动端截图](../.playwright-cli/page-2026-03-22T16-06-52-838Z.png)
- 建议：
  - 移动端收敛为 `品牌 + 搜索入口/图标 + 菜单`。
  - `登录/注册` 放入抽屉式菜单，不要永久占首屏。
  - 顶栏尽量保持单层结构。

### P2 技能中心移动端首屏几乎被筛选器占满，结果列表进入过慢

- 现象：
  - 390 宽度下，技能中心先展示标题，再完整展示关键词、命名空间、标签、License、排序、视图和统计。
  - 用户在首屏内几乎看不到任何技能结果。
- 影响：
  - “技能中心”的主要任务应该是找技能，但当前移动端先把用户带进一个长表单。
  - 这和“高信息密度”的目标冲突，高的是控件密度，不是结果密度。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L660)
  - [styles.css](../skill-home-web/src/styles.css#L593)
  - [styles.css](../skill-home-web/src/styles.css#L1040)
- 证据截图：
  - [技能中心移动端首屏](../.playwright-cli/page-2026-03-22T16-06-52-838Z.png)
  - [技能中心移动端下滚截图](../.playwright-cli/page-2026-03-22T16-07-20-374Z.png)
- 建议：
  - 移动端把筛选器折叠成 `筛选` 抽屉或底部面板。
  - 默认先展示结果列表和结果数，再允许用户展开高级筛选。

### P2 详情页 Hero 区域空间效率偏低，按钮区与内容区关系松散

- 现象：
  - 技能名、描述在左侧堆叠，右上角只有一小组按钮，右侧与中上部留白明显。
  - 关键信息指标被拆到 Hero 下方的另一组卡片，视觉层级被切断。
- 影响：
  - 详情页第一屏没有形成一个紧凑的“摘要 + 主操作”区。
  - 用户先看到大块留白，再看到操作和指标，空间效率偏低。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L896)
  - [App.tsx](../skill-home-web/src/App.tsx#L939)
  - [styles.css](../skill-home-web/src/styles.css#L582)
  - [styles.css](../skill-home-web/src/styles.css#L1040)
- 证据截图：
  - [详情页首屏截图](../.playwright-cli/page-2026-03-22T16-08-31-509Z.png)
- 建议：
  - 详情页首屏改成更紧凑的双栏。
  - 把按钮和 2-3 个关键指标整合到同一个摘要区域，避免上半屏留白。

### P2 复制反馈过弱，失败时没有可见错误提示

- 现象：
  - `CopyButton` 只有按钮文案本地变成 `已复制`，没有全局 toast，也没有失败提示。
  - 本轮在真实浏览器自动化环境中，复制动作后页面未出现额外反馈；同时浏览器环境返回 `clipboard-missing`。
- 影响：
  - 用户很难判断复制是否真的成功。
  - 一旦环境不支持剪贴板，当前界面也不会告诉用户为什么失败。
- 代码定位：
  - [App.tsx](../skill-home-web/src/App.tsx#L128)
  - [App.tsx](../skill-home-web/src/App.tsx#L139)
- 建议：
  - 增加统一 toast。
  - 复制失败时给出明确错误提示，例如“浏览器不支持自动复制，请手动复制命令”。

### P3 当前视觉语言还没有完全落到扁平风

- 现象：
  - 顶栏、面板仍然使用半透明白底、阴影、模糊。
  - 品牌块和部分激活态仍使用渐变。
- 影响：
  - 与“扁平风、高密度”的目标不完全一致，视觉上仍保留了轻玻璃感和层叠质感。
- 代码定位：
  - [styles.css](../skill-home-web/src/styles.css#L80)
  - [styles.css](../skill-home-web/src/styles.css#L89)
  - [styles.css](../skill-home-web/src/styles.css#L108)
  - [styles.css](../skill-home-web/src/styles.css#L681)
- 建议：
  - 去掉 `backdrop-filter`、渐变和大面积柔和阴影。
  - 用纯色面、边框和轻微层级差解决分区关系。

## 建议整改顺序

1. 先删结构性冗余：首页 `Highlights`、技能中心 `即时预览` 和 `PreviewPanel`。
2. 再调移动端：压缩头部、把技能中心筛选器改成折叠式。
3. 再修详情页：压缩 Hero，修正版本时间显示。
4. 最后统一视觉：去模糊、去渐变、再收边距和卡片密度。

## 测试工件

- [首页桌面截图](../.playwright-cli/page-2026-03-22T16-03-32-830Z.png)
- [技能中心桌面截图](../.playwright-cli/page-2026-03-22T16-04-15-781Z.png)
- [首页移动端截图](../.playwright-cli/page-2026-03-22T16-04-56-719Z.png)
- [首页移动端菜单展开](../.playwright-cli/page-2026-03-22T16-05-43-466Z.png)
- [技能中心移动端首屏](../.playwright-cli/page-2026-03-22T16-06-52-838Z.png)
- [技能中心移动端下滚截图](../.playwright-cli/page-2026-03-22T16-07-20-374Z.png)
- [详情页首屏截图](../.playwright-cli/page-2026-03-22T16-08-31-509Z.png)
- [详情页安装区截图](../.playwright-cli/page-2026-03-22T16-08-53-413Z.png)
- [详情页版本区截图](../.playwright-cli/page-2026-03-22T16-11-11-714Z.png)
- [技能详情控制台检查结果](../.playwright-cli/console-2026-03-22T16-08-31-743Z.log)
