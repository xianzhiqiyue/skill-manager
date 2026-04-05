import { APP_BASE_PATH, resolveAPIBase } from './basePath';
import skillTaxonomy from './generated/skillTaxonomy.json';

export const API_BASE = resolveAPIBase(
  typeof window !== 'undefined' ? window.location : null,
  import.meta.env.VITE_REGISTRY_BASE_URL,
  APP_BASE_PATH,
);

export type HealthResponse = {
  service: string;
  status: string;
  version: string;
};

export type AuthUser = {
  id: string;
  username: string;
  email: string;
  avatar_url?: string;
  is_active?: boolean;
  is_admin?: boolean;
  is_super_admin?: boolean;
  created_at?: string;
};

export type AuthResponse = {
  token: string;
  user: {
    id: string;
    username: string;
    email: string;
    is_admin?: boolean;
    is_super_admin?: boolean;
  };
};

export type APIKeySummary = {
  id: string;
  name: string;
  prefix: string;
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
};

export type APIKeyCreateResponse = APIKeySummary & {
  key: string;
};

export type SkillSummary = {
  id: string;
  namespace: string;
  name: string;
  owner_id?: string;
  description?: string;
  description_zh?: string;
  category?: string;
  tags?: string[];
  license?: string;
  download_count: number;
  rating_count: number;
  rating?: number;
  latest_version?: string;
  download_url?: string;
  is_public?: boolean;
  is_deprecated?: boolean;
  is_recommended?: boolean;
  created_at?: string;
  updated_at?: string;
};

export type SkillVersion = {
  id: string;
  version: string;
  size_bytes: number;
  scan_status?: string;
  published_at?: string;
  created_at?: string;
};

export type CommunityTagSummary = {
  tag: string;
  count: number;
};

export type SkillRating = {
  id: string;
  skill_id: string;
  user_id: string;
  rating: number;
  comment?: string;
  created_at?: string;
  updated_at?: string;
};

export type SkillDetail = SkillSummary & {
  tags?: string[];
  community_tags?: CommunityTagSummary[];
  viewer_tags?: string[];
  user_rating?: SkillRating;
  owner?: {
    username?: string;
    email?: string;
  };
  versions?: SkillVersion[];
};

type SkillLike = SkillSummary | SkillDetail;

export type SkillListResponse = {
  total?: number;
  page?: number;
  per_page?: number;
  results: SkillSummary[];
};

export type FetchSkillsParams = {
  query?: string;
  namespace?: string;
  tag?: string;
  license?: string;
  sort?: 'downloads' | 'updated' | 'rating' | 'name';
  perPage?: number;
};

export type PublishPayload = {
  namespace: string;
  name: string;
  description: string;
  descriptionZh: string;
  category: string;
  version: string;
  license: string;
  tags: string[];
  isPublic: boolean;
  archive: File;
};

export type PublishResponse = {
  namespace: string;
  name: string;
  version: string;
  download_url: string;
  published_at?: string;
};

export type UpdateSkillPayload = {
  description: string;
  descriptionZh: string;
  category: string;
  tags: string[];
  license: string;
  isPublic: boolean;
  isDeprecated: boolean;
};

export type UpdateSkillRecommendationPayload = {
  isRecommended: boolean;
};

export type SkillCategoryOption = {
  id: string;
  label: string;
  description: string;
};

export type OfficialTagOption = {
  id: string;
  description: string;
};

type SkillTaxonomyDefinition = {
  categories: SkillCategoryOption[];
  official_tags: OfficialTagOption[];
  aliases: Record<string, string>;
};

const taxonomyDefinition = skillTaxonomy as SkillTaxonomyDefinition;

export const SKILL_CATEGORIES = taxonomyDefinition.categories;
export const OFFICIAL_TAGS = taxonomyDefinition.official_tags;

export function normalizeOfficialTags(tags: string[]) {
  return Array.from(
    new Set(
      tags
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  );
}

export function toggleOfficialTag(tags: string[], tag: string) {
  const normalized = normalizeOfficialTags(tags);
  if (normalized.includes(tag)) {
    return normalized.filter((item) => item !== tag);
  }
  if (normalized.length >= 4) {
    return normalized;
  }
  return [...normalized, tag];
}

export function validateOfficialMetadataInput(category: string, tags: string[]) {
  if (!category.trim()) {
    return '请选择一级分类。';
  }

  const normalizedTags = normalizeOfficialTags(tags);
  if (normalizedTags.length === 0) {
    return '至少选择 1 个官方标签。';
  }
  if (normalizedTags.length > 4) {
    return '最多选择 4 个官方标签。';
  }

  return null;
}

export type MessageResponse = {
  message: string;
};

export type CreateAPIKeyPayload = {
  name: string;
  expiresAt?: string;
};

export type CommunityTagPayload = {
  tag: string;
};

export type RateSkillPayload = {
  rating: number;
  comment?: string;
};

export type RateSkillResponse = {
  skill: SkillDetail;
  user_rating: SkillRating;
};

type RequestOptions = {
  method?: string;
  headers?: HeadersInit;
  body?: BodyInit | null;
  token?: string | null;
};

function buildUrl(path: string, params?: URLSearchParams) {
  const base = API_BASE.replace(/\/$/, '');
  const suffix = params && [...params.keys()].length > 0 ? `?${params}` : '';
  return `${base}${path}${suffix}`;
}

async function request<T>(
  path: string,
  params?: URLSearchParams,
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.token) {
    headers.set('Authorization', `Bearer ${options.token}`);
  }

  const response = await fetch(buildUrl(path, params), {
    method: options.method || 'GET',
    headers,
    body: options.body,
  });
  if (!response.ok) {
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      const errorBody = (await response.json()) as {
        message?: string;
        code?: string;
      };
      throw new Error(errorBody.message || errorBody.code || `Request failed with status ${response.status}`);
    }

    throw new Error(`Request failed with status ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export function fetchHealth() {
  return request<HealthResponse>('/health');
}

export function fetchSkills(filters: FetchSkillsParams = {}) {
  const params = new URLSearchParams({
    page: '1',
    per_page: String(filters.perPage || 100),
  });

  if (filters.namespace && filters.namespace !== 'all') {
    params.set('namespace', filters.namespace);
  }

  if (filters.tag && filters.tag !== 'all') {
    params.set('tag', filters.tag);
  }

  if (filters.license && filters.license !== 'all') {
    params.set('license', filters.license);
  }

  if (filters.sort && filters.sort !== 'downloads') {
    params.set('sort', filters.sort);
  }

  if (filters.query?.trim()) {
    params.set('q', filters.query.trim());
    return request<SkillListResponse>('/api/v1/search', params);
  }

  return request<SkillListResponse>('/api/v1/skills', params);
}

export function fetchSkillDetail(namespace: string, name: string, token?: string) {
  return request<SkillDetail>(`/api/v1/skills/${namespace}/${name}`, undefined, {
    token,
  });
}

function getPrimaryVersion(skill: SkillLike) {
  if (skill.latest_version) {
    return skill.latest_version;
  }

  if ('versions' in skill && skill.versions?.length) {
    return skill.versions[0]?.version;
  }

  return undefined;
}

function isAbsoluteDownloadUrl(downloadUrl: string) {
  return /^https?:\/\//i.test(downloadUrl);
}

export function resolveDownloadUrl(downloadUrl?: string) {
  if (!downloadUrl) {
    return undefined;
  }

  if (isAbsoluteDownloadUrl(downloadUrl)) {
    return downloadUrl;
  }

  return `${API_BASE}${downloadUrl}`;
}

export function getSkillDescription(
  skill?: Pick<SkillSummary, 'description' | 'description_zh'> | null,
) {
  return skill?.description_zh?.trim() || skill?.description?.trim() || '';
}

export function getDownloadUrl(skill: SkillLike) {
  const downloadUrl = resolveDownloadUrl(skill.download_url);
  if (downloadUrl) {
    return downloadUrl;
  }

  const version = getPrimaryVersion(skill) || 'latest';
  return buildUrl(
    `/api/v1/download/${skill.namespace}/${skill.name}/${version}`,
    new URLSearchParams({ format: 'zip' }),
  );
}

export function registerUser(payload: {
  username: string;
  email: string;
  password: string;
}) {
  return request<AuthResponse>('/api/v1/auth/register', undefined, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });
}

export function loginUser(payload: { email: string; password: string }) {
  return request<AuthResponse>('/api/v1/auth/login', undefined, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });
}

export function fetchCurrentUser(token: string) {
  return request<AuthUser>('/api/v1/user', undefined, {
    token,
  });
}

export function fetchMySkills(token: string) {
  return request<SkillSummary[]>('/api/v1/user/skills', undefined, {
    token,
  });
}

export function fetchMyAPIKeys(token: string) {
  return request<APIKeySummary[]>('/api/v1/user/api-keys', undefined, {
    token,
  });
}

export function createAPIKey(token: string, payload: CreateAPIKeyPayload) {
  return request<APIKeyCreateResponse>('/api/v1/user/api-keys', undefined, {
    method: 'POST',
    token,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      name: payload.name,
      expires_at: payload.expiresAt,
    }),
  });
}

export function revokeAPIKey(token: string, id: string) {
  return request<MessageResponse>(`/api/v1/user/api-keys/${id}`, undefined, {
    method: 'DELETE',
    token,
  });
}

export function addCommunityTag(
  token: string,
  namespace: string,
  name: string,
  payload: CommunityTagPayload,
) {
  return request<SkillDetail>(`/api/v1/skills/${namespace}/${name}/community-tags`, undefined, {
    method: 'POST',
    token,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });
}

export function removeCommunityTag(token: string, namespace: string, name: string, tag: string) {
  return request<SkillDetail>(
    `/api/v1/skills/${namespace}/${name}/community-tags/${encodeURIComponent(tag)}`,
    undefined,
    {
      method: 'DELETE',
      token,
    },
  );
}

export function rateSkill(token: string, namespace: string, name: string, payload: RateSkillPayload) {
  return request<RateSkillResponse>(`/api/v1/skills/${namespace}/${name}/rating`, undefined, {
    method: 'POST',
    token,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });
}

export function publishSkill(token: string, payload: PublishPayload) {
  const form = new FormData();
  form.append('namespace', payload.namespace);
  form.append('name', payload.name);
  form.append('description', payload.description);
  form.append('description_zh', payload.descriptionZh);
  form.append('category', payload.category);
  form.append('version', payload.version);
  form.append('license', payload.license);
  form.append('tags', payload.tags.join(','));
  form.append('is_public', payload.isPublic ? 'true' : 'false');
  form.append('skill', payload.archive);

  return request<PublishResponse>('/api/v1/skills', undefined, {
    method: 'POST',
    token,
    body: form,
  });
}

export function updateSkill(
  token: string,
  namespace: string,
  name: string,
  payload: UpdateSkillPayload,
) {
  return request<SkillDetail>(`/api/v1/skills/${namespace}/${name}`, undefined, {
    method: 'PUT',
    token,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      description: payload.description,
      description_zh: payload.descriptionZh,
      category: payload.category,
      tags: payload.tags,
      license: payload.license,
      is_public: payload.isPublic,
      is_deprecated: payload.isDeprecated,
    }),
  });
}

export function updateSkillRecommendation(
  token: string,
  namespace: string,
  name: string,
  payload: UpdateSkillRecommendationPayload,
) {
  return request<SkillSummary>(`/api/v1/admin/skills/${namespace}/${name}/recommendation`, undefined, {
    method: 'PATCH',
    token,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      is_recommended: payload.isRecommended,
    }),
  });
}

export function deleteSkill(token: string, namespace: string, name: string) {
  return request<MessageResponse>(`/api/v1/skills/${namespace}/${name}`, undefined, {
    method: 'DELETE',
    token,
  });
}

export function deleteSkillVersion(
  token: string,
  namespace: string,
  name: string,
  version: string,
) {
  return request<MessageResponse>(
    `/api/v1/skills/${namespace}/${name}/versions/${version}`,
    undefined,
    {
      method: 'DELETE',
      token,
    },
  );
}
