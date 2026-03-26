import { PageHeader } from '../components/layout/PageHeader';
import { SidebarLayout } from '../components/layout/SidebarLayout';
import { CopyActionButton } from '../components/object/CopyActionButton';
import { useRegistryApp } from '../hooks/useRegistryApp';
import { formatDate, skillKey } from '../lib/format';
import { buildSkillPath } from '../lib/routes';

type AppModel = ReturnType<typeof useRegistryApp>;

type HomePageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

const CLI_INSTALL_COMMAND = 'curl -fsSL http://47.122.112.210:8080/install.sh | bash';
const CLI_VERIFY_COMMAND = 'skill-home doctor';

export function HomePage({ model, navigate }: HomePageProps) {
  return (
    <div className="page-stack">
      <section className="surface-panel gh-release-shell">
        <SidebarLayout
          aside={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Registry</h2>
                    <p>当前实例状态和目录规模。</p>
                  </div>
                </div>
                <div className="gh-settings-summary-grid">
                  <article className="gh-settings-summary-item">
                    <span>Status</span>
                    <strong>{model.health?.status || 'unknown'}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Version</span>
                    <strong>{model.health?.version || '未记录'}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Namespaces</span>
                    <strong>{model.quickStats.namespaceCount}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Tags</span>
                    <strong>{model.quickStats.tagCount}</strong>
                  </article>
                </div>
              </section>

              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Workflow</h2>
                    <p>首页只负责把你送进工作区，不重复解释每一步。</p>
                  </div>
                </div>
                <ol className="gh-ordered-list">
                  <li>先搜索技能，带着关键词或标签进入目录。</li>
                  <li>在对象页确认版本、扫描状态和安装命令。</li>
                  <li>需要发布或维护时再进入 `/publish/new` 和 `/settings/*`。</li>
                </ol>
              </section>
            </div>
          )}
          className="gh-sidebar-layout--home-compact"
          content={(
            <div className="gh-settings-stack">
              <PageHeader
                actions={(
                  <div className="gh-page-header__actions">
                    <button className="button button--primary" onClick={() => navigate('/skills')} type="button">
                      Search skills
                    </button>
                    <button className="button button--secondary" onClick={() => navigate('/publish/new')} type="button">
                      Publish a skill
                    </button>
                  </div>
                )}
                description="Search the registry, inspect a skill, then install or publish from one consistent workspace."
                title="Find and ship skills faster"
              />

              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Install CLI</h2>
                    <p>先装好 `skill-home` CLI，再进入技能搜索、安装和发布工作流。</p>
                  </div>
                  <div className="gh-object-command-card__actions">
                    <CopyActionButton
                      className="button button--primary"
                      label="复制安装命令"
                      value={CLI_INSTALL_COMMAND}
                    />
                    <button className="button button--secondary" onClick={() => navigate('/install')} type="button">
                      Open install guide
                    </button>
                  </div>
                </div>

                <div className="gh-object-command-card">
                  <div className="gh-object-command-card__header">
                    <div>
                      <strong>Primary install command</strong>
                      <p>用固定入口脚本拉取对应平台的最新 CLI 版本。</p>
                    </div>
                    <CopyActionButton className="button button--quiet" label="复制命令" value={CLI_INSTALL_COMMAND} />
                  </div>
                  <code>{CLI_INSTALL_COMMAND}</code>
                </div>

                <div className="gh-settings-summary-grid">
                  <article className="gh-settings-summary-item">
                    <span>Entry</span>
                    <strong>get.skill-home.dev</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Artifacts</span>
                    <strong>GitHub Releases</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Verify</span>
                    <strong>{CLI_VERIFY_COMMAND}</strong>
                  </article>
                </div>

                <ol className="gh-ordered-list">
                  <li>先执行安装脚本，把 `skill-home` 放进本机 PATH。</li>
                  <li>运行 `skill-home doctor`，确认 registry 和 IDE 路径可用。</li>
                  <li>再进入技能中心，搜索并安装需要的 skill。</li>
                </ol>
              </section>

              <div className="gh-home-panel-grid">
                <section className="gh-settings-card">
                  <div className="gh-settings-card__header">
                    <div>
                      <h2>Recently updated</h2>
                      <p>直接进入最近仍在维护的技能对象页。</p>
                    </div>
                  </div>
                  {model.latestSkills.length ? (
                    <div className="gh-settings-list">
                      {model.latestSkills.slice(0, 5).map((skill) => (
                        <button
                          className="gh-settings-list__row"
                          key={skillKey(skill)}
                          onClick={() => navigate(buildSkillPath(skill.namespace, skill.name))}
                          type="button"
                        >
                          <div className="gh-settings-list__main">
                            <strong>{skill.name}</strong>
                            <span>@{skill.namespace}/{skill.name}</span>
                          </div>
                          <div className="gh-settings-list__meta">
                            <span className="status-pill status-pill--neutral">{skill.latest_version || 'draft'}</span>
                            <span>{formatDate(skill.updated_at)}</span>
                          </div>
                        </button>
                      ))}
                    </div>
                  ) : (
                    <div className="empty-panel">等待目录加载后展示最近更新。</div>
                  )}
                </section>

                <section className="gh-settings-card">
                  <div className="gh-settings-card__header">
                    <div>
                      <h2>Popular tags</h2>
                      <p>用常见主题直接进入搜索工作台。</p>
                    </div>
                  </div>
                  <div className="filter-chip-row">
                    {model.tagOptions.slice(0, 6).map((tag) => (
                      <button
                        className="filter-chip-button"
                        key={tag}
                        onClick={() => navigate(`/skills?tag=${encodeURIComponent(tag)}`)}
                        type="button"
                      >
                        {tag}
                      </button>
                    ))}
                  </div>
                </section>
              </div>
            </div>
          )}
        />
      </section>
    </div>
  );
}
