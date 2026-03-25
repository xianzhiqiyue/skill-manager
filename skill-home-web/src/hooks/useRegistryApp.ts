import { startTransition, useDeferredValue, useEffect, useState } from 'react';

import {
  deleteSkill,
  deleteSkillVersion,
  fetchCurrentUser,
  fetchHealth,
  fetchMySkills,
  fetchSkillDetail,
  fetchSkills,
  loginUser,
  publishSkill,
  registerUser,
  updateSkill,
  type AuthResponse,
  type AuthUser,
  type FetchSkillsParams,
  type HealthResponse,
  type PublishResponse,
  type SkillDetail,
  type SkillSummary,
} from '../api';
import { parseTags, skillKey } from '../lib/format';
import { buildSkillPath, type AppRoute } from '../lib/routes';

const TOKEN_STORAGE_KEY = 'skill-home-web-token';

export type CatalogSort = NonNullable<FetchSkillsParams['sort']>;
export type CatalogView = 'cards' | 'list';

function loadStoredToken() {
  if (typeof window === 'undefined') {
    return '';
  }
  return window.localStorage.getItem(TOKEN_STORAGE_KEY) || '';
}

function uniq(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

export function useRegistryApp(
  route: AppRoute,
  navigate: (path: string, options?: { replace?: boolean }) => void,
) {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);

  const [catalogFilters, setCatalogFilters] = useState({
    query: '',
    namespace: 'all',
    tag: 'all',
    license: 'all',
    sort: 'downloads' as CatalogSort,
    view: 'list' as CatalogView,
  });
  const deferredQuery = useDeferredValue(catalogFilters.query);

  const [skills, setSkills] = useState<SkillSummary[]>([]);
  const [skillsTotal, setSkillsTotal] = useState(0);
  const [skillsLoading, setSkillsLoading] = useState(true);
  const [skillsError, setSkillsError] = useState<string | null>(null);
  const [catalogNonce, setCatalogNonce] = useState(0);

  const routeSkillKey =
    route.name === 'skill' ? `${route.namespace}/${route.skillName}` : null;

  const [detailSkill, setDetailSkill] = useState<SkillDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [detailNonce, setDetailNonce] = useState(0);

  const [token, setToken] = useState(loadStoredToken);
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [authSuccess, setAuthSuccess] = useState<string | null>(null);
  const [authForm, setAuthForm] = useState({
    username: '',
    email: '',
    password: '',
  });

  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null);
  const [accountLoading, setAccountLoading] = useState(false);
  const [accountError, setAccountError] = useState<string | null>(null);
  const [accountNonce, setAccountNonce] = useState(0);

  const [mySkills, setMySkills] = useState<SkillSummary[]>([]);
  const [managedSkillKey, setManagedSkillKey] = useState<string | null>(null);
  const [managedSkill, setManagedSkill] = useState<SkillDetail | null>(null);
  const [manageLoading, setManageLoading] = useState(false);
  const [manageSaving, setManageSaving] = useState(false);
  const [manageDeletingSkill, setManageDeletingSkill] = useState(false);
  const [manageDeletingVersion, setManageDeletingVersion] = useState<string | null>(null);
  const [manageError, setManageError] = useState<string | null>(null);
  const [manageSuccess, setManageSuccess] = useState<string | null>(null);
  const [manageNonce, setManageNonce] = useState(0);
  const [manageForm, setManageForm] = useState({
    description: '',
    license: 'MIT',
    tags: '',
    isPublic: true,
    isDeprecated: false,
  });

  const [publishLoading, setPublishLoading] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);
  const [publishSuccess, setPublishSuccess] = useState<PublishResponse | null>(null);
  const [publishForm, setPublishForm] = useState({
    namespace: '',
    name: '',
    description: '',
    version: '0.1.0',
    license: 'MIT',
    isPublic: true,
  });
  const [publishFile, setPublishFile] = useState<File | null>(null);

  const namespaceOptions = uniq(skills.map((skill) => skill.namespace)).sort();
  const tagOptions = uniq(skills.flatMap((skill) => skill.tags || [])).sort();
  const licenseOptions = uniq(skills.map((skill) => skill.license || '')).sort();

  const featuredSkills = [...skills]
    .sort((left, right) => right.download_count - left.download_count)
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
      query: deferredQuery,
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
  }, [
    catalogFilters.license,
    catalogFilters.namespace,
    catalogFilters.sort,
    catalogFilters.tag,
    catalogNonce,
    deferredQuery,
  ]);

  useEffect(() => {
    if (!routeSkillKey) {
      setDetailSkill(null);
      setDetailError(null);
      setDetailLoading(false);
      return;
    }

    const [namespace, name] = routeSkillKey.split('/');
    let cancelled = false;
    setDetailLoading(true);
    setDetailError(null);

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
      setAccountError(null);
      setManagedSkillKey(null);
      setManagedSkill(null);
      setManageError(null);
      setManageSuccess(null);
      return;
    }

    let cancelled = false;
    setAccountLoading(true);
    setAccountError(null);

    Promise.all([fetchCurrentUser(token), fetchMySkills(token)])
      .then(([user, ownedSkills]) => {
        if (cancelled) {
          return;
        }

        setCurrentUser(user);
        setMySkills(ownedSkills);
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
    if (!mySkills.length) {
      setManagedSkillKey(null);
      setManagedSkill(null);
      return;
    }

    const exists = mySkills.some((skill) => skillKey(skill) === managedSkillKey);
    if (!exists) {
      setManagedSkillKey(skillKey(mySkills[0]));
    }
  }, [managedSkillKey, mySkills]);

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
          license: data.license || 'MIT',
          tags: (data.tags || []).join(', '),
          isPublic: data.is_public ?? true,
          isDeprecated: data.is_deprecated ?? false,
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

  function setCatalogQuery(nextValue: string) {
    startTransition(() => {
      setCatalogFilters((current) => ({
        ...current,
        query: nextValue,
      }));
    });
  }

  function updateCatalogFilter(
    key: 'namespace' | 'tag' | 'license' | 'sort' | 'view',
    value: string,
  ) {
    setCatalogFilters((current) => ({
      ...current,
      [key]: value,
    }));
  }

  function resetCatalogFilters() {
    setCatalogFilters({
      query: '',
      namespace: 'all',
      tag: 'all',
      license: 'all',
      sort: 'downloads',
      view: 'list',
    });
  }

  function openSkill(namespace: string, name: string) {
    navigate(buildSkillPath(namespace, name));
  }

  function returnToCatalog() {
    navigate('/skills');
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
        email: response.user.email,
      });
      setAuthForm((current) => ({
        ...current,
        password: '',
      }));
      setAuthSuccess(mode === 'register' ? '注册成功，已自动登录。' : '登录成功。');
      setAccountNonce((value) => value + 1);
      navigate('/console');
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
    setManagedSkillKey(null);
    setManagedSkill(null);
    setManageError(null);
    setManageSuccess(null);
    setAuthSuccess('已退出登录。');
    setPublishSuccess(null);
    setPublishError(null);
    navigate('/');
  }

  async function submitPublish() {
    if (!token) {
      setPublishError('请先登录，再发布 skill。');
      navigate('/login');
      return;
    }

    if (!publishFile) {
      setPublishError('请先选择一个 .zip 技能包。');
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
        version: publishForm.version.trim(),
        license: publishForm.license.trim(),
        isPublic: publishForm.isPublic,
        archive: publishFile,
      });

      setPublishSuccess(response);
      setPublishForm((current) => ({
        ...current,
        name: '',
        description: '',
        version: '0.1.0',
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

    setManageSaving(true);
    setManageError(null);
    setManageSuccess(null);

    try {
      await updateSkill(token, managedSkill.namespace, managedSkill.name, {
        description: manageForm.description.trim(),
        tags: parseTags(manageForm.tags),
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
      if (route.name === 'skill' && route.namespace === managedSkill.namespace && route.skillName === managedSkill.name) {
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
    total: mySkills.length,
    publicCount: mySkills.filter((skill) => skill.is_public !== false).length,
    privateCount: mySkills.filter((skill) => skill.is_public === false).length,
  };

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
    refreshCatalog: () => setCatalogNonce((value) => value + 1),
    openSkill,
    returnToCatalog,
    detailSkill,
    detailLoading,
    detailError,
    refreshDetail: () => setDetailNonce((value) => value + 1),
    accountLoading,
    accountError,
    mySkills,
    accountStats,
    managedSkillKey,
    setManagedSkillKey,
    managedSkill,
    manageLoading,
    manageSaving,
    manageDeletingSkill,
    manageDeletingVersion,
    manageError,
    manageSuccess,
    manageForm,
    setManageForm,
    submitManage,
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
