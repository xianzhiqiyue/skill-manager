import { SettingsAuthCallout } from '../../components/settings/SettingsAuthCallout';
import { SettingsLayout } from '../../components/settings/SettingsLayout';
import { useRegistryApp } from '../../hooks/useRegistryApp';
import { getAccountSettingsNav } from '../../lib/settings';

type AppModel = ReturnType<typeof useRegistryApp>;

type StatsSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

function formatRating(averageRating: number, totalRatings: number) {
  if (!totalRatings) {
    return '暂无';
  }

  return `${averageRating.toFixed(1)} / ${totalRatings}`;
}

export function StatsSettingsPage({ model, navigate }: StatsSettingsPageProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后查看你发布的 skill、获赞、安装、下载和评分统计。"
        navigate={navigate}
        redirectTo="/settings/stats"
        title="Stats"
      />
    );
  }

  const canManageUsers = Boolean(model.currentUser?.is_super_admin);
  const rankedSkills = [...model.mySkills]
    .sort(
      (left, right) =>
        (right.install_count || 0) - (left.install_count || 0) ||
        (right.like_count || 0) - (left.like_count || 0) ||
        right.download_count - left.download_count ||
        left.name.localeCompare(right.name),
    )
    .slice(0, 12);

  return (
    <SettingsLayout
      actions={(
        <button className="button button--secondary" onClick={() => navigate('/settings/profile')} type="button">
          Profile
        </button>
      )}
      description="查看账号产出、互动和安装表现。"
      navAriaLabel="Settings"
      navItems={getAccountSettingsNav('stats', canManageUsers)}
      onNavigate={navigate}
      sidebarHeader={model.currentUser ? (
        <div className="gh-settings-sidebar__scope">
          <strong>{model.currentUser.display_name_zh || model.currentUser.username}</strong>
          <span>{model.currentUser.email}</span>
        </div>
      ) : null}
      title="Stats"
    >
      <div className="gh-settings-stack">
        {model.accountError ? (
          <div className="status-banner status-banner--danger">{model.accountError}</div>
        ) : null}

        <section className="gh-settings-card">
          <div className="gh-settings-card__header">
            <div>
              <h2>Account statistics</h2>
              <p>当前账号名下 skill 的汇总指标。</p>
            </div>
          </div>

          <div className="gh-settings-summary-grid">
            <article className="gh-settings-summary-item">
              <span>Skills</span>
              <strong>{model.accountStats.total}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Public</span>
              <strong>{model.accountStats.publicCount}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Private</span>
              <strong>{model.accountStats.privateCount}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Likes</span>
              <strong>{model.accountStats.totalLikes}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Installs</span>
              <strong>{model.accountStats.totalInstalls}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Downloads</span>
              <strong>{model.accountStats.totalDownloads}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Rating</span>
              <strong>{formatRating(model.accountStats.averageRating, model.accountStats.totalRatings)}</strong>
            </article>
          </div>
        </section>

        <section className="gh-settings-card">
          <div className="gh-settings-card__header">
            <div>
              <h2>Skill performance</h2>
              <p>按安装、点赞和下载排序。</p>
            </div>
          </div>

          {model.accountLoading ? (
            <div className="empty-panel">正在读取统计数据...</div>
          ) : rankedSkills.length ? (
            <div className="gh-stats-skill-list">
              {rankedSkills.map((skill) => (
                <button
                  className="gh-stats-skill-row"
                  key={`${skill.namespace}/${skill.name}`}
                  onClick={() => navigate(`/skills/${encodeURIComponent(skill.namespace)}/${encodeURIComponent(skill.name)}`)}
                  type="button"
                >
                  <div className="gh-stats-skill-row__title">
                    <strong>{skill.name}</strong>
                    <span>@{skill.namespace}/{skill.name}</span>
                  </div>
                  <div className="gh-stats-skill-row__metrics">
                    <span>{skill.like_count || 0} likes</span>
                    <span>{skill.install_count || 0} installs</span>
                    <span>{skill.download_count} downloads</span>
                    <span>{formatRating(skill.rating || 0, skill.rating_count)}</span>
                  </div>
                </button>
              ))}
            </div>
          ) : (
            <div className="empty-panel">你还没有发布任何 skill。</div>
          )}
        </section>
      </div>
    </SettingsLayout>
  );
}
