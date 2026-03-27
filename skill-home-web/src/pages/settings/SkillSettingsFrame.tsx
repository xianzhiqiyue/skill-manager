import type { ReactNode } from 'react';

import { SettingsAuthCallout } from '../../components/settings/SettingsAuthCallout';
import { SettingsLayout } from '../../components/settings/SettingsLayout';
import { useRegistryApp } from '../../hooks/useRegistryApp';
import { skillKey, skillRef, summarizeScanStatus } from '../../lib/format';
import { buildSkillSettingsPath, getSkillSettingsNav } from '../../lib/settings';
import type { SkillSettingsSection } from '../../lib/routes';

type AppModel = ReturnType<typeof useRegistryApp>;

type SkillSettingsFrameProps = {
  children: ReactNode;
  description: string;
  model: AppModel;
  navigate: (path: string) => void;
  namespace: string;
  section: SkillSettingsSection;
  skillName: string;
  title: string;
};

function renderStatusBanner(tone: 'danger' | 'success', message: string) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

export function SkillSettingsFrame({
  children,
  description,
  model,
  navigate,
  namespace,
  section,
  skillName,
  title,
}: SkillSettingsFrameProps) {
  if (!model.token) {
    return (
      <SettingsAuthCallout
        description="登录后查看和修改 skill 的对象级设置，包括版本、可见性和危险操作。"
        navigate={navigate}
        redirectTo={buildSkillSettingsPath(namespace, skillName, section)}
        title="Skill settings"
      />
    );
  }

  const targetKey = `${namespace}/${skillName}`;
  const skill =
    model.managedSkill && skillKey(model.managedSkill) === targetKey ? model.managedSkill : null;
  const latestVersion = skill?.versions?.find((version) => version.version === skill.latest_version);
  const state = summarizeScanStatus(latestVersion?.scan_status || skill?.versions?.[0]?.scan_status);
  const isTransitioning = model.managedSkillKey !== targetKey || model.manageLoading;

  return (
    <SettingsLayout
      description={description}
      meta={skill ? <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span> : undefined}
      navAriaLabel="Skill settings"
      navItems={getSkillSettingsNav(namespace, skillName, section)}
      onNavigate={navigate}
      sidebarHeader={(
        <div className="gh-settings-sidebar__scope">
          <strong>{skill?.name || skillName}</strong>
          <span>{skill ? skillRef(skill) : `@${namespace}/${skillName}`}</span>
        </div>
      )}
      title={title}
    >
      <div className="gh-settings-stack">
        {model.manageError && skill ? renderStatusBanner('danger', model.manageError) : null}
        {model.manageSuccess ? renderStatusBanner('success', model.manageSuccess) : null}

        {isTransitioning ? (
          <div className="empty-panel">正在读取技能设置...</div>
        ) : skill ? (
          children
        ) : model.manageError ? (
          <div className="empty-panel empty-panel--danger">读取技能设置失败：{model.manageError}</div>
        ) : (
          <div className="empty-panel">没有找到对应的技能。</div>
        )}
      </div>
    </SettingsLayout>
  );
}
