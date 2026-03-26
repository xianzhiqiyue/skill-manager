# 执行计划（plan）

## 1. 总体执行顺序
1. 先完成文档基线，冻结需求、范围、验收口径与分步计划。
2. 再重构前端骨架，包括路由、全局布局和顶部导航。
3. 按主用户路径逐步迁移页面：首页 → 技能中心 → 技能详情/安装指引 → 发布页/控制台。
4. 最后统一做样式收口、构建验证和 smoke test。

## 2. Step 清单

| Step | 目标 | 修改范围 | DoD | 验证命令 | 回滚方案 |
| --- | --- | --- | --- | --- | --- |
| 1 | 完成 Web 重构文档基线 | `docs/skill-home-web-redesign-spec.md`、`.codex/specflow/tasks/T20260322-001-skill-home-web-flat-redesign/*`、`README.md` | 规格、计划、进度文档齐备，README 可见入口已补充 | 文档人工 review | 删除新增文档并回退 README |
| 2 | 搭建多页面骨架与全站导航 | `skill-home-web/src/main.tsx`、`skill-home-web/src/App.tsx`、新增页面/布局组件、样式文件 | 路由结构建立，导航可用，首页不再承载控制台功能 | `npm run build` | 回退到单页入口实现 |
| 3 | 完成首页与技能中心重构 | `skill-home-web/src/pages/*`、`skill-home-web/src/components/*`、`skill-home-web/src/api.ts`、样式文件 | 首页扁平化完成，技能中心具备增强浏览与查找能力 | `npm run build` | 回退到旧页面结构 |
| 4 | 完成技能详情、AI 安装指引、发布页与控制台迁移 | `skill-home-web/src/pages/*`、`skill-home-web/src/components/*`、`skill-home-web/src/api.ts`、样式文件 | 详情页、复制指引、发布页、控制台完整可用 | `npm run build`，手工 smoke test | 逐页回退到前一版页面 |
| 5 | 样式收口与回归验证 | 前端全量相关文件 | 视觉统一、信息密度达标、主要流程 smoke test 通过 | `npm run build` | 回退最后一轮样式收口 |

## 3. 质量门禁
- [ ] 类型检查通过
- [ ] Lint 通过
- [ ] Build 通过
- [ ] 核心测试通过
- [ ] 手工 smoke test 完成

## 4. 阶段性检查点
- 检查点A：文档完成，需求冻结。
- 检查点B：导航、首页、技能中心可独立访问。
- 检查点C：详情页、安装指引、发布页、控制台全部迁移完成。
