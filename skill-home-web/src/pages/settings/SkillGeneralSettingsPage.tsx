import {
  OFFICIAL_TAGS,
  SKILL_CATEGORIES,
  toggleOfficialTag,
  validateOfficialMetadataInput,
} from '../../api';
import { SkillTagButton } from '../../components/object/SkillTagButton';
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

export function SkillGeneralSettingsPage({
  model,
  navigate,
  namespace,
  skillName,
}: SkillSettingsPageProps) {
  const targetKey = `${namespace}/${skillName}`;
  const skill =
    model.managedSkill && skillKey(model.managedSkill) === targetKey ? model.managedSkill : null;
  const metadataError = validateOfficialMetadataInput(
    model.manageForm.category,
    model.manageForm.tags,
  );

  return (
    <SkillSettingsFrame
      description="编辑 skill 的描述、官方分类和 License。"
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

            <label className="field">
              <span>中文描述</span>
              <textarea
                placeholder="可选。填写后目录和详情页会优先展示中文描述"
                rows={4}
                value={model.manageForm.descriptionZh}
                onChange={(event) =>
                  model.setManageForm((current) => ({
                    ...current,
                    descriptionZh: event.target.value,
                  }))
                }
              />
            </label>

            <div className="form-grid-two">
              <label className="field">
                <span>Category</span>
                <select
                  aria-label="Category"
                  value={model.manageForm.category}
                  onChange={(event) =>
                    model.setManageForm((current) => ({
                      ...current,
                      category: event.target.value,
                    }))
                  }
                >
                  <option value="">请选择一级分类</option>
                  {SKILL_CATEGORIES.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.label} · {item.description}
                    </option>
                  ))}
                </select>
              </label>

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
            </div>

            <div className="field">
              <span>Official tags</span>
              <p className="gh-community-tag-copy">
                选择 1 到 4 个官方标签，保持目录分类和筛选口径一致。
              </p>
              <div className="skill-tag-row skill-tag-row--dense">
                {OFFICIAL_TAGS.map((item) => (
                  <SkillTagButton
                    key={item.id}
                    onSelect={(tag) =>
                      model.setManageForm((current) => ({
                        ...current,
                        tags: toggleOfficialTag(current.tags, tag),
                      }))
                    }
                    selected={model.manageForm.tags.includes(item.id)}
                    tag={item.id}
                  />
                ))}
              </div>
              <p className="gh-community-tag-copy">
                已选 {model.manageForm.tags.length} / 4
                {metadataError ? ` · ${metadataError}` : ''}
              </p>
            </div>
          </section>

          <div className="gh-settings-actions">
            <button
              className="button button--primary"
              disabled={model.manageSaving || Boolean(metadataError)}
              type="submit"
            >
              {model.manageSaving ? '保存中...' : 'Save changes'}
            </button>
          </div>
        </form>
      ) : null}
    </SkillSettingsFrame>
  );
}
