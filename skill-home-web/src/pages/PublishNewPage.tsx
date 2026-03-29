import { resolveDownloadUrl } from '../api';
import { PageHeader } from '../components/layout/PageHeader';
import { SidebarLayout } from '../components/layout/SidebarLayout';
import { CopyActionButton } from '../components/object/CopyActionButton';
import { SettingsAuthCallout } from '../components/settings/SettingsAuthCallout';
import { useRegistryApp } from '../hooks/useRegistryApp';
import { buildSkillPath } from '../lib/routes';

type AppModel = ReturnType<typeof useRegistryApp>;

type PublishNewPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
};

function renderStatusBanner(tone: 'danger' | 'success', message: string) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

export function PublishNewPage({ model, navigate }: PublishNewPageProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后上传 ZIP、填写元信息，并把新版本发布到 Skill Home。"
        navigate={navigate}
        redirectTo="/publish/new"
        title="Create a new release"
      />
    );
  }

  return (
    <div className="page-stack">
      <section className="surface-panel gh-release-shell">
        <PageHeader
          description="Upload a ZIP, declare version metadata, then jump straight into the object page to verify the new release."
          title="Create a new release"
        />

        <SidebarLayout
          aside={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Release flow</h2>
                    <p>保持创建页只做一件事：准备版本、上传压缩包、确认结果。</p>
                  </div>
                </div>
                <ol className="gh-ordered-list">
                  <li>先执行 `skill-home validate` 和 `skill-home pack`。</li>
                  <li>确认命名空间、版本号和 ZIP 内容一致。</li>
                  <li>发布后立刻打开对象页检查版本、扫描状态和安装入口。</li>
                </ol>
              </section>

              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Defaults</h2>
                    <p>这个表单只保留发布真正需要的字段，额外元信息放到对象设置里补充。</p>
                  </div>
                </div>
                <div className="gh-settings-summary-grid">
                  <article className="gh-settings-summary-item">
                    <span>Namespace</span>
                    <strong>{model.publishForm.namespace || '未填写'}</strong>
                  </article>
                  <article className="gh-settings-summary-item">
                    <span>Version</span>
                    <strong>{model.publishForm.version || '未填写'}</strong>
                  </article>
                </div>
              </section>
            </div>
          )}
          className="gh-sidebar-layout--release"
          content={(
            <div className="gh-settings-stack">
              <section className="gh-settings-card">
                {model.publishError ? renderStatusBanner('danger', model.publishError) : null}
                {model.publishSuccess
                  ? renderStatusBanner(
                    'success',
                    `发布成功：${model.publishSuccess.namespace}/${model.publishSuccess.name}@${model.publishSuccess.version}`,
                  )
                  : null}

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
                        placeholder="例如 testuser"
                        required
                        value={model.publishForm.namespace}
                        onChange={(event) =>
                          model.setPublishForm((current) => ({
                            ...current,
                            namespace: event.target.value,
                          }))
                        }
                      />
                    </label>

                    <label className="field">
                      <span>版本号</span>
                      <input
                        placeholder="0.1.0"
                        required
                        value={model.publishForm.version}
                        onChange={(event) =>
                          model.setPublishForm((current) => ({
                            ...current,
                            version: event.target.value,
                          }))
                        }
                      />
                    </label>
                  </div>

                  <label className="field">
                    <span>技能名</span>
                    <input
                      placeholder="例如 skill-home-manager"
                      required
                      value={model.publishForm.name}
                      onChange={(event) =>
                        model.setPublishForm((current) => ({
                          ...current,
                          name: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <label className="field">
                    <span>描述</span>
                    <textarea
                      placeholder="简要说明这个 skill 解决什么问题"
                      required
                      rows={5}
                      value={model.publishForm.description}
                      onChange={(event) =>
                        model.setPublishForm((current) => ({
                          ...current,
                          description: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <label className="field">
                    <span>Tags</span>
                    <input
                      placeholder="automation, github, cli"
                      value={model.publishForm.tags}
                      onChange={(event) =>
                        model.setPublishForm((current) => ({
                          ...current,
                          tags: event.target.value,
                        }))
                      }
                    />
                  </label>

                  <div className="form-grid-two">
                    <label className="field">
                      <span>License</span>
                      <input
                        placeholder="MIT"
                        value={model.publishForm.license}
                        onChange={(event) =>
                          model.setPublishForm((current) => ({
                            ...current,
                            license: event.target.value,
                          }))
                        }
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

                  <div className="gh-settings-actions">
                    <button className="button button--primary" disabled={model.publishLoading} type="submit">
                      {model.publishLoading ? '发布中...' : 'Upload release'}
                    </button>
                  </div>
                </form>
              </section>

              {model.publishSuccess ? (
                <section className="gh-settings-card">
                  <div className="gh-settings-card__header">
                    <div>
                      <h2>Release created</h2>
                      <p>
                        {model.publishSuccess.namespace}/{model.publishSuccess.name}@{model.publishSuccess.version}
                      </p>
                    </div>
                    <div className="gh-object-command-card__actions">
                      <button
                        className="button button--secondary"
                        onClick={() =>
                          navigate(
                            buildSkillPath(model.publishSuccess!.namespace, model.publishSuccess!.name),
                          )
                        }
                        type="button"
                      >
                        Open object page
                      </button>
                      <CopyActionButton
                        className="button button--quiet"
                        label="复制下载链接"
                        value={resolveDownloadUrl(model.publishSuccess.download_url)!}
                      />
                    </div>
                  </div>
                </section>
              ) : null}
            </div>
          )}
        />
      </section>
    </div>
  );
}
