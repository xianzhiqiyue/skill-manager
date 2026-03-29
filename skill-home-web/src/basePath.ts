const fallbackAPIBase = 'https://soulstore.ciqtek.com/skill-home';

export type LocationLike = {
  hostname: string;
  origin: string;
};

export function normalizeBasePath(value?: string | null) {
  const trimmed = value?.trim() || '';
  if (!trimmed || trimmed === '/') {
    return '';
  }

  return `/${trimmed.replace(/^\/+|\/+$/g, '')}`;
}

export function normalizeViteBasePath(value?: string | null) {
  const normalized = normalizeBasePath(value);
  return normalized ? `${normalized}/` : '/';
}

export function applyBasePath(path: string, basePath: string) {
  const normalizedBasePath = normalizeBasePath(basePath);
  const normalizedPath = !path || path === '/' ? '/' : `/${path.replace(/^\/+/, '')}`;

  if (!normalizedBasePath) {
    return normalizedPath;
  }

  return normalizedPath === '/'
    ? `${normalizedBasePath}/`
    : `${normalizedBasePath}${normalizedPath}`;
}

export function stripBasePath(pathname: string, basePath: string) {
  const normalizedPathname = pathname || '/';
  const normalizedBasePath = normalizeBasePath(basePath);

  if (!normalizedBasePath) {
    return normalizedPathname;
  }

  if (normalizedPathname === normalizedBasePath || normalizedPathname === `${normalizedBasePath}/`) {
    return '/';
  }

  if (normalizedPathname.startsWith(`${normalizedBasePath}/`)) {
    return normalizedPathname.slice(normalizedBasePath.length) || '/';
  }

  return normalizedPathname;
}

export function resolveAPIBase(
  locationLike?: LocationLike | null,
  explicitBaseURL?: string | null,
  appBasePath?: string | null,
) {
  const explicit = explicitBaseURL?.trim();
  if (explicit) {
    return explicit.replace(/\/$/, '');
  }

  const normalizedBasePath = normalizeBasePath(appBasePath);
  const isLocalhost =
    !locationLike ||
    locationLike.hostname === '127.0.0.1' ||
    locationLike.hostname === 'localhost';

  if (isLocalhost) {
    return fallbackAPIBase;
  }

  return `${locationLike.origin}${normalizedBasePath}`.replace(/\/$/, '');
}

const runtimeBaseURL =
  typeof import.meta !== 'undefined' && import.meta.env
    ? import.meta.env.BASE_URL
    : undefined;

export const APP_BASE_PATH = normalizeBasePath(runtimeBaseURL);
