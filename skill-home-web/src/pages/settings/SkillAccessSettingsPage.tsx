import { useRegistryApp } from '../../hooks/useRegistryApp';
import { skillKey } from '../../lib/format';
import { SkillSettingsFrame } from './SkillSettingsFrame';

type AppModel = ReturnType<typeof useRegistryApp>;

type SkillSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
  namespace: string;
  skillName: string;
};

export function SkillAccessSettingsPage({
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
      description="控制目录可见性和弃用状态，把访问策略从基础信息里剥离出来。"
      model={model}
      namespace={namespace}
      navigate={navigate}
      section="access"
      skillName={skillName}
      title="Access"
    >
      {skill ? (
        <form
          className="gh-settings-stack"
          onSubmit={(event) => {
            event.preventDefault();
            void model.submitManage();
          }}
        >
          <section className="gh-settings-card">
            <div className="gh-settings-card__header">
              <div>
                <h2>Access</h2>
                <p>目录曝光和弃用状态集中在这一页管理。</p>
              </div>
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
          </section>

          <div className="gh-settings-actions">
            <button className="button button--primary" disabled={model.manageSaving} type="submit">
              {model.manageSaving ? '保存中...' : 'Save changes'}
            </button>
          </div>
        </form>
      ) : null}
    </SkillSettingsFrame>
  );
}
