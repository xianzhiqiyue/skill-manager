export type SkillTab = 'overview' | 'versions' | 'install' | 'activity';

export type SkillTabItem = {
  id: SkillTab;
  label: string;
};

export const skillTabs: SkillTabItem[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'versions', label: 'Versions' },
  { id: 'install', label: 'Install' },
  { id: 'activity', label: 'Activity' },
];

export function buildSkillTabPath(
  namespace: string,
  skillName: string,
  tab: SkillTab,
  search = '',
) {
  const base = `/skills/${encodeURIComponent(namespace)}/${encodeURIComponent(skillName)}`;
  const path = tab === 'overview' ? base : `${base}/${tab}`;
  return `${path}${search}`;
}
