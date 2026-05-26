export type ConsoleSection = 'skills' | 'api-keys';
export type SkillTab = 'overview' | 'versions' | 'install' | 'activity';
export type SettingsSection = 'profile' | 'stats' | 'api-keys' | 'users';
export type SkillSettingsSection = 'general' | 'versions' | 'access' | 'danger';

export type AppRoute =
  | { name: 'home' }
  | { name: 'skills' }
  | { name: 'skill-tab'; namespace: string; skillName: string; tab: SkillTab }
  | { name: 'settings'; section: SettingsSection }
  | { name: 'skill-settings'; namespace: string; skillName: string; section: SkillSettingsSection }
  | { name: 'publish-new' }
  | { name: 'install' }
  | { name: 'auth'; mode: 'login' | 'register' };

function safeDecode(segment: string) {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

export function buildSkillPath(namespace: string, skillName: string) {
  return `/skills/${encodeURIComponent(namespace)}/${encodeURIComponent(skillName)}`;
}

export function buildAuthPath(mode: 'login' | 'register', redirectTo?: string) {
  const path = mode === 'register' ? '/register' : '/login';
  if (!redirectTo) {
    return path;
  }

  return `${path}?redirect=${encodeURIComponent(redirectTo)}`;
}

export function parseAuthRedirect(search: string) {
  const redirect = new URLSearchParams(search).get('redirect');
  if (!redirect || !redirect.startsWith('/') || redirect.startsWith('//')) {
    return null;
  }

  return redirect;
}

export function parseRoute(pathname: string): AppRoute {
  const normalized = pathname.replace(/\/+$/, '') || '/';
  const segments = normalized.split('/').filter(Boolean).map(safeDecode);

  if (!segments.length) {
    return { name: 'home' };
  }

  if (segments[0] === 'skills' && segments.length >= 3) {
    const tab = (segments[3] || 'overview') as SkillTab;
    return {
      name: 'skill-tab',
      namespace: segments[1],
      skillName: segments[2],
      tab: tab === 'versions' || tab === 'install' || tab === 'activity' ? tab : 'overview',
    };
  }

  if (segments[0] === 'skills') {
    return { name: 'skills' };
  }

  if (segments[0] === 'settings') {
    if (segments[1] === 'skills' && segments.length >= 5) {
      const section = segments[4] as SkillSettingsSection;
      return {
        name: 'skill-settings',
        namespace: segments[2],
        skillName: segments[3],
        section:
          section === 'versions' || section === 'access' || section === 'danger'
            ? section
            : 'general',
      };
    }

    if (segments[1] === 'skills' && segments.length >= 4) {
      return {
        name: 'skill-settings',
        namespace: segments[2],
        skillName: segments[3],
        section: 'general',
      };
    }

    return {
      name: 'settings',
      section:
        segments[1] === 'api-keys'
          ? 'api-keys'
          : segments[1] === 'stats'
            ? 'stats'
          : segments[1] === 'users'
            ? 'users'
            : 'profile',
    };
  }

  if (segments[0] === 'publish') {
    return { name: 'publish-new' };
  }

  if (segments[0] === 'console') {
    if (segments[1] === 'api-keys') {
      return { name: 'settings', section: 'api-keys' };
    }

    return { name: 'settings', section: 'profile' };
  }

  if (segments[0] === 'install') {
    return { name: 'install' };
  }

  if (segments[0] === 'login') {
    return { name: 'auth', mode: 'login' };
  }

  if (segments[0] === 'register') {
    return { name: 'auth', mode: 'register' };
  }

  return { name: 'home' };
}

export function routeMatches(route: AppRoute, target: AppRoute['name']) {
  if (target === 'skills') {
    return route.name === 'skills' || route.name === 'skill-tab';
  }

  if (target === 'settings') {
    return route.name === 'settings' || route.name === 'skill-settings';
  }

  if (target === 'auth') {
    return route.name === 'auth';
  }
  return route.name === target;
}
