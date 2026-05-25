import { SettingsAuthCallout } from '../../components/settings/SettingsAuthCallout';
import { SettingsLayout } from '../../components/settings/SettingsLayout';
import { useRegistryApp } from '../../hooks/useRegistryApp';
import { formatDateTime } from '../../lib/format';
import { getAccountSettingsNav } from '../../lib/settings';
import type { AdminUserRoleFilter, AdminUserStatusFilter } from '../../api';

type AppModel = ReturnType<typeof useRegistryApp>;

type UserManagementSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

const roleOptions: { label: string; value: AdminUserRoleFilter }[] = [
  { label: '全部角色', value: 'all' },
  { label: '成员', value: 'member' },
  { label: '管理员', value: 'admin' },
  { label: '超级管理员', value: 'super_admin' },
];

const statusOptions: { label: string; value: AdminUserStatusFilter }[] = [
  { label: '全部状态', value: 'all' },
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
];

function renderStatusBanner(tone: 'danger' | 'success', message: string) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

function getRoleLabel(role: string) {
  if (role === 'super_admin') {
    return '超级管理员';
  }
  if (role === 'admin') {
    return '管理员';
  }
  return '成员';
}

function formatAuditMeta(metadata?: Record<string, unknown>) {
  if (!metadata) {
    return '';
  }

  const username = typeof metadata.username === 'string' ? metadata.username : '';
  const targetUserID = typeof metadata.target_user_id === 'string' ? metadata.target_user_id : '';
  return username || targetUserID;
}

export function UserManagementSettingsPage({ model, navigate }: UserManagementSettingsPageProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后由超级管理员维护用户启停、角色和密码重置。"
        navigate={navigate}
        redirectTo="/settings/users"
        title="Settings"
      />
    );
  }

  const canManageUsers = Boolean(model.currentUser?.is_super_admin);
  const pageCount = Math.max(1, Math.ceil(model.adminUsersTotal / model.adminUserFilters.perPage));

  return (
    <SettingsLayout
      actions={(
        <button
          className="button button--secondary"
          disabled={!canManageUsers || model.adminUsersLoading}
          onClick={model.refreshAdminUsers}
          type="button"
        >
          Refresh
        </button>
      )}
      description="维护平台账号状态、管理员角色和近期权限审计。"
      navAriaLabel="Settings"
      navItems={getAccountSettingsNav('users', canManageUsers)}
      onNavigate={navigate}
      sidebarHeader={model.currentUser ? (
        <div className="gh-settings-sidebar__scope">
          <strong>{model.currentUser.display_name_zh || model.currentUser.username}</strong>
          <span>{model.currentUser.email}</span>
        </div>
      ) : null}
      title="Users"
    >
      {!canManageUsers ? (
        <div className="empty-panel empty-panel--danger">只有超级管理员可以访问用户权限管理。</div>
      ) : (
        <div className="gh-settings-stack">
          {model.adminUsersError ? renderStatusBanner('danger', model.adminUsersError) : null}
          {model.adminUsersSuccess ? renderStatusBanner('success', model.adminUsersSuccess) : null}

          <section className="gh-settings-card">
            <div className="gh-settings-card__header">
              <div>
                <h2>User directory</h2>
                <p>按账号、邮箱、中文名、角色和状态定位用户。</p>
              </div>
              <span className="status-pill status-pill--neutral">{model.adminUsersTotal} users</span>
            </div>

            <form
              className="gh-admin-users-toolbar"
              onSubmit={(event) => {
                event.preventDefault();
                model.refreshAdminUsers();
              }}
            >
              <label className="field">
                <span>搜索</span>
                <input
                  placeholder="用户名、邮箱或中文名"
                  value={model.adminUserFilters.query}
                  onChange={(event) =>
                    model.setAdminUserFilters((current) => ({
                      ...current,
                      page: 1,
                      query: event.target.value,
                    }))
                  }
                />
              </label>
              <label className="field">
                <span>角色</span>
                <select
                  value={model.adminUserFilters.role}
                  onChange={(event) =>
                    model.setAdminUserFilters((current) => ({
                      ...current,
                      page: 1,
                      role: event.target.value as AdminUserRoleFilter,
                    }))
                  }
                >
                  {roleOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field">
                <span>状态</span>
                <select
                  value={model.adminUserFilters.status}
                  onChange={(event) =>
                    model.setAdminUserFilters((current) => ({
                      ...current,
                      page: 1,
                      status: event.target.value as AdminUserStatusFilter,
                    }))
                  }
                >
                  {statusOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
              <button className="button button--secondary" type="submit">
                Apply
              </button>
            </form>

            {model.adminUsersLoading ? (
              <div className="empty-panel">正在读取用户列表...</div>
            ) : model.adminUsers.length ? (
              <div className="gh-admin-users-list">
                {model.adminUsers.map((user) => {
                  const isSelf = user.id === model.currentUser?.id;
                  const saving = model.adminUsersSaving === user.id;
                  const passwordDraft = model.adminUserPasswordDrafts[user.id] || '';

                  return (
                    <article className="gh-admin-user-row" key={user.id}>
                      <div className="gh-admin-user-row__identity">
                        <strong>{user.display_name_zh || user.username}</strong>
                        <span>@{user.username}</span>
                        <span>{user.email}</span>
                      </div>

                      <div className="gh-admin-user-row__badges">
                        <span className={`status-pill status-pill--${user.is_active ? 'success' : 'danger'}`}>
                          {user.is_active ? '启用' : '停用'}
                        </span>
                        <span className={`status-pill status-pill--${user.role === 'member' ? 'neutral' : 'warning'}`}>
                          {getRoleLabel(user.role)}
                        </span>
                        {isSelf ? <span className="status-pill status-pill--neutral">当前账号</span> : null}
                      </div>

                      <div className="gh-admin-user-row__actions">
                        <button
                          className="button button--secondary"
                          disabled={saving || isSelf}
                          onClick={() =>
                            void model.updateAdminUserAccess(user.id, {
                              isActive: !user.is_active,
                            })
                          }
                          type="button"
                        >
                          {user.is_active ? '停用' : '启用'}
                        </button>
                        <button
                          className="button button--secondary"
                          disabled={saving}
                          onClick={() =>
                            void model.updateAdminUserAccess(user.id, {
                              isAdmin: !user.is_admin,
                            })
                          }
                          type="button"
                        >
                          {user.is_admin ? '移除管理员' : '设为管理员'}
                        </button>
                        <button
                          className="button button--secondary"
                          disabled={saving || (isSelf && user.is_super_admin)}
                          onClick={() =>
                            void model.updateAdminUserAccess(user.id, {
                              isSuperAdmin: !user.is_super_admin,
                            })
                          }
                          type="button"
                        >
                          {user.is_super_admin ? '移除超管' : '设为超管'}
                        </button>
                        <label className="field gh-admin-user-row__password">
                          <span>重置密码</span>
                          <input
                            minLength={6}
                            placeholder="新密码"
                            type="password"
                            value={passwordDraft}
                            onChange={(event) =>
                              model.setAdminUserPasswordDrafts((current) => ({
                                ...current,
                                [user.id]: event.target.value,
                              }))
                            }
                          />
                        </label>
                        <button
                          className="button button--secondary"
                          disabled={saving || passwordDraft.trim().length < 6}
                          onClick={() => void model.resetAdminUserPassword(user.id)}
                          type="button"
                        >
                          {saving ? '保存中...' : '重置'}
                        </button>
                      </div>
                    </article>
                  );
                })}
              </div>
            ) : (
              <div className="empty-panel">没有匹配的用户。</div>
            )}

            <div className="gh-admin-users-pagination">
              <button
                className="button button--quiet"
                disabled={model.adminUserFilters.page <= 1}
                onClick={() =>
                  model.setAdminUserFilters((current) => ({
                    ...current,
                    page: Math.max(1, current.page - 1),
                  }))
                }
                type="button"
              >
                Previous
              </button>
              <span>
                Page {model.adminUserFilters.page} / {pageCount}
              </span>
              <button
                className="button button--quiet"
                disabled={model.adminUserFilters.page >= pageCount}
                onClick={() =>
                  model.setAdminUserFilters((current) => ({
                    ...current,
                    page: Math.min(pageCount, current.page + 1),
                  }))
                }
                type="button"
              >
                Next
              </button>
            </div>
          </section>

          <section className="gh-settings-card">
            <div className="gh-settings-card__header">
              <div>
                <h2>Recent audit</h2>
                <p>展示最近的全局管理审计记录。</p>
              </div>
            </div>

            {model.adminAuditLogsError ? renderStatusBanner('danger', model.adminAuditLogsError) : null}
            {model.adminAuditLogsLoading ? (
              <div className="empty-panel">正在读取审计日志...</div>
            ) : model.adminAuditLogs.length ? (
              <div className="gh-admin-audit-list">
                {model.adminAuditLogs.map((log) => (
                  <article className="gh-admin-audit-row" key={log.id}>
                    <div>
                      <strong>{log.action}</strong>
                      <span>{formatAuditMeta(log.metadata) || log.resource_type}</span>
                    </div>
                    <time dateTime={log.created_at}>{formatDateTime(log.created_at)}</time>
                  </article>
                ))}
              </div>
            ) : (
              <div className="empty-panel">暂无审计日志。</div>
            )}
          </section>
        </div>
      )}
    </SettingsLayout>
  );
}
