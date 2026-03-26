import { DangerZone } from '../../components/settings/DangerZone';
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

export function SkillDangerSettingsPage({
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
      description="删除 skill 属于不可逆操作，单独隔离在 Danger Zone 页面。"
      model={model}
      namespace={namespace}
      navigate={navigate}
      section="danger"
      skillName={skillName}
      title="Danger Zone"
    >
      {skill ? (
        <DangerZone
          actionLabel="Delete skill"
          description="删除 skill 会同时移除元信息和版本记录。这个动作不可撤销。"
          disabled={model.manageDeletingSkill}
          onAction={() => void model.removeManagedSkill()}
          pendingLabel="Deleting..."
          title="Delete this skill"
        >
          <p className="gh-danger-zone__warning">
            目标对象：<strong>@{skill.namespace}/{skill.name}</strong>
          </p>
        </DangerZone>
      ) : null}
    </SkillSettingsFrame>
  );
}
