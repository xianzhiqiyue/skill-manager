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
  const currentUser = model.currentUser;
  const canManageOwnedSettings = Boolean(
    currentUser &&
      skill &&
      (currentUser.is_super_admin || (skill.owner_id && currentUser.id === skill.owner_id)),
  );
  const canManageRecommendation = Boolean(currentUser?.is_admin || currentUser?.is_super_admin);
  const recommendationRequiresPublic = skill?.is_public === false;
  const canSaveRecommendation = !recommendationRequiresPublic || !model.manageForm.isRecommended;

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
        <div className="gh-settings-stack">
          {canManageOwnedSettings ? (
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
                  {model.manageSaving ? '保存中...' : 'Save access changes'}
                </button>
              </div>
            </form>
          ) : (
            <section className="gh-settings-card">
              <div className="gh-settings-card__header">
                <div>
                  <h2>Access</h2>
                  <p>只有 skill 所有者和超级管理员可以修改可见性与弃用状态。</p>
                </div>
              </div>
              <div className="empty-panel">当前账号只能管理推荐状态，不能修改这个 skill 的访问控制。</div>
            </section>
          )}

          {canManageRecommendation ? (
            <form
              className="gh-settings-stack"
              onSubmit={(event) => {
                event.preventDefault();
                void model.submitManageRecommendation?.();
              }}
            >
              <section className="gh-settings-card">
                <div className="gh-settings-card__header">
                  <div>
                    <h2>Recommendation</h2>
                    <p>管理员和超级管理员可以把 skill 固定到技能中心结果顶部。</p>
                  </div>
                </div>

                <label className="checkbox-field">
                  <input
                    checked={model.manageForm.isRecommended}
                    disabled={recommendationRequiresPublic && !model.manageForm.isRecommended}
                    onChange={(event) =>
                      model.setManageForm((current) => ({
                        ...current,
                        isRecommended: event.target.checked,
                      }))
                    }
                    type="checkbox"
                  />
                  <span>在技能中心中优先推荐这个 skill</span>
                </label>

                <p className="gh-community-tag-copy">
                  {recommendationRequiresPublic
                    ? '只有公开 skill 才能被推荐；如果转为私有会自动从推荐列表移除。'
                    : '推荐 skill 会在技能中心和搜索结果中优先展示。'}
                </p>
              </section>

              <div className="gh-settings-actions">
                <button
                  className="button button--primary"
                  disabled={model.manageRecommendationSaving || !canSaveRecommendation}
                  type="submit"
                >
                  {model.manageRecommendationSaving ? '保存中...' : 'Save recommendation'}
                </button>
              </div>
            </form>
          ) : null}
        </div>
      ) : null}
    </SkillSettingsFrame>
  );
}
