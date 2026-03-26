import { useRegistryApp } from '../../hooks/useRegistryApp';
import { SkillSettingsFrame } from './SkillSettingsFrame';
import { skillKey } from '../../lib/format';

type AppModel = ReturnType<typeof useRegistryApp>;

type SkillSettingsPageProps = {
  model: AppModel;
  navigate: (path: string) => void;
  namespace: string;
  skillName: string;
};

export function SkillGeneralSettingsPage({
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
      description="编辑 skill 的描述、License 和 tags。"
      model={model}
      namespace={namespace}
      navigate={navigate}
      section="general"
      skillName={skillName}
      title="General"
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
                <h2>General</h2>
                <p>{skill.namespace} / {skill.name}</p>
              </div>
            </div>

            <label className="field">
              <span>描述</span>
              <textarea
                placeholder="补充 skill 的用途、边界和适用场景"
                rows={5}
                value={model.manageForm.description}
                onChange={(event) =>
                  model.setManageForm((current) => ({
                    ...current,
                    description: event.target.value,
                  }))
                }
              />
            </label>

            <div className="form-grid-two">
              <label className="field">
                <span>License</span>
                <input
                  placeholder="MIT"
                  value={model.manageForm.license}
                  onChange={(event) =>
                    model.setManageForm((current) => ({
                      ...current,
                      license: event.target.value,
                    }))
                  }
                />
              </label>

              <label className="field">
                <span>Tags</span>
                <input
                  placeholder="review, codex, registry"
                  value={model.manageForm.tags}
                  onChange={(event) =>
                    model.setManageForm((current) => ({
                      ...current,
                      tags: event.target.value,
                    }))
                  }
                />
              </label>
            </div>
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
