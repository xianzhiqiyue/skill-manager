import { buildSkillTabPath, skillTabs, type SkillTab } from '../../lib/skillTabs';

type SkillObjectTabsProps = {
  activeTab: SkillTab;
  namespace: string;
  navigate: (path: string) => void;
  search?: string;
  skillName: string;
};

export function SkillObjectTabs({
  activeTab,
  namespace,
  navigate,
  search = '',
  skillName,
}: SkillObjectTabsProps) {
  return (
    <nav aria-label="Skill tabs" className="gh-object-tabs">
      {skillTabs.map((tab) => {
        const active = tab.id === activeTab;
        const path = buildSkillTabPath(namespace, skillName, tab.id, search);

        return (
          <a
            aria-current={active ? 'page' : undefined}
            className={`gh-object-tab ${active ? 'is-active' : ''}`}
            href={path}
            key={tab.id}
            onClick={(event) => {
              event.preventDefault();
              navigate(path);
            }}
          >
            {tab.label}
          </a>
        );
      })}
    </nav>
  );
}
