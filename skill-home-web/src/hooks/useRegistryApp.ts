import { useEffect, useRef, useState } from 'react';

import { APP_BASE_PATH, stripBasePath } from '../basePath';
import {
  addCommunityTag,
  createAPIKey,
  deleteSkill,
  deleteSkillVersion,
  fetchAdminAuditLogs,
  fetchAdminUsers,
  fetchCurrentUser,
  fetchCurrentUserStats,
  fetchHealth,
  fetchMyAPIKeys,
  fetchMySkills,
  fetchSkillDetail,
  fetchSkills,
  getSkillDescription,
  likeSkill,
  loginUser,
  publishSkill,
  rateSkill,
  recordShareEvent,
  registerUser,
  removeCommunityTag,
  revokeAPIKey,
  unlikeSkill,
  updateAdminUser,
  updateCurrentUserPassword,
  updateCurrentUserProfile,
  updateSkill,
  updateSkillRecommendation,
  normalizeOfficialTags,
  type AdminUpdateUserPayload,
  type AdminUser,
  type AdminUserRoleFilter,
  type AdminUserStatusFilter,
  type APIKeyCreateResponse,
  type APIKeySummary,
  type AuditLog,
  type AuthResponse,
  type AuthUser,
  type FetchSkillsParams,
  type HealthResponse,
  type PublishResponse,
  type SkillDetail,
  type SkillSummary,
  type UserStats,
  validateOfficialMetadataInput,
} from '../api';
import {
  defaultCatalogFilters,
  filterCatalogSkills,
  parseCatalogSearch,
  toCatalogSearch,
  type CatalogFilters,
  type CatalogSort,
  type CatalogView,
} from '../lib/catalogState';
import { copyText, parseTags, skillKey } from '../lib/format';
import { buildAuthPath, buildSkillPath, parseAuthRedirect, type AppRoute } from '../lib/routes';

const TOKEN_STORAGE_KEY = 'skill-home-web-token';

function loadStoredToken() {
  if (typeof window === 'undefined') {
    return '';
  }
  return window.localStorage.getItem(TOKEN_STORAGE_KEY) || '';
}

function uniq(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

type APIKeyExpiryPreset = 'never' | '7d' | '30d' | '90d' | 'custom';

function getExpiresAtFromPreset(preset: APIKeyExpiryPreset) {
  if (preset === 'never') {
    return undefined;
  }

  if (preset === 'custom') {
    return undefined;
  }

  const days = preset === '7d' ? 7 : preset === '30d' ? 30 : 90;
  return new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();
}

function toISOStringFromLocalDateTime(value: string) {
  if (!value) {
    return undefined;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toISOString();
}

function sortRecommendedFirst(left: SkillSummary, right: SkillSummary) {
  return (
    Number(Boolean(right.is_recommended)) - Number(Boolean(left.is_recommended)) ||
    right.download_count - left.download_count ||
    new Date(right.updated_at || 0).getTime() - new Date(left.updated_at || 0).getTime() ||
    left.name.localeCompare(right.name)
  );
}

export function useRegistryApp(
  route: AppRoute,
  locationSearch: string,
  navigate: (path: string, options?: { replace?: boolean }) => void,
) {
  const catalogFilters =
    route.name === 'skills' || route.name === 'skill-tab'
      ? parseCatalogSearch(locationSearch)
      : defaultCatalogFilters;
  const normalizedCatalogSearch =
    route.name === 'skills' || route.name === 'skill-tab'
      ? toCatalogSearch(catalogFilters)
      : '';
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);

  const [skills, setSkills] = useState<SkillSummary[]>([]);
  const [skillsTotal, setSkillsTotal] = useState(0);
  const [skillsLoading, setSkillsLoading] = useState(true);
  const [skillsError, setSkillsError] = useState<string | null>(null);
  const [catalogNonce, setCatalogNonce] = useState(0);
  const [catalogSkills, setCatalogSkills] = useState<SkillSummary[]>([]);
  const [catalogTotal, setCatalogTotal] = useState(0);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [catalogSearchKey, setCatalogSearchKey] = useState('');
  const [catalogSearchNonce, setCatalogSearchNonce] = useState(0);
  const previousCatalogRouteRef = useRef(route.name);

  const routeSkillKey = route.name === 'skill-tab' ? `${route.namespace}/${route.skillName}` : null;

  const [detailSkill, setDetailSkill] = useState<SkillDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [detailNonce, setDetailNonce] = useState(0);
  const [communityTagSaving, setCommunityTagSaving] = useState(false);
  const [communityTagRemoving, setCommunityTagRemoving] = useState<string | null>(null);
  const [communityTagError, setCommunityTagError] = useState<string | null>(null);
  const [communityTagSuccess, setCommunityTagSuccess] = useState<string | null>(null);
  const [skillRatingSaving, setSkillRatingSaving] = useState(false);
  const [skillRatingError, setSkillRatingError] = useState<string | null>(null);
  const [skillRatingSuccess, setSkillRatingSuccess] = useState<string | null>(null);
  const [skillLikeSaving, setSkillLikeSaving] = useState(false);
  const [skillLikeError, setSkillLikeError] = useState<string | null>(null);
  const [skillShareStatus, setSkillShareStatus] = useState<string | null>(null);

  const [token, setToken] = useState(loadStoredToken);
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [authSuccess, setAuthSuccess] = useState<string | null>(null);
  const [authForm, setAuthForm] = useState({
    username: '',
    displayNameZh: '',
    email: '',
    password: '',
  });

  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [accountLoading, setAccountLoading] = useState(false);
  const [accountError, setAccountError] = useState<string | null>(null);
  const [accountNonce, setAccountNonce] = useState(0);
  const [userStats, setUserStats] = useState<UserStats | null>(null);
  const [profileForm, setProfileForm] = useState({
    displayNameZh: '',
    avatarUrl: '',
  });
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null);
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);
  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([]);
  const [adminUsersTotal, setAdminUsersTotal] = useState(0);
  const [adminUsersLoading, setAdminUsersLoading] = useState(false);
  const [adminUsersError, setAdminUsersError] = useState<string | null>(null);
  const [adminUsersSuccess, setAdminUsersSuccess] = useState<string | null>(null);
  const [adminUsersSaving, setAdminUsersSaving] = useState<string | null>(null);
  const [adminUsersNonce, setAdminUsersNonce] = useState(0);
  const [adminUserFilters, setAdminUserFilters] = useState({
    query: '',
    role: 'all' as AdminUserRoleFilter,
    status: 'all' as AdminUserStatusFilter,
    page: 1,
    perPage: 20,
  });
  const [adminUserPasswordDrafts, setAdminUserPasswordDrafts] = useState<Record<string, string>>({});
  const [adminAuditLogs, setAdminAuditLogs] = useState<AuditLog[]>([]);
  const [adminAuditLogsLoading, setAdminAuditLogsLoading] = useState(false);
  const [adminAuditLogsError, setAdminAuditLogsError] = useState<string | null>(null);

  const [mySkills, setMySkills] = useState<SkillSummary[]>([]);
  const [apiKeys, setAPIKeys] = useState<APIKeySummary[]>([]);
  const [apiKeysLoading, setAPIKeysLoading] = useState(false);
  const [apiKeysError, setAPIKeysError] = useState<string | null>(null);
  const [apiKeysSuccess, setAPIKeysSuccess] = useState<string | null>(null);
  const [apiKeysNonce, setAPIKeysNonce] = useState(0);
  const [apiKeyCreating, setAPIKeyCreating] = useState(false);
  const [apiKeyRevoking, setAPIKeyRevoking] = useState<string | null>(null);
  const [revealedAPIKey, setRevealedAPIKey] = useState<APIKeyCreateResponse | null>(null);
  const [apiKeyForm, setAPIKeyForm] = useState({
    name: '',
    expiryPreset: 'never' as APIKeyExpiryPreset,
    customExpiresAt: '',
  });
  const [managedSkillKey, setManagedSkillKey] = useState<string | null>(null);
  const [managedSkill, setManagedSkill] = useState<SkillDetail | null>(null);
  const [manageLoading, setManageLoading] = useState(false);
  const [manageSaving, setManageSaving] = useState(false);
  const [manageRecommendationSaving, setManageRecommendationSaving] = useState(false);
  const [manageDeletingSkill, setManageDeletingSkill] = useState(false);
  const [manageDeletingVersion, setManageDeletingVersion] = useState<string | null>(null);
  const [manageError, setManageError] = useState<string | null>(null);
  const [manageSuccess, setManageSuccess] = useState<string | null>(null);
  const [manageNonce, setManageNonce] = useState(0);
  const [manageForm, setManageForm] = useState({
    description: '',
    descriptionZh: '',
    category: '',
    license: 'MIT',
    tags: [] as string[],
    isPublic: true,
    isDeprecated: false,
    isRecommended: false,
  });

  const [publishLoading, setPublishLoading] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);
  const [publishSuccess, setPublishSuccess] = useState<PublishResponse | null>(null);
  const [publishForm, setPublishForm] = useState({
    namespace: '',
    name: '',
    description: '',
    descriptionZh: '',
    category: '',
    version: '0.1.0',
    license: 'MIT',
    tags: [] as string[],
    isPublic: true,
  });
  const [publishFile, setPublishFile] = useState<File | null>(null);

  const namespaceOptions = uniq(skills.map((skill) => skill.namespace)).sort();
  const tagOptions = uniq(skills.flatMap((skill) => skill.tags || [])).sort();
  const licenseOptions = uniq(skills.map((skill) => skill.license || '')).sort();
  const settingsSection = route.name === 'settings' ? route.section : null;

  const featuredSkills = [...skills]
    .sort(sortRecommendedFirst)
    .slice(0, 4);
  const latestSkills = [...skills]
    .sort(
      (left, right) =>
        new Date(right.updated_at || 0).getTime() -
        new Date(left.updated_at || 0).getTime(),
    )
    .slice(0, 6);

  const quickStats = {
    namespaceCount: namespaceOptions.length,
    licenseCount: licenseOptions.length,
    tagCount: tagOptions.length,
  };

  useEffect(() => {
    let disposed = false;

    fetchHealth()
      .then((data) => {
        if (!disposed) {
          setHealth(data);
        }
      })
      .catch((error: Error) => {
        if (!disposed) {
          setHealthError(error.message);
        }
      });

    return () => {
      disposed = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setSkillsLoading(true);
    setSkillsError(null);

    fetchSkills({
      perPage: 100,
    })
      .then((data) => {
        if (cancelled) {
          return;
        }

        setSkills(data.results);
        setSkillsTotal(data.total ?? data.results.length);
        setSkillsLoading(false);
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setSkillsError(error.message);
          setSkillsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [catalogNonce]);

  useEffect(() => {
    if (route.name !== 'skills') {
      previousCatalogRouteRef.current = route.name;
      setCatalogLoading(false);
      return;
    }

    const enteringCatalogFromOutside = previousCatalogRouteRef.current !== 'skills';
    previousCatalogRouteRef.current = route.name;

    let cancelled = false;
    if (enteringCatalogFromOutside && catalogSearchKey !== normalizedCatalogSearch) {
      setCatalogSkills([]);
      setCatalogTotal(0);
    }
    setCatalogLoading(true);
    setCatalogError(null);

    fetchSkills({
      query: catalogFilters.query,
      namespace: catalogFilters.namespace,
      tag: catalogFilters.tag,
      license: catalogFilters.license,
      sort: catalogFilters.sort,
      perPage: 100,
    })
      .then((data) => {
        if (cancelled) {
          return;
        }

        setCatalogSkills(data.results);
        setCatalogTotal(data.total ?? data.results.length);
        setCatalogSearchKey(normalizedCatalogSearch);
        setCatalogLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setCatalogError(error.message);
        setCatalogLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [
    catalogFilters.license,
    catalogFilters.namespace,
    catalogFilters.query,
    catalogFilters.sort,
    catalogFilters.tag,
    catalogSearchKey,
    catalogSearchNonce,
    normalizedCatalogSearch,
    route.name,
  ]);

  useEffect(() => {
    if (!routeSkillKey) {
      setDetailSkill(null);
      setDetailError(null);
      setDetailLoading(false);
      setCommunityTagError(null);
      setCommunityTagSuccess(null);
      setCommunityTagSaving(false);
      setCommunityTagRemoving(null);
      setSkillRatingError(null);
      setSkillRatingSuccess(null);
      setSkillRatingSaving(false);
      setSkillLikeSaving(false);
      setSkillLikeError(null);
      setSkillShareStatus(null);
      return;
    }

    const [namespace, name] = routeSkillKey.split('/');
    let cancelled = false;
    setDetailLoading(true);
    setDetailError(null);
    setSkillRatingError(null);
    setSkillRatingSuccess(null);
    setSkillLikeError(null);
    setSkillShareStatus(null);

    fetchSkillDetail(namespace, name, token || undefined)
      .then((data) => {
        if (!cancelled) {
          setDetailSkill(data);
          setDetailLoading(false);
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setDetailError(error.message);
          setDetailLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [detailNonce, routeSkillKey, token]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    if (token) {
      window.localStorage.setItem(TOKEN_STORAGE_KEY, token);
    } else {
      window.localStorage.removeItem(TOKEN_STORAGE_KEY);
    }
  }, [token]);

  useEffect(() => {
    if (!token) {
      setCurrentUser(null);
      setMySkills([]);
      setAPIKeys([]);
      setUserStats(null);
      setAccountError(null);
      setManagedSkillKey(null);
      setManagedSkill(null);
      setManageError(null);
      setManageSuccess(null);
      setManageRecommendationSaving(false);
      setProfileForm({ displayNameZh: '', avatarUrl: '' });
      setProfileError(null);
      setProfileSuccess(null);
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
      setPasswordError(null);
      setPasswordSuccess(null);
      setAdminUsers([]);
      setAdminUsersTotal(0);
      setAdminUsersLoading(false);
      setAdminUsersError(null);
      setAdminUsersSuccess(null);
      setAdminUsersSaving(null);
      setAdminUserPasswordDrafts({});
      setAdminAuditLogs([]);
      setAdminAuditLogsLoading(false);
      setAdminAuditLogsError(null);
      return;
    }

    let cancelled = false;
    setAccountLoading(true);
    setAccountError(null);

    Promise.all([fetchCurrentUser(token), fetchMySkills(token), fetchCurrentUserStats(token).catch(() => null)])
      .then(([user, ownedSkills, stats]) => {
        if (cancelled) {
          return;
        }

        setCurrentUser(user);
        setProfileForm({
          displayNameZh: user.display_name_zh || '',
          avatarUrl: user.avatar_url || '',
        });
        setMySkills(ownedSkills);
        setUserStats(stats);
        setAccountLoading(false);
        setPublishForm((current) => ({
          ...current,
          namespace: current.namespace || user.username,
        }));
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setAccountError(error.message);
        setAccountLoading(false);
        setCurrentUser(null);
        setMySkills([]);
        setToken('');
      });

    return () => {
      cancelled = true;
    };
  }, [accountNonce, token]);

  useEffect(() => {
    if (!token || !currentUser?.is_super_admin || settingsSection !== 'users') {
      setAdminUsersLoading(false);
      setAdminAuditLogsLoading(false);
      if (settingsSection !== 'users') {
        setAdminUsersError(null);
        setAdminAuditLogsError(null);
      }
      return;
    }

    let cancelled = false;
    setAdminUsersLoading(true);
    setAdminUsersError(null);

    fetchAdminUsers(token, {
      page: adminUserFilters.page,
      perPage: adminUserFilters.perPage,
      query: adminUserFilters.query,
      role: adminUserFilters.role,
      status: adminUserFilters.status,
    })
      .then((data) => {
        if (cancelled) {
          return;
        }

        setAdminUsers(data.results);
        setAdminUsersTotal(data.total);
        setAdminUsersLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setAdminUsersError(error.message);
        setAdminUsersLoading(false);
      });

    setAdminAuditLogsLoading(true);
    setAdminAuditLogsError(null);
    fetchAdminAuditLogs(token, { page: 1, perPage: 8 })
      .then((data) => {
        if (cancelled) {
          return;
        }

        setAdminAuditLogs(data.results);
        setAdminAuditLogsLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setAdminAuditLogsError(error.message);
        setAdminAuditLogsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [
    adminUserFilters.page,
    adminUserFilters.perPage,
    adminUserFilters.query,
    adminUserFilters.role,
    adminUserFilters.status,
    adminUsersNonce,
    currentUser?.is_super_admin,
    settingsSection,
    token,
  ]);

  useEffect(() => {
    if (!token) {
      setAPIKeys([]);
      setAPIKeysLoading(false);
      setAPIKeysError(null);
      setAPIKeysSuccess(null);
      setAPIKeyCreating(false);
      setAPIKeyRevoking(null);
      setRevealedAPIKey(null);
      return;
    }

    let cancelled = false;
    setAPIKeysLoading(true);
    setAPIKeysError(null);

    fetchMyAPIKeys(token)
      .then((data) => {
        if (cancelled) {
          return;
        }

        setAPIKeys(data);
        setAPIKeysLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setAPIKeysError(error.message);
        setAPIKeysLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [apiKeysNonce, token]);

  useEffect(() => {
    if (route.name === 'skill-settings') {
      const nextKey = `${route.namespace}/${route.skillName}`;
      setManagedSkillKey((current) => (current === nextKey ? current : nextKey));
      return;
    }

    if (!mySkills.length) {
      setManagedSkillKey(null);
      setManagedSkill(null);
      return;
    }

    const exists = mySkills.some((skill) => skillKey(skill) === managedSkillKey);
    if (!exists) {
      setManagedSkillKey(skillKey(mySkills[0]));
    }
  }, [managedSkillKey, mySkills, route]);

  useEffect(() => {
    if (!token || !managedSkillKey) {
      return;
    }

    const [namespace, name] = managedSkillKey.split('/');
    let cancelled = false;
    setManageLoading(true);
    setManageError(null);

    fetchSkillDetail(namespace, name, token)
      .then((data) => {
        if (cancelled) {
          return;
        }

        setManagedSkill(data);
        setManageForm({
          description: data.description || '',
          descriptionZh: data.description_zh || '',
          category: data.category || '',
          license: data.license || 'MIT',
          tags: data.tags || [],
          isPublic: data.is_public ?? true,
          isDeprecated: data.is_deprecated ?? false,
          isRecommended: data.is_recommended ?? false,
        });
        setManageLoading(false);
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }

        setManageError(error.message);
        setManageLoading(false);
        setManagedSkill(null);
      });

    return () => {
      cancelled = true;
    };
  }, [manageNonce, managedSkillKey, token]);

  function commitCatalogFilters(nextFilters: CatalogFilters) {
    navigate(`/skills${toCatalogSearch(nextFilters)}`, { replace: true });
  }

  function setCatalogQuery(nextValue: string) {
    commitCatalogFilters({
      ...catalogFilters,
      query: nextValue,
    });
  }

  function updateCatalogFilter(
    key: 'namespace' | 'tag' | 'license' | 'sort' | 'view',
    value: string,
  ) {
    commitCatalogFilters({
      ...catalogFilters,
      [key]: value,
    });
  }

  function resetCatalogFilters() {
    navigate('/skills', { replace: true });
  }

  function openSkill(namespace: string, name: string) {
    const search = normalizedCatalogSearch || locationSearch || '';
    navigate(`${buildSkillPath(namespace, name)}${search}`);
  }

  function returnToCatalog() {
    const search = normalizedCatalogSearch || locationSearch || '';
    navigate(`/skills${search}`);
  }

  function getCurrentPath() {
    if (typeof window === 'undefined') {
      return '/';
    }

    return `${stripBasePath(window.location.pathname || '/', APP_BASE_PATH)}${window.location.search || ''}`;
  }

  async function submitAuth(mode: 'login' | 'register') {
    setAuthLoading(true);
    setAuthError(null);
    setAuthSuccess(null);

    try {
      const response: AuthResponse =
        mode === 'register'
          ? await registerUser({
              username: authForm.username.trim(),
              displayNameZh: authForm.displayNameZh.trim(),
              email: authForm.email.trim(),
              password: authForm.password,
            })
          : await loginUser({
              email: authForm.email.trim(),
              password: authForm.password,
            });

      setToken(response.token);
      setCurrentUser({
        id: response.user.id,
        username: response.user.username,
        display_name_zh: response.user.display_name_zh,
        email: response.user.email,
        is_admin: response.user.is_admin,
        is_super_admin: response.user.is_super_admin,
      });
      setProfileForm({
        displayNameZh: response.user.display_name_zh || '',
        avatarUrl: '',
      });
      setAuthForm((current) => ({
        ...current,
        password: '',
      }));
      setAuthSuccess(mode === 'register' ? '注册成功，已自动登录。' : '登录成功。');
      setAccountNonce((value) => value + 1);
      navigate(parseAuthRedirect(locationSearch) || '/settings/profile');
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : '请求失败');
    } finally {
      setAuthLoading(false);
    }
  }

  function handleLogout() {
    setToken('');
    setCurrentUser(null);
    setMySkills([]);
    setAPIKeys([]);
    setManagedSkillKey(null);
    setManagedSkill(null);
    setManageError(null);
    setManageSuccess(null);
    setManageRecommendationSaving(false);
    setAPIKeysError(null);
    setAPIKeysSuccess(null);
    setRevealedAPIKey(null);
    setProfileForm({ displayNameZh: '', avatarUrl: '' });
    setProfileError(null);
    setProfileSuccess(null);
    setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
    setPasswordError(null);
    setPasswordSuccess(null);
    setAdminUsers([]);
    setAdminUsersTotal(0);
    setAdminUsersError(null);
    setAdminUsersSuccess(null);
    setAdminUsersSaving(null);
    setAdminUserPasswordDrafts({});
    setAdminAuditLogs([]);
    setAdminAuditLogsError(null);
    setAuthSuccess('已退出登录。');
    setPublishSuccess(null);
    setPublishError(null);
    navigate('/');
  }

  async function submitProfileUpdate() {
    if (!token) {
      setProfileError('请先登录，再更新个人资料。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    const displayNameZh = profileForm.displayNameZh.trim();
    const avatarUrl = profileForm.avatarUrl.trim();
    if (!displayNameZh) {
      setProfileError('中文名不能为空。');
      return;
    }

    setProfileSaving(true);
    setProfileError(null);
    setProfileSuccess(null);

    try {
      const user = await updateCurrentUserProfile(token, {
        displayNameZh,
        avatarUrl: avatarUrl || undefined,
      });
      setCurrentUser(user);
      setProfileForm({
        displayNameZh: user.display_name_zh || '',
        avatarUrl: user.avatar_url || '',
      });
      setProfileSuccess('个人资料已更新。');
    } catch (error) {
      setProfileError(error instanceof Error ? error.message : '更新个人资料失败');
    } finally {
      setProfileSaving(false);
    }
  }

  async function submitPasswordUpdate() {
    if (!token) {
      setPasswordError('请先登录，再修改密码。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    if (!passwordForm.currentPassword) {
      setPasswordError('请填写当前密码。');
      return;
    }
    if (passwordForm.newPassword.length < 6) {
      setPasswordError('新密码至少 6 个字符。');
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordError('两次输入的新密码不一致。');
      return;
    }

    setPasswordSaving(true);
    setPasswordError(null);
    setPasswordSuccess(null);

    try {
      await updateCurrentUserPassword(token, {
        currentPassword: passwordForm.currentPassword,
        newPassword: passwordForm.newPassword,
      });
      setPasswordForm({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      });
      setPasswordSuccess('密码已更新。');
    } catch (error) {
      setPasswordError(error instanceof Error ? error.message : '修改密码失败');
    } finally {
      setPasswordSaving(false);
    }
  }

  async function updateAdminUserAccess(userID: string, payload: AdminUpdateUserPayload) {
    if (!token || !currentUser?.is_super_admin) {
      setAdminUsersError('只有超级管理员可以管理用户。');
      return false;
    }

    setAdminUsersSaving(userID);
    setAdminUsersError(null);
    setAdminUsersSuccess(null);

    try {
      const updated = await updateAdminUser(token, userID, payload);
      setAdminUsers((current) => current.map((item) => (item.id === updated.id ? updated : item)));
      setAdminUsersSuccess('用户权限已更新。');
      if (updated.id === currentUser.id) {
        setCurrentUser(updated);
      }
      return true;
    } catch (error) {
      setAdminUsersError(error instanceof Error ? error.message : '更新用户权限失败');
      return false;
    } finally {
      setAdminUsersSaving(null);
    }
  }

  async function resetAdminUserPassword(userID: string) {
    const password = (adminUserPasswordDrafts[userID] || '').trim();
    if (password.length < 6) {
      setAdminUsersError('新密码至少 6 个字符。');
      return;
    }

    const updated = await updateAdminUserAccess(userID, { password });
    if (updated) {
      setAdminUserPasswordDrafts((current) => ({
        ...current,
        [userID]: '',
      }));
    }
  }

  async function submitAPIKeyCreate() {
    if (!token) {
      setAPIKeysError('请先登录，再创建 API Key。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    const name = apiKeyForm.name.trim();
    if (!name) {
      setAPIKeysError('请先填写 API Key 名称。');
      return;
    }

    const expiresAt =
      apiKeyForm.expiryPreset === 'custom'
        ? toISOStringFromLocalDateTime(apiKeyForm.customExpiresAt)
        : getExpiresAtFromPreset(apiKeyForm.expiryPreset);
    if (apiKeyForm.expiryPreset === 'custom' && !expiresAt) {
      setAPIKeysError('请填写有效的自定义到期时间。');
      return;
    }
    if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
      setAPIKeysError('到期时间必须晚于当前时间。');
      return;
    }

    setAPIKeyCreating(true);
    setAPIKeysError(null);
    setAPIKeysSuccess(null);

    try {
      const response = await createAPIKey(token, {
        name,
        expiresAt,
      });

      setRevealedAPIKey(response);
      setAPIKeysSuccess('API Key 已生成，请立即复制并妥善保存。');
      setAPIKeyForm({
        name: '',
        expiryPreset: 'never',
        customExpiresAt: '',
      });
      setAPIKeysNonce((value) => value + 1);
    } catch (error) {
      setAPIKeysError(error instanceof Error ? error.message : '创建 API Key 失败');
    } finally {
      setAPIKeyCreating(false);
    }
  }

  async function submitPublish() {
    if (!token) {
      setPublishError('请先登录，再发布 skill。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    if (!publishFile) {
      setPublishError('请先选择一个 .zip 技能包。');
      return;
    }
    const metadataError = validateOfficialMetadataInput(publishForm.category, publishForm.tags);
    if (metadataError) {
      setPublishError(metadataError);
      return;
    }

    setPublishLoading(true);
    setPublishError(null);
    setPublishSuccess(null);

    try {
      const response = await publishSkill(token, {
        namespace: publishForm.namespace.trim(),
        name: publishForm.name.trim(),
        description: publishForm.description.trim(),
        descriptionZh: publishForm.descriptionZh.trim(),
        category: publishForm.category.trim(),
        version: publishForm.version.trim(),
        license: publishForm.license.trim(),
        tags: normalizeOfficialTags(publishForm.tags),
        isPublic: publishForm.isPublic,
        archive: publishFile,
      });

      setPublishSuccess(response);
      setPublishForm((current) => ({
        ...current,
        name: '',
        description: '',
        descriptionZh: '',
        category: '',
        version: '0.1.0',
        tags: [],
      }));
      setPublishFile(null);
      setAccountNonce((value) => value + 1);
      setCatalogNonce((value) => value + 1);
      setDetailNonce((value) => value + 1);
      navigate(buildSkillPath(response.namespace, response.name));
    } catch (error) {
      setPublishError(error instanceof Error ? error.message : '发布失败');
    } finally {
      setPublishLoading(false);
    }
  }

  async function submitManage() {
    if (!token || !managedSkill) {
      setManageError('请先登录并选择一个 skill。');
      return;
    }
    const metadataError = validateOfficialMetadataInput(manageForm.category, manageForm.tags);
    if (metadataError) {
      setManageError(metadataError);
      return;
    }

    setManageSaving(true);
    setManageError(null);
    setManageSuccess(null);

    try {
      await updateSkill(token, managedSkill.namespace, managedSkill.name, {
        description: manageForm.description.trim(),
        descriptionZh: manageForm.descriptionZh.trim(),
        category: manageForm.category.trim(),
        tags: normalizeOfficialTags(manageForm.tags),
        license: manageForm.license.trim(),
        isPublic: manageForm.isPublic,
        isDeprecated: manageForm.isDeprecated,
      });

      setManageSuccess('技能信息已更新。');
      setAccountNonce((value) => value + 1);
      setCatalogNonce((value) => value + 1);
      setDetailNonce((value) => value + 1);
      setManageNonce((value) => value + 1);
    } catch (error) {
      setManageError(error instanceof Error ? error.message : '更新失败');
    } finally {
      setManageSaving(false);
    }
  }

  async function submitManageRecommendation() {
    if (!token || !managedSkill) {
      setManageError('请先登录并选择一个 skill。');
      return;
    }
    if (!(currentUser?.is_admin || currentUser?.is_super_admin)) {
      setManageError('只有管理员和超级管理员可以调整推荐状态。');
      return;
    }

    setManageRecommendationSaving(true);
    setManageError(null);
    setManageSuccess(null);

    try {
      await updateSkillRecommendation(token, managedSkill.namespace, managedSkill.name, {
        isRecommended: manageForm.isRecommended,
      });

      setManageSuccess(manageForm.isRecommended ? '技能已加入推荐列表。' : '技能已从推荐列表移除。');
      setAccountNonce((value) => value + 1);
      setCatalogNonce((value) => value + 1);
      setDetailNonce((value) => value + 1);
      setManageNonce((value) => value + 1);
    } catch (error) {
      setManageError(error instanceof Error ? error.message : '更新推荐状态失败');
    } finally {
      setManageRecommendationSaving(false);
    }
  }

  async function removeManagedSkill() {
    if (!token || !managedSkill) {
      return;
    }

    const targetRef = `@${managedSkill.namespace}/${managedSkill.name}`;
    if (!window.confirm(`确认删除 ${targetRef} 吗？这个动作不可撤销。`)) {
      return;
    }

    setManageDeletingSkill(true);
    setManageError(null);
    setManageSuccess(null);

    try {
      await deleteSkill(token, managedSkill.namespace, managedSkill.name);
      setManageSuccess(`${targetRef} 已删除。`);
      setMySkills((current) =>
        current.filter((skill) => skillKey(skill) !== skillKey(managedSkill)),
      );
      setManagedSkill(null);
      setManagedSkillKey(null);
      setAccountNonce((value) => value + 1);
      setCatalogNonce((value) => value + 1);
      setDetailNonce((value) => value + 1);
      if (route.name === 'skill-settings' && route.namespace === managedSkill.namespace && route.skillName === managedSkill.name) {
        navigate('/settings/profile');
      } else if (route.name === 'skill-tab' && route.namespace === managedSkill.namespace && route.skillName === managedSkill.name) {
        navigate('/skills');
      }
    } catch (error) {
      setManageError(error instanceof Error ? error.message : '删除 skill 失败');
    } finally {
      setManageDeletingSkill(false);
    }
  }

  async function removeManagedVersion(version: string) {
    if (!token || !managedSkill) {
      return;
    }

    const targetRef = `@${managedSkill.namespace}/${managedSkill.name}`;
    if (!window.confirm(`确认删除 ${targetRef} 的版本 ${version} 吗？`)) {
      return;
    }

    setManageDeletingVersion(version);
    setManageError(null);
    setManageSuccess(null);

    try {
      await deleteSkillVersion(token, managedSkill.namespace, managedSkill.name, version);
      setManageSuccess(`版本 ${version} 已删除。`);
      setAccountNonce((value) => value + 1);
      setCatalogNonce((value) => value + 1);
      setDetailNonce((value) => value + 1);
      setManageNonce((value) => value + 1);
    } catch (error) {
      setManageError(error instanceof Error ? error.message : '删除版本失败');
    } finally {
      setManageDeletingVersion(null);
    }
  }

  async function removeAPIKey(id: string) {
    if (!token) {
      return;
    }

    const target = apiKeys.find((item) => item.id === id);
    const label = target?.name || target?.prefix || '当前 API Key';
    if (!window.confirm(`确认撤销 ${label} 吗？撤销后依赖它的脚本和集成会立即失效。`)) {
      return;
    }

    setAPIKeyRevoking(id);
    setAPIKeysError(null);
    setAPIKeysSuccess(null);

    try {
      await revokeAPIKey(token, id);
      setAPIKeys((current) => current.filter((item) => item.id !== id));
      setAPIKeysSuccess('API Key 已撤销。');
      if (revealedAPIKey?.id === id) {
        setRevealedAPIKey(null);
      }
    } catch (error) {
      setAPIKeysError(error instanceof Error ? error.message : '撤销 API Key 失败');
    } finally {
      setAPIKeyRevoking(null);
    }
  }

  async function submitCommunityTag(rawTag: string) {
    if (!routeSkillKey || !detailSkill) {
      return;
    }

    if (!token) {
      setCommunityTagError('请先登录，再添加社区标签。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    const tag = parseTags(rawTag)[0]?.trim();
    if (!tag) {
      setCommunityTagError('请先输入一个社区标签。');
      return;
    }

    setCommunityTagSaving(true);
    setCommunityTagError(null);
    setCommunityTagSuccess(null);

    try {
      const [namespace, name] = routeSkillKey.split('/');
      const response = await addCommunityTag(token, namespace, name, { tag });
      setDetailSkill(response);
      setCommunityTagSuccess('社区标签已更新。');
    } catch (error) {
      setCommunityTagError(error instanceof Error ? error.message : '添加社区标签失败');
    } finally {
      setCommunityTagSaving(false);
    }
  }

  async function submitSkillRating(rating: number, comment = '') {
    if (!routeSkillKey || !detailSkill) {
      return;
    }

    if (!token) {
      setSkillRatingError('请先登录，再为 skill 评分。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    if (rating < 1 || rating > 5) {
      setSkillRatingError('请选择 1 到 5 分。');
      return;
    }

    setSkillRatingSaving(true);
    setSkillRatingError(null);
    setSkillRatingSuccess(null);

    try {
      const [namespace, name] = routeSkillKey.split('/');
      const response = await rateSkill(token, namespace, name, {
        rating,
        comment: comment.trim(),
      });

      setDetailSkill((current) => {
        if (!current) {
          return {
            ...response.skill,
            user_rating: response.user_rating,
          };
        }

        return {
          ...current,
          ...response.skill,
          tags: response.skill.tags ?? current.tags,
          owner: response.skill.owner ?? current.owner,
          versions: response.skill.versions?.length ? response.skill.versions : current.versions,
          community_tags: current.community_tags,
          viewer_tags: current.viewer_tags,
          download_url: response.skill.download_url ?? current.download_url,
          user_rating: response.user_rating,
        };
      });
      setSkillRatingSuccess('评分已保存。');
    } catch (error) {
      setSkillRatingError(error instanceof Error ? error.message : '评分失败');
    } finally {
      setSkillRatingSaving(false);
    }
  }

  async function toggleSkillLike() {
    if (!routeSkillKey || !detailSkill) {
      return;
    }
    if (!token) {
      setSkillLikeError('请先登录，再点赞 skill。');
      navigate(buildAuthPath('login', getCurrentPath()));
      return;
    }

    setSkillLikeSaving(true);
    setSkillLikeError(null);

    try {
      const [namespace, name] = routeSkillKey.split('/');
      const response = detailSkill.viewer_liked
        ? await unlikeSkill(token, namespace, name)
        : await likeSkill(token, namespace, name);
      setDetailSkill(response);
      setSkills((current) =>
        current.map((skill) => (skillKey(skill) === skillKey(response) ? { ...skill, ...response } : skill)),
      );
      setCatalogSkills((current) =>
        current.map((skill) => (skillKey(skill) === skillKey(response) ? { ...skill, ...response } : skill)),
      );
      setAccountNonce((value) => value + 1);
    } catch (error) {
      setSkillLikeError(error instanceof Error ? error.message : '点赞操作失败');
    } finally {
      setSkillLikeSaving(false);
    }
  }

  async function shareDetailSkill() {
    if (!detailSkill) {
      return;
    }

    const title = `${detailSkill.namespace}/${detailSkill.name}`;
    const path = buildSkillPath(detailSkill.namespace, detailSkill.name);
    const url =
      typeof window === 'undefined'
        ? path
        : `${window.location.origin}${APP_BASE_PATH === '/' ? '' : APP_BASE_PATH}${path}`;

    try {
      if (typeof navigator !== 'undefined' && navigator.share) {
        await navigator.share({
          title,
          text: getSkillDescription(detailSkill),
          url,
        });
        setSkillShareStatus('分享面板已打开。');
        void recordShareEvent(token || undefined, detailSkill.namespace, detailSkill.name, 'web-share');
        return;
      }

      await copyText(url);
      setSkillShareStatus('详情链接已复制。');
      void recordShareEvent(token || undefined, detailSkill.namespace, detailSkill.name, 'copy-link');
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        return;
      }
      setSkillShareStatus(error instanceof Error ? error.message : '分享失败');
    }
  }

  async function removeCommunitySkillTag(tag: string) {
    if (!routeSkillKey || !detailSkill || !token) {
      return;
    }

    setCommunityTagRemoving(tag);
    setCommunityTagError(null);
    setCommunityTagSuccess(null);

    try {
      const [namespace, name] = routeSkillKey.split('/');
      const response = await removeCommunityTag(token, namespace, name, tag);
      setDetailSkill(response);
      setCommunityTagSuccess('社区标签已移除。');
    } catch (error) {
      setCommunityTagError(error instanceof Error ? error.message : '移除社区标签失败');
    } finally {
      setCommunityTagRemoving(null);
    }
  }

  const relatedSkills =
    detailSkill == null
      ? []
      : skills
          .filter((skill) => skillKey(skill) !== skillKey(detailSkill))
          .filter(
            (skill) =>
              skill.namespace === detailSkill.namespace ||
              (skill.tags || []).some((tag) => (detailSkill.tags || []).includes(tag)),
          )
          .slice(0, 4);

  const accountStats = {
    total: userStats?.skill_count ?? mySkills.length,
    publicCount: userStats?.public_skill_count ?? mySkills.filter((skill) => skill.is_public !== false).length,
    privateCount:
      userStats != null
        ? Math.max(userStats.skill_count - userStats.public_skill_count, 0)
        : mySkills.filter((skill) => skill.is_public === false).length,
    totalLikes: userStats?.total_like_count ?? mySkills.reduce((sum, skill) => sum + (skill.like_count || 0), 0),
    totalDownloads:
      userStats?.total_download_count ?? mySkills.reduce((sum, skill) => sum + (skill.download_count || 0), 0),
    totalInstalls:
      userStats?.total_install_count ?? mySkills.reduce((sum, skill) => sum + (skill.install_count || 0), 0),
    averageRating: userStats?.average_rating ?? 0,
    totalRatings: userStats?.total_rating_count ?? mySkills.reduce((sum, skill) => sum + (skill.rating_count || 0), 0),
  };

  const apiKeyStats = {
    total: apiKeys.length,
    active: apiKeys.filter((item) => !item.expires_at || new Date(item.expires_at).getTime() > Date.now()).length,
    expiringSoon: apiKeys.filter((item) => {
      if (!item.expires_at) {
        return false;
      }
      const delta = new Date(item.expires_at).getTime() - Date.now();
      return delta > 0 && delta <= 7 * 24 * 60 * 60 * 1000;
    }).length,
  };

  const catalogPreviewSkills =
    catalogSearchKey === normalizedCatalogSearch
      ? catalogSkills
      : filterCatalogSkills(skills, catalogFilters);
  const catalogDisplaySkills = route.name === 'skills' ? catalogPreviewSkills : [];
  const catalogDisplayTotal =
    catalogSearchKey === normalizedCatalogSearch ? catalogTotal : catalogPreviewSkills.length;
  const catalogDisplayLoading =
    route.name === 'skills' && (catalogLoading || catalogSearchKey !== normalizedCatalogSearch);
  const catalogDisplayError =
    route.name === 'skills' && catalogSearchKey === normalizedCatalogSearch ? catalogError : null;

  return {
    health,
    healthError,
    token,
    currentUser,
    authLoading,
    authError,
    authSuccess,
    authForm,
    setAuthForm,
    submitAuth,
    handleLogout,
    skills,
    skillsTotal,
    skillsLoading,
    skillsError,
    catalogSkills,
    catalogTotal,
    catalogLoading,
    catalogError,
    catalogDisplaySkills,
    catalogDisplayTotal,
    catalogDisplayLoading,
    catalogDisplayError,
    catalogFilters,
    namespaceOptions,
    tagOptions,
    licenseOptions,
    quickStats,
    featuredSkills,
    latestSkills,
    setCatalogQuery,
    updateCatalogFilter,
    resetCatalogFilters,
    refreshCatalog: () => setCatalogSearchNonce((value) => value + 1),
    openSkill,
    returnToCatalog,
    detailSkill,
    detailLoading,
    detailError,
    communityTagSaving,
    communityTagRemoving,
    communityTagError,
    communityTagSuccess,
    skillRatingSaving,
    skillRatingError,
    skillRatingSuccess,
    skillLikeSaving,
    skillLikeError,
    skillShareStatus,
    submitCommunityTag,
    submitSkillRating,
    toggleSkillLike,
    shareDetailSkill,
    removeCommunityTag: removeCommunitySkillTag,
    refreshDetail: () => setDetailNonce((value) => value + 1),
    accountLoading,
    accountError,
    profileForm,
    setProfileForm,
    profileSaving,
    profileError,
    profileSuccess,
    submitProfileUpdate,
    passwordForm,
    setPasswordForm,
    passwordSaving,
    passwordError,
    passwordSuccess,
    submitPasswordUpdate,
    adminUsers,
    adminUsersTotal,
    adminUsersLoading,
    adminUsersError,
    adminUsersSuccess,
    adminUsersSaving,
    adminUserFilters,
    setAdminUserFilters,
    adminUserPasswordDrafts,
    setAdminUserPasswordDrafts,
    adminAuditLogs,
    adminAuditLogsLoading,
    adminAuditLogsError,
    updateAdminUserAccess,
    resetAdminUserPassword,
    refreshAdminUsers: () => setAdminUsersNonce((value) => value + 1),
    mySkills,
    apiKeys,
    apiKeysLoading,
    apiKeysError,
    apiKeysSuccess,
    apiKeyCreating,
    apiKeyRevoking,
    revealedAPIKey,
    setRevealedAPIKey,
    apiKeyForm,
    setAPIKeyForm,
    submitAPIKeyCreate,
    removeAPIKey,
    refreshAPIKeys: () => setAPIKeysNonce((value) => value + 1),
    accountStats,
    apiKeyStats,
    managedSkillKey,
    setManagedSkillKey,
    managedSkill,
    manageLoading,
    manageSaving,
    manageRecommendationSaving,
    manageDeletingSkill,
    manageDeletingVersion,
    manageError,
    manageSuccess,
    manageForm,
    setManageForm,
    submitManage,
    submitManageRecommendation,
    removeManagedSkill,
    removeManagedVersion,
    publishLoading,
    publishError,
    publishSuccess,
    publishForm,
    setPublishForm,
    publishFile,
    setPublishFile,
    submitPublish,
    relatedSkills,
  };
}
