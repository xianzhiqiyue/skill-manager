import { useState } from 'react';

import type { SkillDetail, SkillSummary } from '../api';
import { PageHeader } from '../components/layout/PageHeader';
import { SidebarLayout } from '../components/layout/SidebarLayout';
import { CopyActionButton } from '../components/object/CopyActionButton';
import { useRegistryApp } from '../hooks/useRegistryApp';
import {
  formatBytes,
  formatDateTime,
  getInstallRecipes,
  skillRef,
  summarizeScanStatus,
  type InstallIDE,
} from '../lib/format';
import { buildSkillPath } from '../lib/routes';

type AppModel = ReturnType<typeof useRegistryApp>;

type InstallDocsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

function CommandCard({
  title,
  description,
  value,
}: {
  description: string;
  title: string;
  value: string;
}) {
  return (
    <article className="gh-object-command-card">
      <div className="gh-object-command-card__header">
        <div>
          <strong>{title}</strong>
          <p>{description}</p>
        </div>
        <CopyActionButton className="button button--quiet" value={value} />
      </div>
      <code>{value}</code>
    </article>
  );
}

function isSkillDetail(skill: SkillDetail | SkillSummary): skill is SkillDetail {
  return Array.isArray((skill as SkillDetail).versions);
}

export function InstallDocsPage({ model, navigate }: InstallDocsPageProps) {
  const installTarget = model.detailSkill || model.featuredSkills[0] || model.skills[0];
  const [selectedIDE, setSelectedIDE] = useState<InstallIDE>('codex');

  if (!installTarget) {
    return <div className="empty-panel">等待技能目录加载完成后展示安装示例。</div>;
  }

  const recipes = getInstallRecipes(installTarget);
  const recipe = recipes.find((item) => item.id === selectedIDE) || recipes[0];
  const versions = isSkillDetail(installTarget) ? installTarget.versions || [] : [];
  const latestVersion = versions.find((version) => version.version === installTarget.latest_version) || null;
  const scan = summarizeScanStatus(latestVersion?.scan_status);

  return (
    <div className="page-stack">
      <section className="surface-panel gh-release-shell">
        <PageHeader
          actions={(
            <button
              className="button button--secondary"
              onClick={() => navigate(buildSkillPath(installTarget.namespace, installTarget.name))}
              type="button"
            >
              Open object page
            </button>
          )}
          description="Pick a target, copy the primary install command, then use support commands only for verification or repair."
          title="Install from the registry"
        />

        <SidebarLayout
          aside={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Selected skill</h2>
                    <p>{installTarget.description || '暂无描述。'}</p>
                  </div>
                </div>
                <div className="gh-settings-summary-grid">
                  <article className="gh-settings-summary-item">
                    <span>Reference</span>
                    <strong>{skillRef(installTarget)}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Latest version</span>
                    <strong>{installTarget.latest_version || '未记录'}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Package size</span>
                    <strong>{formatBytes(latestVersion?.size_bytes)}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Scan</span>
                    <strong>{scan.label}</strong>
                  </article>
                </div>
              </section>

              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Workflow</h2>
                    <p>让安装页保持操作导向，不重复详情页已经展示的说明。</p>
                  </div>
                </div>
                <ol className="gh-ordered-list">
                  <li>先执行 `skill-home doctor` 检查环境。</li>
                  <li>用主安装命令完成同步。</li>
                  <li>只在需要排查时再运行搜索或 pull 命令。</li>
                </ol>
              </section>
            </div>
          )}
          className="gh-sidebar-layout--release"
          content={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Install target</h2>
                    <p>{skillRef(installTarget)}</p>
                  </div>
                  <div className="gh-object-command-card__actions">
                    <CopyActionButton className="button button--secondary" label="复制引用名" value={skillRef(installTarget)} />
                    <CopyActionButton className="button button--quiet" label="复制下载链接" value={recipe.download} />
                  </div>
                </div>

                <div className="segmented-group segmented-group--wide">
                  {recipes.map((item) => (
                    <button
                      className={`segmented-button ${item.id === recipe.id ? 'is-active' : ''}`}
                      key={item.id}
                      onClick={() => setSelectedIDE(item.id)}
                      type="button"
                    >
                      {item.label}
                    </button>
                  ))}
                </div>

                <div className="gh-object-command-card">
                  <div className="gh-object-command-card__header">
                    <div>
                      <strong>Primary install command</strong>
                      <p>{recipe.description}</p>
                    </div>
                    <div className="gh-object-command-card__actions">
                      <CopyActionButton className="button button--primary" label="复制安装命令" value={recipe.install} />
                      <CopyActionButton className="button button--quiet" label="复制 pull 命令" value={recipe.pull} />
                    </div>
                  </div>
                  <code>{recipe.install}</code>
                </div>
              </section>

              <div className="gh-object-install-grid">
                <CommandCard
                  description="先确认 CLI、registry 和 IDE 路径可用。"
                  title="Doctor"
                  value={recipe.verify}
                />
                <CommandCard
                  description="先确认名字和可用版本，再决定是否安装。"
                  title="Search"
                  value={recipe.search}
                />
                <CommandCard
                  description="需要检查缓存或包内容时再执行。"
                  title="Pull package"
                  value={recipe.pull}
                />
                <CommandCard
                  description={`最近更新时间：${formatDateTime(latestVersion?.published_at || latestVersion?.created_at)}`}
                  title="Download archive"
                  value={recipe.download}
                />
              </div>
            </div>
          )}
        />
      </section>
    </div>
  );
}
