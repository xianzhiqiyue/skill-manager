import type { SettingsSection, SkillSettingsSection } from './routes';

export type SettingsNavItem = {
  current?: boolean;
  href: string;
  label: string;
  tone?: 'default' | 'danger';
};

export function buildSettingsPath(section: SettingsSection) {
  return `/settings/${section}`;
}

export function buildSkillSettingsPath(
  namespace: string,
  skillName: string,
  section: SkillSettingsSection = 'general',
) {
  return `/settings/skills/${encodeURIComponent(namespace)}/${encodeURIComponent(skillName)}/${section}`;
}

export function getAccountSettingsNav(section: SettingsSection): SettingsNavItem[] {
  return [
    {
      current: section === 'profile',
      href: buildSettingsPath('profile'),
      label: 'Profile',
    },
    {
      current: section === 'api-keys',
      href: buildSettingsPath('api-keys'),
      label: 'API Keys',
    },
  ];
}

export function getSkillSettingsNav(
  namespace: string,
  skillName: string,
  section: SkillSettingsSection,
): SettingsNavItem[] {
  return [
    {
      current: section === 'general',
      href: buildSkillSettingsPath(namespace, skillName, 'general'),
      label: 'General',
    },
    {
      current: section === 'versions',
      href: buildSkillSettingsPath(namespace, skillName, 'versions'),
      label: 'Versions',
    },
    {
      current: section === 'access',
      href: buildSkillSettingsPath(namespace, skillName, 'access'),
      label: 'Access',
    },
    {
      current: section === 'danger',
      href: buildSkillSettingsPath(namespace, skillName, 'danger'),
      label: 'Danger Zone',
      tone: 'danger',
    },
  ];
}

export function summarizeAPIKeyStatus(expiresAt?: string | null) {
  if (!expiresAt) {
    return { label: '可用', tone: 'success' as const };
  }

  const timestamp = new Date(expiresAt).getTime();
  if (Number.isNaN(timestamp)) {
    return { label: '可用', tone: 'neutral' as const };
  }

  const delta = timestamp - Date.now();
  if (delta <= 0) {
    return { label: '已过期', tone: 'danger' as const };
  }

  if (delta <= 7 * 24 * 60 * 60 * 1000) {
    return { label: '即将过期', tone: 'warning' as const };
  }

  return { label: '可用', tone: 'success' as const };
}

export function formatDateTimeLocalValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}
