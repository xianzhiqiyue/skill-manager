export type AppRoute =
  | { name: 'home' }
  | { name: 'skills' }
  | { name: 'skill'; namespace: string; skillName: string }
  | { name: 'publish' }
  | { name: 'console' }
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

export function parseRoute(pathname: string): AppRoute {
  const normalized = pathname.replace(/\/+$/, '') || '/';
  const segments = normalized.split('/').filter(Boolean).map(safeDecode);

  if (!segments.length) {
    return { name: 'home' };
  }

  if (segments[0] === 'skills' && segments.length >= 3) {
    return {
      name: 'skill',
      namespace: segments[1],
      skillName: segments[2],
    };
  }

  if (segments[0] === 'skills') {
    return { name: 'skills' };
  }

  if (segments[0] === 'publish') {
    return { name: 'publish' };
  }

  if (segments[0] === 'console') {
    return { name: 'console' };
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
  if (target === 'auth') {
    return route.name === 'auth';
  }
  return route.name === target;
}
