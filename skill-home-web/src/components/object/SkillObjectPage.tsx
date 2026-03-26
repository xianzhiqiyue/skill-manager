import type { ReactNode } from 'react';

import type { SkillDetail } from '../../api';
import type { SkillTab } from '../../lib/skillTabs';
import { SidebarLayout } from '../layout/SidebarLayout';
import { SkillHeader } from './SkillHeader';
import { SkillObjectMetadata } from './SkillObjectMetadata';
import { SkillObjectTabs } from './SkillObjectTabs';

export type SkillObjectPageModel = {
  detailError: string | null;
  detailLoading: boolean;
  detailSkill: SkillDetail | null;
  returnToCatalog?: () => void;
};

type SkillObjectPageProps = {
  activeTab: SkillTab;
  children: (skill: SkillDetail) => ReactNode;
  model: SkillObjectPageModel;
  navigate: (path: string) => void;
  search?: string;
};

export function SkillObjectPage({
  activeTab,
  children,
  model,
  navigate,
  search = '',
}: SkillObjectPageProps) {
  if (model.detailLoading) {
    return <div className="empty-panel">正在读取技能详情...</div>;
  }

  if (model.detailError) {
    return <div className="empty-panel empty-panel--danger">读取详情失败：{model.detailError}</div>;
  }

  if (!model.detailSkill) {
    return <div className="empty-panel">没有找到对应的技能。</div>;
  }

  const skill = model.detailSkill;
  const panelId = `skill-panel-${activeTab}`;
  const handleTagSelect = (tag: string) => {
    navigate(`/skills?tag=${encodeURIComponent(tag)}`);
  };

  return (
    <div className="page-stack gh-object-page">
      <section className="surface-panel gh-object-shell">
        <SkillHeader onBack={model.returnToCatalog} onTagSelect={handleTagSelect} skill={skill} />
        <SkillObjectTabs
          activeTab={activeTab}
          navigate={navigate}
          namespace={skill.namespace}
          search={search}
          skillName={skill.name}
        />

        <SidebarLayout
          aside={<SkillObjectMetadata skill={skill} />}
          className="gh-sidebar-layout--object"
          content={(
            <section className="gh-object-panel" id={panelId}>
              {children(skill)}
            </section>
          )}
        />
      </section>
    </div>
  );
}
