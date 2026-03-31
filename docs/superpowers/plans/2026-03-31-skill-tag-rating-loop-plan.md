# Skill Home 标签与评分轻闭环实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Skill Home Web 端补齐“社区标签 + 评分”详情页轻闭环，并让列表页展示评分信号，使标签负责场景理解、评分负责信任判断。

**Architecture:** 复用现有 server 的社区标签和评分接口，只在 Web 端补全类型、状态管理和交互展示。详情页把社区标签卡片前置，并新增评分卡片；列表页继续保留官方 tags 筛选，同时在结果项展示平均分与评分人数，让 `高评分` 排序具备可见解释。

**Tech Stack:** React 18、TypeScript、Vite、Vitest、Testing Library、现有 `skill-home-web` 页面与 hook 架构

---

## 文件结构映射

### 现有文件

- 修改：`skill-home-web/src/api.ts`
  - 补充评分相关类型与评分请求函数。
- 修改：`skill-home-web/src/hooks/useRegistryApp.ts`
  - 增加评分状态、提交逻辑和详情刷新。
- 修改：`skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
  - 调整详情页卡片顺序并新增评分卡片。
- 修改：`skill-home-web/src/pages/skill/SkillOverviewPage.test.tsx`
  - 覆盖评分卡片展示、登录态与提交行为。
- 修改：`skill-home-web/src/hooks/useRegistryApp.test.tsx`
  - 覆盖评分提交、登录跳转和状态更新。
- 修改：`skill-home-web/src/components/search/SkillResultList.tsx`
  - 在列表结果项展示均分与评分人数。
- 修改：`skill-home-web/src/components/search/SkillResultList.test.tsx`
  - 覆盖列表页评分信号展示。
- 修改：`skill-home-web/src/styles.css`
  - 为评分卡片与评分徽标补最小样式。

### 新文件

- 无

---

### Task 1: 补评分 API 契约与 hook 测试

**Files:**
- Modify: `skill-home-web/src/api.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`

- [ ] **Step 1: 写失败测试**
  - 在 `useRegistryApp.test.tsx` 中新增测试：
    - 已登录用户可以提交评分并更新 `detailSkill.user_rating`
    - 未登录用户尝试评分时会提示并跳转登录

- [ ] **Step 2: 运行测试确认失败**

Run: `npm test -- --run src/hooks/useRegistryApp.test.tsx`
Expected: FAIL，因为当前没有评分请求函数和评分状态管理。

- [ ] **Step 3: 做最小实现**
  - 在 `api.ts` 中新增：
    - `SkillRating`
    - `RateSkillPayload`
    - `RateSkillResponse`
    - `rateSkill(token, namespace, name, payload)`
  - 把 `SkillDetail` 类型补齐 `user_rating`
  - 在 `useRegistryApp.ts` 中新增评分相关 state 和 `submitSkillRating()`

- [ ] **Step 4: 重新运行测试**

Run: `npm test -- --run src/hooks/useRegistryApp.test.tsx`
Expected: PASS

- [ ] **Step 5: 自查重构**
  - 确认详情页 route 切换时评分状态会被正确清理
  - 确认登录跳转沿用现有 `buildAuthPath` 回跳机制

### Task 2: 先让详情页评分卡片跑通

**Files:**
- Modify: `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- Modify: `skill-home-web/src/pages/skill/SkillOverviewPage.test.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: 写失败测试**
  - 在 `SkillOverviewPage.test.tsx` 中新增测试：
    - 详情页会优先展示社区标签卡片和评分卡片
    - 已登录用户可点击星级并提交评分
    - 已有 `user_rating` 时会显示当前评分

- [ ] **Step 2: 运行测试确认失败**

Run: `npm test -- --run src/pages/skill/SkillOverviewPage.test.tsx`
Expected: FAIL，因为当前页面没有评分卡片和相关交互。

- [ ] **Step 3: 做最小实现**
  - 在 `SkillOverviewPage.tsx` 中：
    - 把社区标签卡片前移
    - 新增评分卡片
    - 支持 1-5 星选择
    - 支持可选短评输入和提交按钮
    - 展示均分、评分人数和当前用户评分
  - 在 `styles.css` 中补充评分卡片、星级按钮、轻量状态提示样式

- [ ] **Step 4: 重新运行测试**

Run: `npm test -- --run src/pages/skill/SkillOverviewPage.test.tsx`
Expected: PASS

- [ ] **Step 5: 自查重构**
  - 确认未登录态文案清晰
  - 确认评分交互不会破坏现有社区标签提交流程

### Task 3: 在列表页展示评分信号

**Files:**
- Modify: `skill-home-web/src/components/search/SkillResultList.tsx`
- Modify: `skill-home-web/src/components/search/SkillResultList.test.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: 写失败测试**
  - 在 `SkillResultList.test.tsx` 中新增断言：
    - 结果项会显示均分和评分人数
    - 无评分 skill 会显示“暂无评分”之类的弱提示

- [ ] **Step 2: 运行测试确认失败**

Run: `npm test -- --run src/components/search/SkillResultList.test.tsx`
Expected: FAIL，因为当前结果项不展示评分信号。

- [ ] **Step 3: 做最小实现**
  - 在 `SkillResultList.tsx` 中为卡片视图和列表视图补评分显示
  - 保持官方 tags 筛选逻辑不变
  - 控制结果项信息密度，不引入社区 tags 主展示

- [ ] **Step 4: 重新运行测试**

Run: `npm test -- --run src/components/search/SkillResultList.test.tsx`
Expected: PASS

- [ ] **Step 5: 自查重构**
  - 确认 `rating` 排序时结果项信息具备解释性
  - 确认零评分和少量评分文案不过度夸大

### Task 4: 运行回归验证

**Files:**
- Modify: `skill-home-web/src/api.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- Modify: `skill-home-web/src/pages/skill/SkillOverviewPage.test.tsx`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- Modify: `skill-home-web/src/components/search/SkillResultList.tsx`
- Modify: `skill-home-web/src/components/search/SkillResultList.test.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: 运行目标测试集**

Run:
- `npm test -- --run src/hooks/useRegistryApp.test.tsx`
- `npm test -- --run src/pages/skill/SkillOverviewPage.test.tsx`
- `npm test -- --run src/components/search/SkillResultList.test.tsx`

Expected: PASS

- [ ] **Step 2: 运行完整 Web 测试**

Run: `npm test`
Expected: PASS

- [ ] **Step 3: 运行构建验证**

Run: `npm run build`
Expected: PASS

- [ ] **Step 4: 人工验收要点**
  - 登录状态进入详情页，能够提交社区标签和评分
  - 未登录状态点击评分时会跳登录
  - 列表页能看到均分与评分人数
  - `高评分` 排序结果和列表展示一致
