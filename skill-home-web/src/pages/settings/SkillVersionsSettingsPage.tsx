import { useRegistryApp } from '../../hooks/useRegistryApp';
import { formatBytes, formatDateTime, skillKey, summarizeScanStatus } from '../../lib/format';
import { SkillSettingsFrame } from './SkillSettingsFrame';

type AppModel = ReturnType<typeof useRegistryApp>;

type SkillSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
  namespace: string;
  skillName: string;
};

export function SkillVersionsSettingsPage({
  model,
  navigate,
  namespace,
  skillName,
}: SkillSettingsPageProps) {
  const targetKey = `${namespace}/${skillName}`;
  const skill =
    model.managedSkill && skillKey(model.managedSkill) === targetKey ? model.managedSkill : null;

  return (
    <SkillSettingsFrame
      description="查看可管理版本，并删除不再需要保留的发布记录。"
      model={model}
      namespace={namespace}
      navigate={navigate}
      section="versions"
      skillName={skillName}
      title="Versions"
    >
      {skill ? (
        <section className="gh-settings-card">
          <div className="gh-settings-card__header">
            <div>
              <h2>Versions</h2>
              <p>版本历史保留在对象设置里，删除动作也集中在这里。</p>
            </div>
          </div>

          <div className="version-table">
            <div className="version-table__header">
              <span>版本</span>
              <span>时间</span>
              <span>大小</span>
              <span>状态</span>
              <span>操作</span>
            </div>

            {skill.versions?.length ? (
              skill.versions.map((version) => {
                const state = summarizeScanStatus(version.scan_status);
                return (
                  <div className="version-table__row" key={version.id}>
                    <strong>{version.version}</strong>
                    <span>{formatDateTime(version.published_at || version.created_at)}</span>
                    <span>{formatBytes(version.size_bytes)}</span>
                    <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span>
                    <button
                      className="button button--ghost button--danger-text"
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
        </section>
      ) : null}
    </SkillSettingsFrame>
  );
}
