import { buildAuthPath, parseAuthRedirect } from '../lib/routes';
import { PageHeader } from '../components/layout/PageHeader';
import { SidebarLayout } from '../components/layout/SidebarLayout';
import { useRegistryApp } from '../hooks/useRegistryApp';

type AppModel = ReturnType<typeof useRegistryApp>;

type AuthPageProps = {
  locationSearch: string;
  model: AppModel;
  mode: 'login' | 'register';
  navigate: (path: string) => void;
};

function renderStatusBanner(tone: 'danger' | 'success', message: string) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

export function AuthPage({ locationSearch, model, mode, navigate }: AuthPageProps) {
  const redirectTo = parseAuthRedirect(locationSearch) || undefined;

  return (
    <div className="page-stack">
      <section className="surface-panel gh-release-shell">
        <PageHeader
          description="Use one account to publish skills, manage settings, and create API keys for automation."
          title={mode === 'register' ? 'Create your Skill Home account' : 'Sign in to Skill Home'}
        />

        <SidebarLayout
          aside={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>What you unlock</h2>
                    <p>登录后进入统一的产品工作区，而不是再停留在分散的落地页入口。</p>
                  </div>
                </div>
                <ul className="gh-bullet-list">
                  <li>发布和更新 skill 版本</li>
                  <li>进入 `/settings/*` 管理对象级设置</li>
                  <li>为 CLI、CI 和外部集成生成 API Keys</li>
                </ul>
              </section>
            </div>
          )}
          className="gh-sidebar-layout--release"
          content={(
            <section className="gh-settings-card">
              {model.authError ? renderStatusBanner('danger', model.authError) : null}
              {model.authSuccess ? renderStatusBanner('success', model.authSuccess) : null}

              <div className="segmented-group segmented-group--wide">
                <button
                  className={`segmented-button ${mode === 'login' ? 'is-active' : ''}`}
                  onClick={() => navigate(buildAuthPath('login', redirectTo))}
                  type="button"
                >
                  登录
                </button>
                <button
                  className={`segmented-button ${mode === 'register' ? 'is-active' : ''}`}
                  onClick={() => navigate(buildAuthPath('register', redirectTo))}
                  type="button"
                >
                  注册
                </button>
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
                      placeholder="例如 testuser"
                      required
                      value={model.authForm.username}
                      onChange={(event) =>
                        model.setAuthForm((current) => ({
                          ...current,
                          username: event.target.value,
                        }))
                      }
                    />
                  </label>
                ) : null}

                <label className="field">
                  <span>邮箱</span>
                  <input
                    placeholder="you@example.com"
                    required
                    type="email"
                    value={model.authForm.email}
                    onChange={(event) =>
                      model.setAuthForm((current) => ({
                        ...current,
                        email: event.target.value,
                      }))
                    }
                  />
                </label>

                <label className="field">
                  <span>密码</span>
                  <input
                    minLength={6}
                    placeholder="至少 6 位"
                    required
                    type="password"
                    value={model.authForm.password}
                    onChange={(event) =>
                      model.setAuthForm((current) => ({
                        ...current,
                        password: event.target.value,
                      }))
                    }
                  />
                </label>

                <div className="gh-settings-actions">
                  <button className="button button--primary" disabled={model.authLoading} type="submit">
                    {model.authLoading
                      ? '提交中...'
                      : mode === 'register'
                        ? 'Create account'
                        : 'Sign in'}
                  </button>
                </div>
              </form>
            </section>
          )}
        />
      </section>
    </div>
  );
}
