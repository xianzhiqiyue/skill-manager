import { buildSkillSettingsPath, getAccountSettingsNav } from '../../lib/settings';
import { useRegistryApp } from '../../hooks/useRegistryApp';
import { SettingsAuthCallout } from '../../components/settings/SettingsAuthCallout';
import { SettingsLayout } from '../../components/settings/SettingsLayout';

type AppModel = ReturnType<typeof useRegistryApp>;

type ProfileSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

function renderStatusBanner(tone: 'danger' | 'success', message: string) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

export function ProfileSettingsPage({ model, navigate }: ProfileSettingsPageProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后查看账号概览、已发布技能，并进入每个 skill 的设置工作区。"
        navigate={navigate}
        redirectTo="/settings/profile"
        title="Settings"
      />
    );
  }

  const canManageCatalog = Boolean(model.currentUser?.is_admin || model.currentUser?.is_super_admin);

  function canEditSkillSettings(skill: (typeof model.mySkills)[number]) {
    return Boolean(
      model.currentUser &&
        (model.currentUser.is_super_admin || (skill.owner_id && skill.owner_id === model.currentUser.id)),
    );
  }

  return (
    <SettingsLayout
      actions={(
        <button className="button button--secondary" onClick={() => navigate('/publish/new')} type="button">
          New release
        </button>
      )}
      description="管理账号概览、已发布技能和进入对象级设置。"
      navAriaLabel="Settings"
      navItems={getAccountSettingsNav('profile')}
      onNavigate={navigate}
      sidebarHeader={model.currentUser ? (
        <div className="gh-settings-sidebar__scope">
          <strong>{model.currentUser.username}</strong>
          <span>{model.currentUser.email}</span>
        </div>
      ) : null}
      title="Profile"
    >
      <div className="gh-settings-stack">
        {model.accountError ? renderStatusBanner('danger', model.accountError) : null}

        <section className="gh-settings-card">
          <div className="gh-settings-card__header">
            <div>
              <h2>Account</h2>
              <p>账号基础信息和当前管理范围。</p>
            </div>
          </div>

          <div className="gh-settings-summary-grid">
            <article className="gh-settings-summary-item">
              <span>Username</span>
              <strong>{model.currentUser?.username || '未命名用户'}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Joined</span>
              <strong>{model.currentUser?.created_at ? new Date(model.currentUser.created_at).toLocaleDateString('zh-CN') : '未知'}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>Skills</span>
              <strong>{model.accountStats.total}</strong>
            </article>
            <article className="gh-settings-summary-item">
              <span>API keys</span>
              <strong>{model.apiKeyStats.total}</strong>
            </article>
          </div>
        </section>

        <section className="gh-settings-card">
          <div className="gh-settings-card__header">
            <div>
              <h2>{canManageCatalog ? 'Managed skills' : 'Your skills'}</h2>
              <p>
                {canManageCatalog
                  ? '管理角色视角会列出你可管理的技能，便于快速进入推荐设置。'
                  : '每个 skill 都进入独立的设置工作区，不再在这里堆叠整页表单。'}
              </p>
            </div>
          </div>

          {model.accountLoading ? (
            <div className="empty-panel">正在读取已发布技能...</div>
          ) : model.mySkills.length ? (
            <div className="gh-settings-list">
              {model.mySkills.map((skill) => {
                return (
                  <button
                    className="gh-settings-list__row"
                    key={`${skill.namespace}/${skill.name}`}
                    onClick={() =>
                      navigate(
                        buildSkillSettingsPath(
                          skill.namespace,
                          skill.name,
                          canEditSkillSettings(skill) ? 'general' : 'access',
                        ),
                      )
                    }
                    type="button"
                  >
                    <div className="gh-settings-list__main">
                      <strong>{skill.name}</strong>
                      <span>@{skill.namespace}/{skill.name}</span>
                    </div>
                    <div className="gh-settings-list__meta">
                      {skill.is_recommended ? (
                        <span className="status-pill status-pill--success">Recommended</span>
                      ) : null}
                      <span className={`status-pill status-pill--${skill.is_public === false ? 'neutral' : 'success'}`}>
                        {skill.is_public === false ? 'Private' : 'Public'}
                      </span>
                      {skill.is_deprecated ? (
                        <span className="status-pill status-pill--warning">Deprecated</span>
                      ) : null}
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="empty-panel">你还没有发布任何技能。</div>
          )}
        </section>
      </div>
    </SettingsLayout>
  );
}
