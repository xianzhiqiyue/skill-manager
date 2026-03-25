import { startTransition, useEffect, useState, type FormEvent, type ReactNode } from 'react';

import { API_BASE, getDownloadUrl, type SkillDetail, type SkillSummary } from './api';
import { useRegistryApp } from './hooks/useRegistryApp';
import { useRoute } from './hooks/useRoute';
import {
  copyText,
  formatBytes,
  formatDate,
  formatDateTime,
  getInstallRecipes,
  skillKey,
  skillRef,
  summarizeScanStatus,
  type InstallIDE,
} from './lib/format';
import { buildSkillPath, routeMatches } from './lib/routes';

type AppModel = ReturnType<typeof useRegistryApp>;

type ToastTone = 'success' | 'danger' | 'warning' | 'neutral';

const homeStartSteps = [
  {
    step: '01',
    title: '进入技能中心',
    body: '先按关键词、命名空间、标签和 License 缩小范围，再决定查看哪个 skill。',
  },
  {
    step: '02',
    title: '查看详情与版本',
    body: '详情页负责展示扫描状态、版本、包大小、更新时间和安装入口，不再在列表里重复预览。',
  },
  {
    step: '03',
    title: '一键复制安装',
    body: '按目标 AI 复制 pull、install 或下载链接，把发现动作直接转成安装动作。',
  },
];

const faqItems = [
  {
    title: '为什么需要网页入口，而不是只用 README？',
    body: 'README 适合少量核心用户，网页入口适合新人、跨团队协作和外部宣传。搜索、筛选、安装和版本理解都更直接。',
  },
  {
    title: '技能包现在用什么格式？',
    body: '当前统一使用 ZIP。网页下载、CLI 打包和服务端下载默认都已经切换为 ZIP。',
  },
  {
    title: '哪些 AI/IDE 能直接安装？',
    body: '当前前端已提供 Codex、Claude Code、Cursor、GitHub Copilot 四类安装指引，命令由 skill-home CLI 执行。',
  },
];

function dispatchToast(detail: { tone: ToastTone; message: string }) {
  if (typeof window === 'undefined') {
    return;
  }

  window.dispatchEvent(new CustomEvent('skill-home-toast', { detail }));
}

function InternalButton({
  active = false,
  className = '',
  label,
  onClick,
}: {
  active?: boolean;
  className?: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className={`${className} ${active ? 'is-active' : ''}`.trim()}
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}

function StatusBanner({
  tone,
  message,
}: {
  tone: 'success' | 'danger' | 'warning' | 'neutral';
  message: string;
}) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

function MetricCard({
  label,
  value,
  helper,
}: {
  label: string;
  value: string;
  helper?: string;
}) {
  return (
    <article className="metric-card">
      <span className="metric-card__label">{label}</span>
      <strong className="metric-card__value">{value}</strong>
      {helper ? <span className="metric-card__helper">{helper}</span> : null}
    </article>
  );
}

function CopyButton({
  label,
  value,
  className = '',
}: {
  label: string;
  value: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await copyText(value);
      setCopied(true);
      dispatchToast({
        tone: 'success',
        message: label === '复制'
          ? '内容已复制到剪贴板。'
          : `${label.replace(/^复制/, '').trim() || '内容'}已复制。`,
      });
      window.setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
      dispatchToast({
        tone: 'warning',
        message: '复制失败，请手动复制当前内容。',
      });
    }
  }

  return (
    <button className={className} onClick={handleCopy} type="button">
      {copied ? '已复制' : label}
    </button>
  );
}

function CommandCard({
  title,
  description,
  value,
}: {
  title: string;
  description: string;
  value: string;
}) {
  return (
    <article className="command-card">
      <div className="command-card__top">
        <div>
          <strong>{title}</strong>
          <p>{description}</p>
        </div>
        <CopyButton className="button button--quiet" label="复制" value={value} />
      </div>
      <code>{value}</code>
    </article>
  );
}

function SkillBadge({ value }: { value: string }) {
  return <span className="skill-badge">{value}</span>;
}

function SectionHeading({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="section-heading">
      <div>
        <span className="eyebrow">{eyebrow}</span>
        <h2>{title}</h2>
        {description ? <p>{description}</p> : null}
      </div>
      {action ? <div className="section-heading__action">{action}</div> : null}
    </div>
  );
}

function CompactStat({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="compact-stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function DetailFact({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="detail-fact">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function collectTopTags(skills: SkillSummary[]) {
  const counts = new Map<string, number>();

  skills.forEach((skill) => {
    (skill.tags || []).forEach((tag) => {
      counts.set(tag, (counts.get(tag) || 0) + 1);
    });
  });

  return Array.from(counts.entries())
    .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0], 'zh-CN'))
    .slice(0, 8)
    .map(([tag]) => tag);
}

function buildActiveFilterLabels(model: AppModel) {
  const labels: string[] = [];

  if (model.catalogFilters.query.trim()) {
    labels.push(`关键词：${model.catalogFilters.query.trim()}`);
  }

  if (model.catalogFilters.namespace !== 'all') {
    labels.push(`命名空间：@${model.catalogFilters.namespace}`);
  }

  if (model.catalogFilters.tag !== 'all') {
    labels.push(`标签：${model.catalogFilters.tag}`);
  }

  if (model.catalogFilters.license !== 'all') {
    labels.push(`License：${model.catalogFilters.license}`);
  }

  return labels;
}

function SkillSummaryCard({
  skill,
  tone = 'default',
  onOpen,
}: {
  skill: SkillSummary;
  tone?: 'default' | 'accent';
  onOpen: () => void;
}) {
  return (
    <article className={`skill-summary-card skill-summary-card--${tone}`}>
      <div className="skill-summary-card__top">
        <span className="skill-summary-card__ref">{skillRef(skill)}</span>
        <div className="chip-row">
          <span className="skill-summary-card__version">
            {skill.latest_version || 'draft'}
          </span>
          {skill.is_deprecated ? <span className="status-pill status-pill--warning">已弃用</span> : null}
        </div>
      </div>
      <h3>{skill.name}</h3>
      <p>{skill.description || '暂无描述。'}</p>
      <div className="skill-summary-card__meta">
        <span>{skill.download_count} 下载</span>
        <span>{skill.rating_count} 评分</span>
        <span>{formatDate(skill.updated_at)}</span>
      </div>
      {(skill.tags || []).length ? (
        <div className="skill-tag-row">
          {(skill.tags || []).slice(0, 3).map((tag) => (
            <SkillBadge key={tag} value={tag} />
          ))}
        </div>
      ) : null}
      <div className="skill-summary-card__actions">
        <button className="button button--secondary" onClick={onOpen} type="button">
          查看详情
        </button>
        <CopyButton
          className="button button--quiet"
          label="复制引用"
          value={skillRef(skill)}
        />
      </div>
    </article>
  );
}

function CatalogCard({
  skill,
  onOpen,
}: {
  skill: SkillSummary;
  onOpen: () => void;
}) {
  return (
    <article className="catalog-card">
      <div className="catalog-card__header">
        <div>
          <strong>{skill.name}</strong>
          <span>{skillRef(skill)}</span>
        </div>
        <div className="chip-row">
          <span className="catalog-chip">{skill.latest_version || 'draft'}</span>
          {skill.is_deprecated ? <span className="status-pill status-pill--warning">已弃用</span> : null}
        </div>
      </div>
      <p>{skill.description || '暂无描述。'}</p>
      <div className="catalog-card__facts">
        <span>{skill.download_count} 下载</span>
        <span>{skill.rating_count} 评分</span>
        <span>{skill.license || '未填写 License'}</span>
      </div>
      {(skill.tags || []).length ? (
        <div className="skill-tag-row">
          {(skill.tags || []).slice(0, 4).map((tag) => (
            <SkillBadge key={tag} value={tag} />
          ))}
        </div>
      ) : null}
      <div className="catalog-card__actions">
        <button className="button button--secondary" onClick={onOpen} type="button">
          详情与安装
        </button>
        <CopyButton
          className="button button--quiet"
          label="复制引用"
          value={skillRef(skill)}
        />
      </div>
    </article>
  );
}

function CatalogRow({
  skill,
  onOpen,
}: {
  skill: SkillSummary;
  onOpen: () => void;
}) {
  return (
    <article className="catalog-row">
      <div className="catalog-row__main">
        <div className="catalog-row__title">
          <strong>{skill.name}</strong>
          <div className="chip-row">
            <span className="catalog-chip">{skill.latest_version || 'draft'}</span>
            {skill.is_deprecated ? <span className="status-pill status-pill--warning">已弃用</span> : null}
          </div>
        </div>
        <span>{skillRef(skill)}</span>
        <p>{skill.description || '暂无描述。'}</p>
        {(skill.tags || []).length ? (
          <div className="skill-tag-row skill-tag-row--dense">
            {(skill.tags || []).slice(0, 4).map((tag) => (
              <SkillBadge key={tag} value={tag} />
            ))}
          </div>
        ) : null}
      </div>
      <div className="catalog-row__facts">
        <div className="catalog-row__fact">
          <span>下载</span>
          <strong>{skill.download_count}</strong>
        </div>
        <div className="catalog-row__fact">
          <span>评分</span>
          <strong>{skill.rating_count}</strong>
        </div>
        <div className="catalog-row__fact">
          <span>License</span>
          <strong>{skill.license || '未填写'}</strong>
        </div>
        <div className="catalog-row__fact">
          <span>更新</span>
          <strong>{formatDate(skill.updated_at)}</strong>
        </div>
      </div>
      <div className="catalog-row__action">
        <button
          className="button button--secondary"
          onClick={onOpen}
          type="button"
        >
          查看详情
        </button>
        <CopyButton
          className="button button--quiet"
          label="复制引用"
          value={skillRef(skill)}
        />
      </div>
    </article>
  );
}

function QuickActionCard({
  eyebrow,
  title,
  body,
  primaryLabel,
  onPrimary,
  secondaryLabel,
  onSecondary,
}: {
  eyebrow: string;
  title: string;
  body: string;
  primaryLabel: string;
  onPrimary: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
}) {
  return (
    <article className="quick-action-card">
      <span className="eyebrow">{eyebrow}</span>
      <h3>{title}</h3>
      <p>{body}</p>
      <div className="quick-action-card__actions">
        <button className="button button--primary" onClick={onPrimary} type="button">
          {primaryLabel}
        </button>
        {secondaryLabel && onSecondary ? (
          <button className="button button--quiet" onClick={onSecondary} type="button">
            {secondaryLabel}
          </button>
        ) : null}
      </div>
    </article>
  );
}

function SpotlightSkillCard({
  skill,
  onOpen,
}: {
  skill: SkillSummary;
  onOpen: () => void;
}) {
  return (
    <article className="spotlight-skill-card">
      <div className="spotlight-skill-card__top">
        <span className="eyebrow">Top skill</span>
        <span className="catalog-chip">{skill.latest_version || 'draft'}</span>
      </div>
      <h3>{skill.name}</h3>
      <p className="spotlight-skill-card__ref">{skillRef(skill)}</p>
      <p>{skill.description || '暂无描述。'}</p>
      <div className="spotlight-skill-card__facts">
        <CompactStat label="下载" value={String(skill.download_count)} />
        <CompactStat label="评分" value={String(skill.rating_count)} />
        <CompactStat label="更新" value={formatDate(skill.updated_at)} />
      </div>
      {(skill.tags || []).length ? (
        <div className="skill-tag-row">
          {(skill.tags || []).slice(0, 4).map((tag) => (
            <SkillBadge key={tag} value={tag} />
          ))}
        </div>
      ) : null}
      <div className="spotlight-skill-card__actions">
        <button className="button button--primary" onClick={onOpen} type="button">
          查看详情与安装
        </button>
        <CopyButton
          className="button button--quiet"
          label="复制引用"
          value={skillRef(skill)}
        />
      </div>
    </article>
  );
}

function CompactSkillLink({
  skill,
  onOpen,
}: {
  skill: SkillSummary;
  onOpen: () => void;
}) {
  return (
    <button className="compact-skill-link" onClick={onOpen} type="button">
      <div>
        <strong>{skill.name}</strong>
        <span>{skillRef(skill)}</span>
      </div>
      <span>{formatDate(skill.updated_at)}</span>
    </button>
  );
}

function InstallGuide({
  skill,
  title = 'AI 安装指引',
}: {
  skill: SkillSummary | SkillDetail;
  title?: string;
}) {
  const recipes = getInstallRecipes(skill);
  const [activeIDE, setActiveIDE] = useState<InstallIDE>('codex');
  const recipe = recipes.find((item) => item.id === activeIDE) || recipes[0];

  return (
    <section className="install-guide">
      <SectionHeading
        eyebrow="AI install"
        title={title}
        description="把技能发现动作直接转成可复制的命令，安装前先用 doctor 做环境检查。"
      />

      <div className="install-guide__tabs">
        {recipes.map((item) => (
          <InternalButton
            key={item.id}
            active={item.id === recipe.id}
            className="segmented-button"
            label={item.label}
            onClick={() => setActiveIDE(item.id)}
          />
        ))}
      </div>

      <div className="install-guide__hero">
        <div>
          <h3>{recipe.label}</h3>
          <p>{recipe.description}</p>
          <code className="install-guide__ref">{recipe.reference}</code>
        </div>
        <div className="install-guide__hero-actions">
          <CopyButton
            className="button button--secondary"
            label="复制引用名"
            value={recipe.reference}
          />
          <CopyButton
            className="button button--quiet"
            label="复制下载链接"
            value={recipe.download}
          />
        </div>
      </div>

      <div className="install-guide__primary-command">
        <div className="install-guide__primary-command-top">
          <div>
            <span className="eyebrow">Primary command</span>
            <h3>直接安装命令</h3>
            <p>默认给出可直接同步到 {recipe.label} 的安装命令，适合作为首选动作。</p>
          </div>
          <div className="install-guide__primary-command-actions">
            <CopyButton
              className="button button--primary"
              label="复制安装命令"
              value={recipe.install}
            />
            <CopyButton
              className="button button--quiet"
              label="复制 pull 命令"
              value={recipe.pull}
            />
          </div>
        </div>
        <code>{recipe.install}</code>
      </div>

      <div className="install-guide__grid">
        <CommandCard
          title="步骤 1：环境检查"
          description="确认本机已安装 CLI，且 registry 与 IDE 路径可用。"
          value={recipe.verify}
        />
        <CommandCard
          title="步骤 2：搜索技能"
          description="先在 registry 中确认技能名称和版本。"
          value={recipe.search}
        />
        <CommandCard
          title="步骤 3：拉取技能"
          description="把技能包拉到本地缓存，便于后续检查和安装。"
          value={recipe.pull}
        />
      </div>

      <div className="install-guide__note">
        <strong>复制后下一步</strong>
        <p>
          先执行环境检查，再执行安装命令。如果你在 WSL + Windows 混合环境中，保留
          `--mode mirror` 可避免符号链接兼容问题。
        </p>
      </div>
    </section>
  );
}

function HomePage({
  model,
  navigate,
}: {
  model: AppModel;
  navigate: (path: string) => void;
}) {
  const spotlightSkill = model.featuredSkills[0];
  const trendingTags = collectTopTags(model.skills).slice(0, 6);

  return (
    <div className="page-stack">
      <section className="hero-panel">
        <div className="hero-panel__copy">
          <span className="eyebrow">Skill Home Registry</span>
          <h1>把 AI Skill 做成可搜索、可安装、可持续运营的产品体验</h1>
          <p>
            不再给团队丢一个 README 路径，而是给他们一个结构清晰的技能中心、可信的详情页
            和可复制的安装指引。Skill Home 负责把 Registry、CLI 和 Web 接成一个完整闭环。
          </p>
          <div className="hero-panel__actions">
            <button className="button button--primary" onClick={() => navigate('/skills')} type="button">
              进入技能中心
            </button>
            <button className="button button--secondary" onClick={() => navigate('/publish')} type="button">
              发布新技能
            </button>
          </div>
        </div>

        <div className="hero-panel__metrics">
          <MetricCard label="Registry 状态" value={model.health?.status || '检查中'} />
          <MetricCard label="公开技能数" value={String(model.skillsTotal)} />
          <MetricCard label="命名空间" value={String(model.quickStats.namespaceCount)} />
          <MetricCard
            label="API 入口"
            value={API_BASE.replace(/^https?:\/\//, '')}
            helper={model.healthError || model.health?.version || '读取中'}
          />
        </div>
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="Quick access"
          title="先走最短动作，不先读整页说明"
          description="首页只负责把人快速送进搜索、安装和发布三条主路径。"
        />
        <div className="quick-action-grid">
          <QuickActionCard
            eyebrow="Search"
            title="先找对 skill"
            body="进入技能中心按关键词、命名空间和标签筛选，直接缩小到候选 skill。"
            primaryLabel="打开技能中心"
            onPrimary={() => navigate('/skills')}
            secondaryLabel="查看热门技能"
            onSecondary={() => navigate('/skills')}
          />
          <QuickActionCard
            eyebrow="Install"
            title="直接复制安装命令"
            body="安装页会优先展示主安装命令，再补 doctor、search 和 pull 这些辅助命令。"
            primaryLabel="打开安装页"
            onPrimary={() => navigate('/install')}
            secondaryLabel="查看安装指南"
            onSecondary={() => navigate('/install')}
          />
          <QuickActionCard
            eyebrow="Publish"
            title="发布并维护 skill"
            body="登录后上传 ZIP、生成版本并回到控制台维护元信息和版本记录。"
            primaryLabel={model.token ? '进入控制台' : '前往发布'}
            onPrimary={() => navigate(model.token ? '/console' : '/publish')}
            secondaryLabel={model.token ? '发布新版本' : '登录账号'}
            onSecondary={() => navigate(model.token ? '/publish' : '/login')}
          />
        </div>
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="Recommended"
          title="先从最值得看的技能开始"
          description="首页优先给出一个重点 skill，再把最近更新和高频主题收在右侧。"
          action={
            <button className="button button--quiet" onClick={() => navigate('/skills')} type="button">
              查看全部技能
            </button>
          }
        />
        <div className="home-discovery-layout">
          {spotlightSkill ? (
            <SpotlightSkillCard
              onOpen={() => navigate(buildSkillPath(spotlightSkill.namespace, spotlightSkill.name))}
              skill={spotlightSkill}
            />
          ) : (
            <div className="empty-panel">当前还没有可展示的公开技能。</div>
          )}

          <div className="home-discovery-side">
            <article className="info-card">
              <span className="eyebrow">Latest</span>
              <h3>最近更新</h3>
              <p>先看最近在维护的技能，避免把时间浪费在陈旧版本上。</p>
              <div className="compact-skill-list">
                {model.latestSkills.length ? (
                  model.latestSkills.slice(0, 4).map((skill) => (
                    <CompactSkillLink
                      key={skillKey(skill)}
                      onOpen={() => navigate(buildSkillPath(skill.namespace, skill.name))}
                      skill={skill}
                    />
                  ))
                ) : (
                  <div className="empty-panel empty-panel--compact">等待技能目录加载后展示最近更新。</div>
                )}
              </div>
            </article>

            <article className="info-card">
              <span className="eyebrow">Topics</span>
              <h3>高频主题</h3>
              <p>按常见标签进入搜索，通常比从零开始浏览更快。</p>
              <div className="filter-chip-row">
                {trendingTags.length ? (
                  trendingTags.map((tag) => (
                    <button
                      className="filter-chip-button"
                      key={tag}
                      onClick={() => {
                        model.updateCatalogFilter('tag', tag);
                        navigate('/skills');
                      }}
                      type="button"
                    >
                      {tag}
                    </button>
                  ))
                ) : (
                  <span className="active-filter-chip">等待目录加载标签</span>
                )}
              </div>
            </article>
          </div>
        </div>
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="Get started"
          title="第一次使用的最短路径"
          description="把完整工作流收成三步，避免首页再次变成说明书。"
          action={
            <button className="button button--secondary" onClick={() => navigate('/install')} type="button">
              查看安装指南
            </button>
          }
        />
        <div className="two-column-layout home-start-layout">
          <div className="home-path-list">
            {homeStartSteps.map((item) => (
              <div className="home-path-item" key={item.step}>
                <span className="workflow-card__step">{item.step}</span>
                <div>
                  <h3>{item.title}</h3>
                  <p>{item.body}</p>
                </div>
              </div>
            ))}
            <div className="home-path-actions">
              <button className="button button--primary" onClick={() => navigate('/skills')} type="button">
                现在去找技能
              </button>
              <button className="button button--quiet" onClick={() => navigate('/install')} type="button">
                先看安装页
              </button>
            </div>
          </div>
          <aside className="side-notes">
            <article className="info-card">
              <span className="eyebrow">Operator note</span>
              <h3>第一次装不上的常见原因</h3>
              <p>多数失败并不是 skill 本身有问题，而是 CLI、registry 或 IDE 路径还没打通。</p>
              <div className="home-skill-list">
                <div className="home-skill-list__item">
                  <strong>先跑 doctor</strong>
                  <span>`skill-home doctor` 可以先确认 registry、认证和 IDE 路径。</span>
                </div>
                <div className="home-skill-list__item">
                  <strong>先确定 skill 版本</strong>
                  <span>别直接复制旧命令，先去详情页确认最新版本和扫描状态。</span>
                </div>
                <div className="home-skill-list__item">
                  <strong>WSL 环境保留 mirror</strong>
                  <span>混合环境建议保留 `--mode mirror`，避免符号链接兼容问题。</span>
                </div>
              </div>
            </article>
          </aside>
        </div>
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="FAQ"
          title="使用 Skill Home 前最常见的几个问题"
        />
        <div className="faq-list faq-list--compact">
          {faqItems.map((item) => (
            <article className="faq-item" key={item.title}>
              <h3>{item.title}</h3>
              <p>{item.body}</p>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}

function SkillsPage({
  model,
}: {
  model: AppModel;
}) {
  const [filtersOpen, setFiltersOpen] = useState(false);
  const activeFilterLabels = buildActiveFilterLabels(model);
  const topTags = collectTopTags(model.skills);

  return (
    <div className="page-stack">
      <section className="surface-panel">
        <SectionHeading
          eyebrow="Skill center"
          title="技能中心"
          description="搜索、筛选并直接进入详情页，列表页不再承载重复预览。"
        />

        <div className="catalog-workbench">
          <aside className={`filter-panel ${filtersOpen ? 'is-open' : ''}`}>
            <label className="field">
              <span>关键词</span>
              <input
                value={model.catalogFilters.query}
                onChange={(event) => model.setCatalogQuery(event.target.value)}
                placeholder="搜索名字、用途、场景"
              />
            </label>

            <label className="field">
              <span>命名空间</span>
              <select
                value={model.catalogFilters.namespace}
                onChange={(event) =>
                  model.updateCatalogFilter('namespace', event.target.value)
                }
              >
                <option value="all">全部</option>
                {model.namespaceOptions.map((namespace) => (
                  <option key={namespace} value={namespace}>
                    @{namespace}
                  </option>
                ))}
              </select>
            </label>

            <label className="field">
              <span>标签</span>
              <select
                value={model.catalogFilters.tag}
                onChange={(event) => model.updateCatalogFilter('tag', event.target.value)}
              >
                <option value="all">全部</option>
                {model.tagOptions.map((tag) => (
                  <option key={tag} value={tag}>
                    {tag}
                  </option>
                ))}
              </select>
            </label>

            {topTags.length ? (
              <div className="filter-panel__group">
                <strong>常用标签</strong>
                <div className="filter-chip-row">
                  {topTags.map((tag) => (
                    <button
                      className={`filter-chip-button ${
                        model.catalogFilters.tag === tag ? 'is-active' : ''
                      }`.trim()}
                      key={tag}
                      onClick={() =>
                        model.updateCatalogFilter(
                          'tag',
                          model.catalogFilters.tag === tag ? 'all' : tag,
                        )
                      }
                      type="button"
                    >
                      {tag}
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            <label className="field">
              <span>License</span>
              <select
                value={model.catalogFilters.license}
                onChange={(event) =>
                  model.updateCatalogFilter('license', event.target.value)
                }
              >
                <option value="all">全部</option>
                {model.licenseOptions.map((license) => (
                  <option key={license} value={license}>
                    {license}
                  </option>
                ))}
              </select>
            </label>

            <div className="filter-panel__group">
              <strong>排序</strong>
              <div className="segmented-group">
                <InternalButton
                  active={model.catalogFilters.sort === 'downloads'}
                  className="segmented-button"
                  label="热门"
                  onClick={() => model.updateCatalogFilter('sort', 'downloads')}
                />
                <InternalButton
                  active={model.catalogFilters.sort === 'updated'}
                  className="segmented-button"
                  label="最新"
                  onClick={() => model.updateCatalogFilter('sort', 'updated')}
                />
                <InternalButton
                  active={model.catalogFilters.sort === 'rating'}
                  className="segmented-button"
                  label="高评分"
                  onClick={() => model.updateCatalogFilter('sort', 'rating')}
                />
                <InternalButton
                  active={model.catalogFilters.sort === 'name'}
                  className="segmented-button"
                  label="名称"
                  onClick={() => model.updateCatalogFilter('sort', 'name')}
                />
              </div>
            </div>

            <div className="filter-panel__group">
              <strong>视图</strong>
              <div className="segmented-group">
                <InternalButton
                  active={model.catalogFilters.view === 'cards'}
                  className="segmented-button"
                  label="卡片"
                  onClick={() => model.updateCatalogFilter('view', 'cards')}
                />
                <InternalButton
                  active={model.catalogFilters.view === 'list'}
                  className="segmented-button"
                  label="列表"
                  onClick={() => model.updateCatalogFilter('view', 'list')}
                />
              </div>
            </div>

            <div className="filter-panel__summary">
              <CompactStat label="在线技能" value={String(model.skillsTotal)} />
              <CompactStat label="命名空间" value={String(model.quickStats.namespaceCount)} />
              <CompactStat label="标签维度" value={String(model.quickStats.tagCount)} />
              <CompactStat label="License" value={String(model.quickStats.licenseCount)} />
            </div>

            <button className="button button--quiet" onClick={model.resetCatalogFilters} type="button">
              重置筛选
            </button>
          </aside>

          <div className="catalog-results">
            <div className="catalog-results__toolbar">
              <div>
                <strong>{model.skillsTotal}</strong>
                <span>个公开技能</span>
              </div>
              <div className="toolbar-actions">
                <button
                  className="button button--quiet catalog-results__filters-button"
                  onClick={() => setFiltersOpen((value) => !value)}
                  type="button"
                >
                  {filtersOpen ? '收起筛选' : '筛选'}
                </button>
                <button className="button button--quiet" onClick={model.refreshCatalog} type="button">
                  刷新目录
                </button>
              </div>
            </div>

            {activeFilterLabels.length ? (
              <div className="catalog-active-filters">
                {activeFilterLabels.map((label) => (
                  <span className="active-filter-chip" key={label}>
                    {label}
                  </span>
                ))}
                <button
                  className="button button--quiet"
                  onClick={model.resetCatalogFilters}
                  type="button"
                >
                  清除筛选
                </button>
              </div>
            ) : null}

            {model.skillsLoading ? (
              <div className="empty-panel">正在读取技能目录...</div>
            ) : model.skillsError ? (
              <div className="empty-panel empty-panel--danger">
                读取技能目录失败：{model.skillsError}
              </div>
            ) : !model.skills.length ? (
              <div className="empty-panel">没有找到匹配的 skill，换个关键词或筛选条件试试。</div>
            ) : (
              <div className="catalog-results__content">
                <div
                  className={
                    model.catalogFilters.view === 'cards'
                      ? 'catalog-card-grid'
                      : 'catalog-list-table'
                  }
                >
                  {model.catalogFilters.view === 'cards' ? (
                    model.skills.map((skill) => (
                      <CatalogCard
                        key={skillKey(skill)}
                        onOpen={() => model.openSkill(skill.namespace, skill.name)}
                        skill={skill}
                      />
                    ))
                  ) : (
                    model.skills.map((skill) => (
                      <CatalogRow
                        key={skillKey(skill)}
                        onOpen={() => model.openSkill(skill.namespace, skill.name)}
                        skill={skill}
                      />
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function SkillDetailPage({
  model,
  navigate,
}: {
  model: AppModel;
  navigate: (path: string) => void;
}) {
  if (model.detailLoading) {
    return <div className="empty-panel">正在读取技能详情...</div>;
  }

  if (model.detailError) {
    return <div className="empty-panel empty-panel--danger">读取详情失败：{model.detailError}</div>;
  }

  if (!model.detailSkill) {
    return <div className="empty-panel">没有找到对应的技能。</div>;
  }

  const skill = model.detailSkill;
  const latestVersion = skill.versions?.[0];
  const latestScan = summarizeScanStatus(latestVersion?.scan_status);

  return (
    <div className="page-stack">
      <div className="breadcrumb-row">
        <button className="breadcrumb-link" onClick={() => navigate('/')} type="button">
          首页
        </button>
        <span>/</span>
        <button className="breadcrumb-link" onClick={model.returnToCatalog} type="button">
          技能中心
        </button>
        <span>/</span>
        <span>{skill.name}</span>
      </div>

      <section className="surface-panel">
        <div className="detail-hero">
          <div className="detail-hero__summary">
            <div className="detail-hero__top">
              <span className="eyebrow">Skill detail</span>
              <div className="chip-row">
                <span className={`status-pill status-pill--${latestScan.tone}`}>
                  {latestScan.label}
                </span>
                {skill.is_deprecated ? <span className="status-pill status-pill--warning">已弃用</span> : null}
              </div>
            </div>
            <h1>{skill.name}</h1>
            <p className="detail-hero__ref">{skillRef(skill)}</p>
            <p className="detail-hero__description">{skill.description || '暂无描述。'}</p>
            {skill.is_deprecated ? (
              <StatusBanner
                message="这个 skill 已被作者显式标记为弃用。历史版本仍可安装，但新接入前建议先确认替代方案。"
                tone="warning"
              />
            ) : null}
            {(skill.tags || []).length ? (
              <div className="skill-tag-row">
                {(skill.tags || []).map((tag) => (
                  <SkillBadge key={tag} value={tag} />
                ))}
              </div>
            ) : null}
          </div>

          <aside className="detail-hero__aside">
            <div className="detail-hero__actions">
              <a
                className="button button--primary"
                href={getDownloadUrl(skill)}
                rel="noreferrer"
                target="_blank"
              >
                下载 ZIP
              </a>
              <CopyButton
                className="button button--secondary"
                label="复制引用"
                value={skillRef(skill)}
              />
              <CopyButton
                className="button button--quiet"
                label="复制下载链接"
                value={getDownloadUrl(skill)}
              />
            </div>

            <div className="detail-fact-list">
              <DetailFact label="最新版本" value={skill.latest_version || 'draft'} />
              <DetailFact label="下载量" value={String(skill.download_count)} />
              <DetailFact label="版本数" value={String(skill.versions?.length || 0)} />
              <DetailFact label="更新时间" value={formatDateTime(skill.updated_at)} />
              <DetailFact label="License" value={skill.license || '未填写'} />
              <DetailFact label="状态" value={skill.is_deprecated ? '已弃用' : '活跃'} />
              <DetailFact label="可见性" value={skill.is_public === false ? '私有' : '公开'} />
            </div>
          </aside>
        </div>
      </section>

      <section className="surface-panel">
        <InstallGuide skill={skill} />
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="Versions"
          title="版本与扫描记录"
          description="版本列表用于确认安装目标、文件大小和扫描状态。"
        />
        {skill.versions?.length ? (
          <div className="version-table">
            <div className="version-table__header">
              <span>版本</span>
              <span>发布时间</span>
              <span>包大小</span>
              <span>扫描状态</span>
              <span>操作</span>
            </div>
            {skill.versions.map((version) => {
              const state = summarizeScanStatus(version.scan_status);
              return (
                <div className="version-table__row" key={version.id}>
                  <strong>{version.version}</strong>
                  <span>{formatDateTime(version.published_at || version.created_at)}</span>
                  <span>{formatBytes(version.size_bytes)}</span>
                  <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span>
                  <CopyButton
                    className="button button--quiet"
                    label="复制 pull"
                    value={`skill-home pull ${skillRef(skill)}@${version.version}`}
                  />
                </div>
              );
            })}
          </div>
        ) : (
          <div className="empty-panel">这个 skill 还没有公开的版本记录。</div>
        )}
      </section>

      <section className="surface-panel">
        <SectionHeading
          eyebrow="Related"
          title="继续探索相关技能"
          description="优先展示同命名空间或同标签的技能，避免在当前页堆太多安装说明。"
        />
        <div className="info-grid info-grid--3">
          {model.relatedSkills.length ? (
            model.relatedSkills.map((item) => (
              <SkillSummaryCard
                key={skillKey(item)}
                onOpen={() => navigate(buildSkillPath(item.namespace, item.name))}
                skill={item}
                tone="default"
              />
            ))
          ) : (
            <div className="empty-panel">还没有找到相关技能。</div>
          )}
        </div>
      </section>
    </div>
  );
}

function PublishPage({
  model,
  navigate,
}: {
  model: AppModel;
  navigate: (path: string) => void;
}) {
  if (!model.token) {
    return (
      <div className="page-stack">
        <section className="surface-panel auth-callout">
          <SectionHeading
            eyebrow="Publish"
            title="先登录，再发布你的 skill"
            description="发布页聚焦单一任务：上传 ZIP、填写元信息、生成发布结果和下一步动作。"
          />
          <div className="auth-callout__actions">
            <button className="button button--primary" onClick={() => navigate('/login')} type="button">
              前往登录
            </button>
            <button className="button button--secondary" onClick={() => navigate('/register')} type="button">
              注册账号
            </button>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <section className="surface-panel">
        <SectionHeading
          eyebrow="Publish"
          title="发布新的 skill 版本"
          description="上传 ZIP 包、填写元信息，服务端会完成安全扫描并生成版本记录。"
        />

        <div className="two-column-layout">
          <div className="side-notes">
            <article className="info-card">
              <h3>发布前检查</h3>
              <p>建议先在本地执行 `skill-home validate` 和 `skill-home pack`，确保 SKILL.md 和 scripts 结构完整。</p>
            </article>
            <article className="info-card">
              <h3>推荐流程</h3>
              <p>先打包 ZIP，再上传到 registry；发布后去详情页检查版本、扫描状态和安装指引是否正常。</p>
            </article>
          </div>

          <div className="form-panel">
            {model.publishError ? (
              <StatusBanner message={model.publishError} tone="danger" />
            ) : null}
            {model.publishSuccess ? (
              <StatusBanner
                message={`发布成功：${model.publishSuccess.namespace}/${model.publishSuccess.name}@${model.publishSuccess.version}`}
                tone="success"
              />
            ) : null}

            <form
              className="form-grid-stack"
              onSubmit={(event) => {
                event.preventDefault();
                void model.submitPublish();
              }}
            >
              <div className="form-grid-two">
                <label className="field">
                  <span>命名空间</span>
                  <input
                    required
                    value={model.publishForm.namespace}
                    onChange={(event) =>
                      model.setPublishForm((current) => ({
                        ...current,
                        namespace: event.target.value,
                      }))
                    }
                    placeholder="例如 testuser"
                  />
                </label>

                <label className="field">
                  <span>版本号</span>
                  <input
                    required
                    value={model.publishForm.version}
                    onChange={(event) =>
                      model.setPublishForm((current) => ({
                        ...current,
                        version: event.target.value,
                      }))
                    }
                    placeholder="0.1.0"
                  />
                </label>
              </div>

              <label className="field">
                <span>技能名</span>
                <input
                  required
                  value={model.publishForm.name}
                  onChange={(event) =>
                    model.setPublishForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                  placeholder="例如 skill-home-manager"
                />
              </label>

              <label className="field">
                <span>描述</span>
                <textarea
                  required
                  rows={5}
                  value={model.publishForm.description}
                  onChange={(event) =>
                    model.setPublishForm((current) => ({
                      ...current,
                      description: event.target.value,
                    }))
                  }
                  placeholder="简要说明这个 skill 解决什么问题"
                />
              </label>

              <div className="form-grid-two">
                <label className="field">
                  <span>License</span>
                  <input
                    value={model.publishForm.license}
                    onChange={(event) =>
                      model.setPublishForm((current) => ({
                        ...current,
                        license: event.target.value,
                      }))
                    }
                    placeholder="MIT"
                  />
                </label>

                <label className="field">
                  <span>技能包（ZIP）</span>
                  <input
                    accept=".zip,application/zip"
                    required
                    type="file"
                    onChange={(event) => model.setPublishFile(event.target.files?.[0] || null)}
                  />
                </label>
              </div>

              <label className="checkbox-field">
                <input
                  checked={model.publishForm.isPublic}
                  onChange={(event) =>
                    model.setPublishForm((current) => ({
                      ...current,
                      isPublic: event.target.checked,
                    }))
                  }
                  type="checkbox"
                />
                <span>发布为公开技能</span>
              </label>

              <button className="button button--primary" disabled={model.publishLoading} type="submit">
                {model.publishLoading ? '发布中...' : '上传并发布'}
              </button>
            </form>

            {model.publishSuccess ? (
              <div className="publish-result-card">
                <strong>发布完成，下一步建议</strong>
                <div className="publish-result-card__actions">
                  <button
                    className="button button--secondary"
                    onClick={() =>
                      navigate(
                        buildSkillPath(
                          model.publishSuccess!.namespace,
                          model.publishSuccess!.name,
                        ),
                      )
                    }
                    type="button"
                  >
                    打开详情页
                  </button>
                  <CopyButton
                    className="button button--quiet"
                    label="复制下载链接"
                    value={`${API_BASE}${model.publishSuccess.download_url}?format=zip`}
                  />
                </div>
              </div>
            ) : null}
          </div>
        </div>
      </section>
    </div>
  );
}

function ConsolePage({
  model,
  navigate,
}: {
  model: AppModel;
  navigate: (path: string) => void;
}) {
  if (!model.token) {
    return (
      <div className="page-stack">
        <section className="surface-panel auth-callout">
          <SectionHeading
            eyebrow="Console"
            title="登录后管理你的技能"
            description="控制台负责维护‘我的技能’、编辑元信息、删除版本和查看发布结果。"
          />
          <div className="auth-callout__actions">
            <button className="button button--primary" onClick={() => navigate('/login')} type="button">
              登录
            </button>
            <button className="button button--secondary" onClick={() => navigate('/register')} type="button">
              注册
            </button>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <section className="surface-panel">
        <SectionHeading
          eyebrow="Console"
          title="我的技能控制台"
          description="用列表 + 详情编辑的方式维护你的 skill，不再把管理操作塞到首页里。"
        />

        {model.accountError ? <StatusBanner message={model.accountError} tone="danger" /> : null}
        {model.manageError ? <StatusBanner message={model.manageError} tone="danger" /> : null}
        {model.manageSuccess ? <StatusBanner message={model.manageSuccess} tone="success" /> : null}

        <div className="console-layout">
          <aside className="console-sidebar">
            {model.currentUser ? (
              <div className="profile-card">
                <strong>{model.currentUser.username}</strong>
                <span>{model.currentUser.email}</span>
                <span>注册于 {formatDateTime(model.currentUser.created_at)}</span>
              </div>
            ) : null}

            <div className="info-grid info-grid--3">
              <MetricCard label="总技能数" value={String(model.accountStats.total)} />
              <MetricCard label="公开" value={String(model.accountStats.publicCount)} />
              <MetricCard label="私有" value={String(model.accountStats.privateCount)} />
            </div>

            <div className="skill-list-panel">
              <div className="skill-list-panel__header">
                <strong>我的技能</strong>
                <button className="button button--quiet" onClick={() => navigate('/publish')} type="button">
                  新建发布
                </button>
              </div>
              {model.accountLoading ? (
                <div className="empty-panel">正在读取账号技能...</div>
              ) : model.mySkills.length ? (
                model.mySkills.map((skill) => (
                  <button
                    className={`owned-skill-row ${
                      model.managedSkillKey === skillKey(skill) ? 'owned-skill-row--active' : ''
                    }`}
                    key={skillKey(skill)}
                    onClick={() => model.setManagedSkillKey(skillKey(skill))}
                    type="button"
                  >
                    <div>
                      <strong>{skill.name}</strong>
                      <span>{skillRef(skill)}</span>
                    </div>
                    <span>{skill.is_deprecated ? '已弃用' : skill.is_public === false ? '私有' : '公开'}</span>
                  </button>
                ))
              ) : (
                <div className="empty-panel">你还没有发布任何技能。</div>
              )}
            </div>
          </aside>

          <div className="console-main">
            {model.manageLoading ? (
              <div className="empty-panel">正在读取技能管理信息...</div>
            ) : model.managedSkill ? (
              <>
                <div className="console-main__header">
                  <div>
                    <span className="eyebrow">Manage skill</span>
                    <h3>{skillRef(model.managedSkill)}</h3>
                    <p>{model.managedSkill.latest_version || 'draft'}</p>
                    {model.managedSkill.is_deprecated ? (
                      <span className="status-pill status-pill--warning">已弃用</span>
                    ) : null}
                  </div>
                  <button
                    className="button button--danger"
                    disabled={model.manageDeletingSkill}
                    onClick={() => void model.removeManagedSkill()}
                    type="button"
                  >
                    {model.manageDeletingSkill ? '删除中...' : '删除 skill'}
                  </button>
                </div>

                <form
                  className="form-grid-stack"
                  onSubmit={(event) => {
                    event.preventDefault();
                    void model.submitManage();
                  }}
                >
                  <label className="field">
                    <span>描述</span>
                    <textarea
                      rows={5}
                      value={model.manageForm.description}
                      onChange={(event) =>
                        model.setManageForm((current) => ({
                          ...current,
                          description: event.target.value,
                        }))
                      }
                      placeholder="补充 skill 的用途、边界和适用场景"
                    />
                  </label>

                  <div className="form-grid-two">
                    <label className="field">
                      <span>License</span>
                      <input
                        value={model.manageForm.license}
                        onChange={(event) =>
                          model.setManageForm((current) => ({
                            ...current,
                            license: event.target.value,
                          }))
                        }
                        placeholder="MIT"
                      />
                    </label>

                    <label className="field">
                      <span>Tags</span>
                      <input
                        value={model.manageForm.tags}
                        onChange={(event) =>
                          model.setManageForm((current) => ({
                            ...current,
                            tags: event.target.value,
                          }))
                        }
                        placeholder="review, codex, registry"
                      />
                    </label>
                  </div>

                  <label className="checkbox-field">
                    <input
                      checked={model.manageForm.isPublic}
                      onChange={(event) =>
                        model.setManageForm((current) => ({
                          ...current,
                          isPublic: event.target.checked,
                        }))
                      }
                      type="checkbox"
                    />
                    <span>公开展示到在线目录</span>
                  </label>

                  <label className="checkbox-field">
                    <input
                      checked={model.manageForm.isDeprecated}
                      onChange={(event) =>
                        model.setManageForm((current) => ({
                          ...current,
                          isDeprecated: event.target.checked,
                        }))
                      }
                      type="checkbox"
                    />
                    <span>标记为弃用，但保留历史版本安装能力</span>
                  </label>

                  <button className="button button--primary" disabled={model.manageSaving} type="submit">
                    {model.manageSaving ? '保存中...' : '保存技能信息'}
                  </button>
                </form>

                <div className="version-table">
                  <div className="version-table__header">
                    <span>版本</span>
                    <span>时间</span>
                    <span>大小</span>
                    <span>状态</span>
                    <span>操作</span>
                  </div>
                  {model.managedSkill.versions?.length ? (
                    model.managedSkill.versions.map((version) => {
                      const state = summarizeScanStatus(version.scan_status);
                      return (
                        <div className="version-table__row" key={version.id}>
                          <strong>{version.version}</strong>
                          <span>{formatDateTime(version.published_at || version.created_at)}</span>
                          <span>{formatBytes(version.size_bytes)}</span>
                          <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span>
                          <button
                            className="button button--quiet button--danger-text"
                            disabled={model.manageDeletingVersion === version.version}
                            onClick={() => void model.removeManagedVersion(version.version)}
                            type="button"
                          >
                            {model.manageDeletingVersion === version.version ? '删除中...' : '删除版本'}
                          </button>
                        </div>
                      );
                    })
                  ) : (
                    <div className="empty-panel">当前没有可管理的版本。</div>
                  )}
                </div>
              </>
            ) : (
              <div className="empty-panel">从左侧选择一个技能开始管理。</div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function AuthPage({
  model,
  mode,
  navigate,
}: {
  model: AppModel;
  mode: 'login' | 'register';
  navigate: (path: string) => void;
}) {
  return (
    <div className="page-stack">
      <section className="surface-panel auth-shell">
        <div className="auth-shell__aside">
          <span className="eyebrow">Account</span>
          <h1>{mode === 'register' ? '创建账号并开始发布你的 skill' : '登录后进入技能控制台'}</h1>
          <p>
            登录后你可以直接在 Web 里发布技能、查看我的技能、编辑描述、删除版本，并复制面向 AI 的安装命令。
          </p>
        </div>

        <div className="auth-shell__form">
          {model.authError ? <StatusBanner message={model.authError} tone="danger" /> : null}
          {model.authSuccess ? <StatusBanner message={model.authSuccess} tone="success" /> : null}

          <div className="segmented-group segmented-group--wide">
            <InternalButton
              active={mode === 'login'}
              className="segmented-button"
              label="登录"
              onClick={() => navigate('/login')}
            />
            <InternalButton
              active={mode === 'register'}
              className="segmented-button"
              label="注册"
              onClick={() => navigate('/register')}
            />
          </div>

          <form
            className="form-grid-stack"
            onSubmit={(event) => {
              event.preventDefault();
              void model.submitAuth(mode);
            }}
          >
            {mode === 'register' ? (
              <label className="field">
                <span>用户名</span>
                <input
                  required
                  value={model.authForm.username}
                  onChange={(event) =>
                    model.setAuthForm((current) => ({
                      ...current,
                      username: event.target.value,
                    }))
                  }
                  placeholder="例如 testuser"
                />
              </label>
            ) : null}

            <label className="field">
              <span>邮箱</span>
              <input
                required
                type="email"
                value={model.authForm.email}
                onChange={(event) =>
                  model.setAuthForm((current) => ({
                    ...current,
                    email: event.target.value,
                  }))
                }
                placeholder="you@example.com"
              />
            </label>

            <label className="field">
              <span>密码</span>
              <input
                required
                minLength={6}
                type="password"
                value={model.authForm.password}
                onChange={(event) =>
                  model.setAuthForm((current) => ({
                    ...current,
                    password: event.target.value,
                  }))
                }
                placeholder="至少 6 位"
              />
            </label>

            <button className="button button--primary" disabled={model.authLoading} type="submit">
              {model.authLoading
                ? '提交中...'
                : mode === 'register'
                  ? '注册并登录'
                  : '登录'}
            </button>
          </form>
        </div>
      </section>
    </div>
  );
}

function InstallPage({
  model,
  navigate,
}: {
  model: AppModel;
  navigate: (path: string) => void;
}) {
  const installTarget = model.detailSkill || model.featuredSkills[0] || model.skills[0];

  return (
    <div className="page-stack">
      <section className="surface-panel">
        <SectionHeading
          eyebrow="Install"
          title="安装指南"
          description="先打通 CLI 和 registry，再优先复制主安装命令；其余命令只做排查和辅助。"
        />
        <div className="info-grid info-grid--3">
          <article className="info-card">
            <h3>先确认环境</h3>
            <p>确保本机已有 `skill-home`，先执行 `skill-home doctor` 检查 registry、认证和 IDE 路径。</p>
          </article>
          <article className="info-card">
            <h3>再确认 skill 版本</h3>
            <p>先看详情页版本和扫描状态，再复制目标版本的 install 或 pull 命令。</p>
          </article>
          <article className="info-card">
            <h3>最后执行安装</h3>
            <p>优先使用 `--global --mode mirror`，这样在多 IDE / WSL 混合环境下更稳妥。</p>
          </article>
        </div>
      </section>

      {installTarget ? (
        <section className="surface-panel">
          <div className="install-target-card">
            <div>
              <span className="eyebrow">Example target</span>
              <h3>{installTarget.name}</h3>
              <p>{installTarget.description || '暂无描述。'}</p>
              <code>{skillRef(installTarget)}</code>
            </div>
            <div className="install-target-card__actions">
              <button
                className="button button--secondary"
                onClick={() => navigate(buildSkillPath(installTarget.namespace, installTarget.name))}
                type="button"
              >
                查看详情页
              </button>
              <CopyButton
                className="button button--quiet"
                label="复制引用"
                value={skillRef(installTarget)}
              />
            </div>
          </div>
          <InstallGuide skill={installTarget} title="以当前推荐技能为例的安装指引" />
        </section>
      ) : (
        <div className="empty-panel">等待技能目录加载完成后展示安装示例。</div>
      )}
    </div>
  );
}

export default function App() {
  const { route, navigate } = useRoute();
  const model = useRegistryApp(route, navigate);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);
  const [toast, setToast] = useState<{ tone: ToastTone; message: string } | null>(null);

  const activeNav = route.name === 'skill' ? 'skills' : route.name;

  useEffect(() => {
    let timer = 0;

    function handleToast(event: Event) {
      const detail = (event as CustomEvent<{ tone: ToastTone; message: string }>).detail;
      if (!detail) {
        return;
      }

      setToast(detail);
      window.clearTimeout(timer);
      timer = window.setTimeout(() => setToast(null), 2200);
    }

    window.addEventListener('skill-home-toast', handleToast);

    return () => {
      window.removeEventListener('skill-home-toast', handleToast);
      window.clearTimeout(timer);
    };
  }, []);

  function navigateInternal(path: string) {
    setMobileNavOpen(false);
    setMobileSearchOpen(false);
    navigate(path);
  }

  function handleLogout() {
    setMobileNavOpen(false);
    setMobileSearchOpen(false);
    model.handleLogout();
  }

  function submitGlobalSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigateInternal('/skills');
  }

  return (
    <div className="app-frame">
      {toast ? (
        <div className="toast-stack" aria-live="polite">
          <StatusBanner message={toast.message} tone={toast.tone} />
        </div>
      ) : null}

      <header className="topbar">
        <div className="topbar__brand">
          <button className="brand-mark" onClick={() => navigateInternal('/')} type="button">
            <span>SH</span>
          </button>
          <div>
            <strong>Skill Home</strong>
            <span>AI skill registry</span>
          </div>
        </div>

        <nav className="topbar__nav">
          <InternalButton
            active={routeMatches(route, 'home')}
            className="nav-button"
            label="首页"
            onClick={() => navigateInternal('/')}
          />
          <InternalButton
            active={activeNav === 'skills'}
            className="nav-button"
            label="技能中心"
            onClick={() => navigateInternal('/skills')}
          />
          <InternalButton
            active={routeMatches(route, 'install')}
            className="nav-button"
            label="安装指南"
            onClick={() => navigateInternal('/install')}
          />
          <InternalButton
            active={routeMatches(route, 'publish')}
            className="nav-button"
            label="发布"
            onClick={() => navigateInternal('/publish')}
          />
          <InternalButton
            active={routeMatches(route, 'console')}
            className="nav-button"
            label="控制台"
            onClick={() => navigateInternal('/console')}
          />
        </nav>

        <form
          className={`topbar__search ${mobileSearchOpen ? 'is-open' : ''}`.trim()}
          onSubmit={submitGlobalSearch}
        >
          <input
            value={model.catalogFilters.query}
            onChange={(event) => {
              const nextValue = event.target.value;
              startTransition(() => model.setCatalogQuery(nextValue));
            }}
            placeholder="搜索 skill、能力、场景"
          />
          <button className="button button--quiet" type="submit">
            搜索
          </button>
        </form>

        <div className="topbar__account">
          {model.currentUser ? (
            <>
              <button className="account-chip" onClick={() => navigateInternal('/console')} type="button">
                <strong>{model.currentUser.username}</strong>
                <span>控制台</span>
              </button>
              <button className="button button--quiet" onClick={handleLogout} type="button">
                退出
              </button>
            </>
          ) : (
            <>
              <button className="button button--quiet" onClick={() => navigateInternal('/login')} type="button">
                登录
              </button>
              <button className="button button--secondary" onClick={() => navigateInternal('/register')} type="button">
                注册
              </button>
            </>
          )}
        </div>

        <div className="topbar__mobile-actions">
          <button
            aria-label="切换搜索"
            className="mobile-search-toggle"
            onClick={() => {
              setMobileSearchOpen((value) => !value);
              setMobileNavOpen(false);
            }}
            type="button"
          >
            {mobileSearchOpen ? '收起搜索' : '搜索'}
          </button>

          <button
            aria-label="切换导航"
            className="mobile-nav-toggle"
            onClick={() => {
              setMobileNavOpen((value) => !value);
              setMobileSearchOpen(false);
            }}
            type="button"
          >
            {mobileNavOpen ? '关闭' : '菜单'}
          </button>
        </div>
      </header>

      {mobileNavOpen ? (
        <div className="mobile-nav">
          <InternalButton className="nav-button" label="首页" onClick={() => navigateInternal('/')} />
          <InternalButton className="nav-button" label="技能中心" onClick={() => navigateInternal('/skills')} />
          <InternalButton className="nav-button" label="安装指南" onClick={() => navigateInternal('/install')} />
          <InternalButton className="nav-button" label="发布" onClick={() => navigateInternal('/publish')} />
          <InternalButton className="nav-button" label="控制台" onClick={() => navigateInternal('/console')} />
          <div className="mobile-nav__section">
            {model.currentUser ? (
              <>
                <button className="account-chip" onClick={() => navigateInternal('/console')} type="button">
                  <strong>{model.currentUser.username}</strong>
                  <span>查看控制台</span>
                </button>
                <button className="button button--quiet" onClick={handleLogout} type="button">
                  退出登录
                </button>
              </>
            ) : (
              <>
                <button className="button button--quiet" onClick={() => navigateInternal('/login')} type="button">
                  登录
                </button>
                <button className="button button--secondary" onClick={() => navigateInternal('/register')} type="button">
                  注册
                </button>
              </>
            )}
          </div>
        </div>
      ) : null}

      <main className="app-main">
        {route.name === 'home' ? <HomePage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'skills' ? <SkillsPage model={model} /> : null}
        {route.name === 'skill' ? <SkillDetailPage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'publish' ? <PublishPage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'console' ? <ConsolePage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'install' ? <InstallPage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'auth' ? (
          <AuthPage model={model} mode={route.mode} navigate={navigateInternal} />
        ) : null}
      </main>

      <footer className="footer-bar">
        <div>
          <strong>Skill Home</strong>
          <span>统一 AI skill 的发布、发现、安装和运营入口。</span>
        </div>
        <div className="footer-bar__links">
          <a href={API_BASE} rel="noreferrer" target="_blank">
            Registry API
          </a>
          <button className="footer-link" onClick={() => navigateInternal('/skills')} type="button">
            技能中心
          </button>
          <button className="footer-link" onClick={() => navigateInternal('/publish')} type="button">
            发布
          </button>
          <button className="footer-link" onClick={() => navigateInternal('/install')} type="button">
            安装指南
          </button>
        </div>
      </footer>
    </div>
  );
}
