import { CopyActionButton } from '../../components/object/CopyActionButton';
import { SettingsAuthCallout } from '../../components/settings/SettingsAuthCallout';
import { SettingsLayout } from '../../components/settings/SettingsLayout';
import { useRegistryApp } from '../../hooks/useRegistryApp';
import { formatDateTime } from '../../lib/format';
import {
  formatDateTimeLocalValue,
  getAccountSettingsNav,
  summarizeAPIKeyStatus,
} from '../../lib/settings';
import { API_BASE } from '../../api';

type AppModel = ReturnType<typeof useRegistryApp>;

type APIKeysSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

const apiKeyExpiryOptions = [
  { value: 'never', label: '永不过期' },
  { value: '7d', label: '7 天' },
  { value: '30d', label: '30 天' },
  { value: '90d', label: '90 天' },
  { value: 'custom', label: '自定义' },
] as const;

function renderStatusBanner(
  tone: 'danger' | 'success',
  message: string,
) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

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

export function APIKeysSettingsPage({
  model,
  navigate,
}: APIKeysSettingsPageProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后创建、撤销和查看用于 CLI、自动化任务与外部集成的 API Keys。"
        navigate={navigate}
        redirectTo="/settings/api-keys"
        title="Settings"
      />
    );
  }

  const minCustomExpiry = formatDateTimeLocalValue(new Date(Date.now() + 5 * 60 * 1000));
  const revealedCurlCommand = model.revealedAPIKey
    ? `curl -H "Authorization: Bearer ${model.revealedAPIKey.key}" ${API_BASE}/api/v1/user`
    : '';
  const revealedLoginCommand = model.revealedAPIKey
    ? `skill-home login --api-key "${model.revealedAPIKey.key}"`
    : '';
  const revealedEnvCommand = model.revealedAPIKey
    ? `export SKILL_HOME_API_KEY="${model.revealedAPIKey.key}"`
    : '';
  const canManageUsers = Boolean(model.currentUser?.is_super_admin);

  return (
    <SettingsLayout
      actions={(
        <button className="button button--secondary" onClick={model.refreshAPIKeys} type="button">
          Refresh
        </button>
      )}
      description="管理用于脚本和集成的长期凭证。完整密钥只会在创建成功后展示一次。"
      navAriaLabel="Settings"
      navItems={getAccountSettingsNav('api-keys', canManageUsers)}
      onNavigate={navigate}
      sidebarHeader={model.currentUser ? (
        <div className="gh-settings-sidebar__scope">
          <strong>{model.currentUser.username}</strong>
          <span>{model.currentUser.email}</span>
        </div>
      ) : null}
      title="API Keys"
    >
      <div className="gh-settings-stack">
        {model.apiKeysError ? renderStatusBanner('danger', model.apiKeysError) : null}
        {model.apiKeysSuccess ? renderStatusBanner('success', model.apiKeysSuccess) : null}

        <div className="gh-settings-summary-grid">
          <article className="gh-settings-summary-item">
            <span>Total</span>
            <strong>{model.apiKeyStats.total}</strong>
          </article>
          <article className="gh-settings-summary-item">
            <span>Active</span>
            <strong>{model.apiKeyStats.active}</strong>
          </article>
          <article className="gh-settings-summary-item">
            <span>Expiring soon</span>
            <strong>{model.apiKeyStats.expiringSoon}</strong>
          </article>
        </div>

        <section className="gh-settings-card gh-settings-card--api-keys">
          <div className="gh-settings-card__section gh-settings-card__section--top">
            <div className="gh-settings-card__header">
              <div>
                <h2>Create new key</h2>
                <p>按用途命名，长期 key 只发给稳定运行的脚本或集成。</p>
              </div>
              <span className="status-pill status-pill--neutral">JWT 适合浏览器，API Key 适合脚本</span>
            </div>

            <form
              className="form-grid-stack"
              onSubmit={(event) => {
                event.preventDefault();
                void model.submitAPIKeyCreate();
              }}
            >
              <label className="field">
                <span>名称</span>
                <input
                  placeholder="例如：CI deploy / Local CLI / Cursor integration"
                  value={model.apiKeyForm.name}
                  onChange={(event) =>
                    model.setAPIKeyForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
              </label>

              <div className="field">
                <span>过期时间</span>
                <div className="segmented-group">
                  {apiKeyExpiryOptions.map((option) => (
                    <button
                      className={`segmented-button ${model.apiKeyForm.expiryPreset === option.value ? 'is-active' : ''}`}
                      key={option.value}
                      onClick={() =>
                        model.setAPIKeyForm((current) => ({
                          ...current,
                          expiryPreset: option.value,
                        }))
                      }
                      type="button"
                    >
                      {option.label}
                    </button>
                  ))}
                </div>

                {model.apiKeyForm.expiryPreset === 'custom' ? (
                  <label className="field api-key-custom-expiry">
                    <span>自定义到期时间</span>
                    <input
                      min={minCustomExpiry}
                      type="datetime-local"
                      value={model.apiKeyForm.customExpiresAt}
                      onChange={(event) =>
                        model.setAPIKeyForm((current) => ({
                          ...current,
                          customExpiresAt: event.target.value,
                        }))
                      }
                    />
                  </label>
                ) : null}
              </div>

              <button className="button button--primary" disabled={model.apiKeyCreating} type="submit">
                {model.apiKeyCreating ? '生成中...' : '生成 API Key'}
              </button>
            </form>
          </div>

          <div className="gh-settings-card__section">
            <div className="gh-settings-card__header">
              <div>
                <h2>Existing keys</h2>
                <p>按名称、前缀和时间信息管理，撤销后依赖它的脚本会立即失效。</p>
              </div>
            </div>

            <div className="version-table api-key-table">
              <div className="version-table__header api-key-table__header">
                <span>名称</span>
                <span>前缀</span>
                <span>时间信息</span>
                <span>状态</span>
                <span>操作</span>
              </div>

              {model.apiKeysLoading ? (
                <div className="empty-panel">正在读取 API Key 列表...</div>
              ) : model.apiKeys.length ? (
                model.apiKeys.map((apiKey) => {
                  const state = summarizeAPIKeyStatus(apiKey.expires_at);
                  return (
                    <div className="version-table__row api-key-table__row" key={apiKey.id}>
                      <strong>{apiKey.name}</strong>
                      <code>{apiKey.prefix}...</code>
                      <div className="api-key-table__time">
                        <div className="api-key-table__time-item">
                          <span>创建</span>
                          <time dateTime={apiKey.created_at}>{formatDateTime(apiKey.created_at)}</time>
                        </div>
                        <div className="api-key-table__time-item">
                          <span>最近使用</span>
                          {apiKey.last_used_at ? (
                            <time dateTime={apiKey.last_used_at}>{formatDateTime(apiKey.last_used_at)}</time>
                          ) : (
                            <strong>未使用</strong>
                          )}
                        </div>
                        <div className="api-key-table__time-item">
                          <span>过期</span>
                          {apiKey.expires_at ? (
                            <time dateTime={apiKey.expires_at}>{formatDateTime(apiKey.expires_at)}</time>
                          ) : (
                            <strong>永不过期</strong>
                          )}
                        </div>
                      </div>
                      <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span>
                      <button
                        className="button button--ghost button--danger-text"
                        disabled={model.apiKeyRevoking === apiKey.id}
                        onClick={() => void model.removeAPIKey(apiKey.id)}
                        type="button"
                      >
                        {model.apiKeyRevoking === apiKey.id ? '撤销中...' : '撤销'}
                      </button>
                    </div>
                  );
                })
              ) : (
                <div className="empty-panel">还没有 API Key。</div>
              )}
            </div>
          </div>
        </section>

        {model.revealedAPIKey ? (
          <section className="gh-settings-card">
            <div className="gh-settings-card__header">
              <div>
                <h2>Save this key now</h2>
                <p>关闭、刷新或离开页面后，完整密钥不会再显示。</p>
              </div>
              <div className="gh-object-command-card__actions">
                <CopyActionButton className="button button--primary" label="复制 Key" value={model.revealedAPIKey.key} />
                <button className="button button--secondary" onClick={() => model.setRevealedAPIKey(null)} type="button">
                  我已保存
                </button>
              </div>
            </div>
            <code className="gh-settings-inline-code">{model.revealedAPIKey.key}</code>
            <div className="gh-object-install-grid">
              <CommandCard
                description="直接请求当前用户接口，确认这把 key 已可用。"
                title="Verify current key"
                value={revealedCurlCommand}
              />
              <CommandCard
                description="适合本地 CLI 首次登录，命令会校验并保存到本机配置。"
                title="Login CLI"
                value={revealedLoginCommand}
              />
              <CommandCard
                description="适合放进本地 shell、CI secret 或受控环境变量。"
                title="Export CLI env"
                value={revealedEnvCommand}
              />
            </div>
          </section>
        ) : null}
      </div>
    </SettingsLayout>
  );
}
