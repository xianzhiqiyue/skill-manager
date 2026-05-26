import { act, cleanup, render, renderHook, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  addCommunityTag,
  deleteSkill,
  fetchAdminAuditLogs,
  fetchAdminUsers,
  fetchCurrentUser,
  fetchCurrentUserStats,
  fetchSkills,
  fetchSkillDetail,
  likeSkill,
  loginUser,
  publishSkill,
  recordShareEvent,
  removeCommunityTag,
  registerUser,
  unlikeSkill,
  updateAdminUser,
  updateCurrentUserPassword,
  updateCurrentUserProfile,
  updateSkill,
  updateSkillRecommendation,
} from '../api';
import { PublishNewPage } from '../pages/PublishNewPage';
import { SkillOverviewPage } from '../pages/skill/SkillOverviewPage';
import { useRegistryApp } from './useRegistryApp';

const mockRegistryBase = 'http://127.0.0.1:8080';
const { mockedRateSkill } = vi.hoisted(() => ({
  mockedRateSkill: vi.fn(),
}));

vi.mock('../components/object/CopyActionButton', () => ({
  CopyActionButton: ({
    className,
    label = '复制',
    value,
  }: {
    className?: string;
    label?: string;
    value: string;
  }) => (
    <button className={className} data-value={value} type="button">
      {label}
    </button>
  ),
}));

vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api')>();

  return {
    ...actual,
    createAPIKey: vi.fn(),
    deleteSkill: vi.fn(),
    deleteSkillVersion: vi.fn(),
    fetchAdminAuditLogs: vi.fn().mockResolvedValue({
      total: 0,
      page: 1,
      per_page: 8,
      results: [],
    }),
    fetchAdminUsers: vi.fn().mockResolvedValue({
      total: 0,
      page: 1,
      per_page: 20,
      results: [],
    }),
    fetchCurrentUser: vi.fn().mockResolvedValue({
      id: 'user-1',
      username: 'testuser',
      display_name_zh: '测试用户',
      email: 'test@example.com',
      is_admin: false,
      is_super_admin: false,
      created_at: '2026-03-20T10:00:00Z',
    }),
    fetchCurrentUserStats: vi.fn().mockResolvedValue({
      user_id: 'user-1',
      username: 'testuser',
      display_name_zh: '测试用户',
      skill_count: 0,
      public_skill_count: 0,
      total_like_count: 0,
      total_install_count: 0,
      total_download_count: 0,
      average_rating: 0,
      total_rating_count: 0,
    }),
    fetchHealth: vi.fn().mockResolvedValue({
      service: 'skill-home',
      status: 'ok',
      version: '1.0.0',
    }),
    fetchMyAPIKeys: vi.fn().mockResolvedValue([]),
    fetchMySkills: vi.fn().mockResolvedValue([]),
    fetchSkillDetail: vi.fn().mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      owner_id: 'user-1',
      description: 'Interact with GitHub using gh.',
      category: 'integration',
      tags: ['api'],
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      is_public: true,
      is_recommended: false,
      versions: [],
    }),
    fetchSkills: vi.fn().mockResolvedValue({
      total: 0,
      results: [],
    }),
    loginUser: vi.fn(),
    likeSkill: vi.fn(),
    publishSkill: vi.fn(),
    rateSkill: mockedRateSkill,
    recordShareEvent: vi.fn().mockResolvedValue({ message: 'recorded' }),
    addCommunityTag: vi.fn(),
    removeCommunityTag: vi.fn(),
    registerUser: vi.fn(),
    revokeAPIKey: vi.fn(),
    unlikeSkill: vi.fn(),
    updateAdminUser: vi.fn(),
    updateCurrentUserPassword: vi.fn(),
    updateCurrentUserProfile: vi.fn(),
    updateSkill: vi.fn(),
    updateSkillRecommendation: vi.fn(),
  };
});

describe('useRegistryApp catalog URL sync', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
  });

  it('initializes catalog filters from the current location search', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=doc&namespace=testuser&sort=updated', navigate),
    );

    expect(result.current.catalogFilters.query).toBe('doc');
    expect(result.current.catalogFilters.namespace).toBe('testuser');
    expect(result.current.catalogFilters.sort).toBe('updated');
  });

  it('commits non-default catalog filters to the skills URL instead of mutating local state', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=doc', navigate),
    );

    act(() => {
      result.current.updateCatalogFilter('sort', 'updated');
    });

    expect(navigate).toHaveBeenCalledWith('/skills?q=doc&sort=updated', { replace: true });
    expect(result.current.catalogFilters.query).toBe('doc');
    expect(result.current.catalogFilters.sort).toBe('downloads');
  });

  it('commits submitted catalog queries to the skills URL instead of mutating local state', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=doc&sort=updated', navigate),
    );

    act(() => {
      result.current.setCatalogQuery('fmea');
    });

    expect(navigate).toHaveBeenCalledWith('/skills?q=fmea&sort=updated', { replace: true });
    expect(result.current.catalogFilters.query).toBe('doc');
    expect(result.current.catalogFilters.sort).toBe('updated');
  });

  it('commits filter resets to the bare skills URL', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=doc&namespace=testuser&sort=updated', navigate),
    );

    act(() => {
      result.current.resetCatalogFilters();
    });

    expect(navigate).toHaveBeenCalledWith('/skills', { replace: true });
  });

  it('preserves the current catalog search when opening a skill result', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=github&tag=automation', navigate),
    );

    act(() => {
      result.current.openSkill('testuser', 'github');
    });

    expect(navigate).toHaveBeenCalledWith('/skills/testuser/github?q=github&tag=automation');
  });

  it('returns to the filtered catalog URL from a skill object page', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '?q=github&tag=automation',
        navigate,
      ),
    );

    act(() => {
      result.current.returnToCatalog();
    });

    expect(navigate).toHaveBeenCalledWith('/skills?q=github&tag=automation');
  });

  it('hydrates the global search query from deep-linked skill object URLs', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '?q=github&tag=automation',
        navigate,
      ),
    );

    expect(result.current.catalogFilters.query).toBe('github');
    expect(result.current.catalogFilters.tag).toBe('automation');
  });

  it('hydrates catalog filters immediately when routing from home into a filtered skills URL', () => {
    const navigate = vi.fn();
    type CatalogRoute = { name: 'home' } | { name: 'skills' };
    const { result, rerender } = renderHook(
      ({ route, search }: { route: CatalogRoute; search: string }) =>
        useRegistryApp(route, search, navigate),
      {
        initialProps: {
          route: { name: 'home' } as CatalogRoute,
          search: '',
        },
      },
    );

    rerender({
      route: { name: 'skills' } as CatalogRoute,
      search: '?q=fmea',
    });

    expect(result.current.catalogFilters.query).toBe('fmea');
    expect(navigate).not.toHaveBeenCalledWith('/skills', { replace: true });
  });

  it('uses the directory snapshot as a preview while a new committed catalog search is loading', async () => {
    const navigate = vi.fn();
    type CatalogRoute = { name: 'home' } | { name: 'skills' };
    let resolveCatalogFetch:
      | ((value: { total: number; results: Array<Record<string, unknown>> }) => void)
      | null = null;
    const pendingCatalogFetch = new Promise<{ total: number; results: Array<Record<string, unknown>> }>(
      (resolve) => {
        resolveCatalogFetch = resolve;
      },
    );

    vi.mocked(fetchSkills)
      .mockResolvedValueOnce({
        total: 2,
        results: [
          {
            id: '1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            tags: ['automation', 'github'],
            license: 'MIT',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            updated_at: '2026-03-22T21:32:00Z',
          },
          {
            id: '2',
            namespace: 'zhuyuxiao314',
            name: 'openclaw-fmea-cocreator',
            description: 'Builds FMEA drafts.',
            tags: ['analysis', 'docs'],
            license: 'MIT',
            download_count: 3,
            rating_count: 0,
            latest_version: '0.2.0',
            updated_at: '2026-04-02T00:00:00Z',
          },
        ],
      })
      .mockImplementationOnce(() => pendingCatalogFetch as never);

    const { result, rerender } = renderHook(
      ({ route, search }: { route: CatalogRoute; search: string }) =>
        useRegistryApp(route, search, navigate),
      {
        initialProps: {
          route: { name: 'home' } as CatalogRoute,
          search: '',
        },
      },
    );

    await waitFor(() => {
      expect(result.current.skills).toHaveLength(2);
    });

    rerender({
      route: { name: 'skills' } as CatalogRoute,
      search: '?q=fmea',
    });

    expect(result.current.catalogDisplayLoading).toBe(true);
    expect(result.current.catalogDisplayTotal).toBe(1);
    expect(result.current.catalogDisplaySkills.map((skill) => skill.name)).toEqual([
      'openclaw-fmea-cocreator',
    ]);

    resolveCatalogFetch!({
      total: 1,
      results: [
        {
          id: '2',
          namespace: 'zhuyuxiao314',
          name: 'openclaw-fmea-cocreator',
          description: 'Builds FMEA drafts.',
          tags: ['analysis', 'docs'],
          license: 'MIT',
          download_count: 3,
          rating_count: 0,
          latest_version: '0.2.0',
          updated_at: '2026-04-02T00:00:00Z',
        },
      ],
    });

    await waitFor(() => {
      expect(result.current.catalogDisplayLoading).toBe(false);
    });
  });

  it('returns to the encoded redirect target after a successful login', async () => {
    vi.mocked(loginUser).mockResolvedValue({
      token: 'token-1',
      user: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
      },
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'auth', mode: 'login' },
        '?redirect=%2Fsettings%2Fapi-keys',
        navigate,
      ),
    );

    act(() => {
      result.current.setAuthForm({
        username: '',
        displayNameZh: '',
        email: 'test@example.com',
        password: 'secret123',
      });
    });

    await act(async () => {
      await result.current.submitAuth('login');
    });

    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith('/settings/api-keys');
    });
  });

  it('submits the required Chinese display name when registering', async () => {
    vi.mocked(fetchCurrentUser).mockResolvedValueOnce({
      id: 'user-2',
      username: 'newuser',
      display_name_zh: '新用户',
      email: 'new@example.com',
      is_admin: false,
      is_super_admin: false,
      created_at: '2026-03-20T10:00:00Z',
    });
    vi.mocked(registerUser).mockResolvedValue({
      token: 'token-2',
      user: {
        id: 'user-2',
        username: 'newuser',
        display_name_zh: '新用户',
        email: 'new@example.com',
      },
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'auth', mode: 'register' }, '', navigate),
    );

    act(() => {
      result.current.setAuthForm({
        username: 'newuser',
        displayNameZh: '新用户',
        email: 'new@example.com',
        password: 'secret123',
      });
    });

    await act(async () => {
      await result.current.submitAuth('register');
    });

    await waitFor(() => {
      expect(registerUser).toHaveBeenCalledWith({
        username: 'newuser',
        displayNameZh: '新用户',
        email: 'new@example.com',
        password: 'secret123',
      });
      expect(result.current.currentUser?.display_name_zh).toBe('新用户');
    });
  });

  it('uses server-side user stats in the account summary', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(fetchCurrentUserStats).mockResolvedValueOnce({
      user_id: 'user-1',
      username: 'testuser',
      display_name_zh: '测试用户',
      skill_count: 3,
      public_skill_count: 2,
      total_like_count: 11,
      total_install_count: 7,
      total_download_count: 19,
      average_rating: 4.5,
      total_rating_count: 4,
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'settings', section: 'profile' }, '', navigate),
    );

    await waitFor(() => {
      expect(result.current.accountStats.total).toBe(3);
      expect(result.current.accountStats.privateCount).toBe(1);
      expect(result.current.accountStats.totalLikes).toBe(11);
      expect(result.current.accountStats.totalInstalls).toBe(7);
    });
  });

  it('updates the current user profile through the account settings action', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(updateCurrentUserProfile).mockResolvedValueOnce({
      id: 'user-1',
      username: 'testuser',
      display_name_zh: '新中文名',
      email: 'test@example.com',
      avatar_url: 'https://example.com/avatar.png',
      is_admin: false,
      is_super_admin: false,
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'settings', section: 'profile' }, '', navigate),
    );

    await waitFor(() => {
      expect(result.current.currentUser?.username).toBe('testuser');
    });

    act(() => {
      result.current.setProfileForm({
        displayNameZh: '新中文名',
        avatarUrl: 'https://example.com/avatar.png',
      });
    });

    await act(async () => {
      await result.current.submitProfileUpdate();
    });

    expect(updateCurrentUserProfile).toHaveBeenCalledWith('token-1', {
      displayNameZh: '新中文名',
      avatarUrl: 'https://example.com/avatar.png',
    });
    expect(result.current.currentUser?.display_name_zh).toBe('新中文名');
    expect(result.current.profileSuccess).toBe('个人资料已更新。');
  });

  it('updates the current user password through the account settings action', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(updateCurrentUserPassword).mockResolvedValueOnce({ message: 'Password updated' });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'settings', section: 'profile' }, '', navigate),
    );

    await waitFor(() => {
      expect(result.current.currentUser?.username).toBe('testuser');
    });

    act(() => {
      result.current.setPasswordForm({
        currentPassword: 'old-password',
        newPassword: 'new-password',
        confirmPassword: 'new-password',
      });
    });

    await act(async () => {
      await result.current.submitPasswordUpdate();
    });

    expect(updateCurrentUserPassword).toHaveBeenCalledWith('token-1', {
      currentPassword: 'old-password',
      newPassword: 'new-password',
    });
    expect(result.current.passwordForm.newPassword).toBe('');
    expect(result.current.passwordSuccess).toBe('密码已更新。');
  });

  it('loads and updates admin users for super admins', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(fetchCurrentUser).mockResolvedValueOnce({
      id: 'admin-1',
      username: 'root',
      display_name_zh: '平台超管',
      email: 'root@example.com',
      is_admin: true,
      is_super_admin: true,
      created_at: '2026-03-20T10:00:00Z',
    });
    vi.mocked(fetchAdminUsers).mockResolvedValueOnce({
      total: 1,
      page: 1,
      per_page: 20,
      results: [
        {
          id: 'user-1',
          username: 'member',
          display_name_zh: '成员',
          email: 'member@example.com',
          role: 'member',
          is_active: true,
          is_admin: false,
          is_super_admin: false,
          created_at: '2026-03-20T10:00:00Z',
        },
      ],
    });
    vi.mocked(fetchAdminAuditLogs).mockResolvedValueOnce({
      total: 0,
      page: 1,
      per_page: 8,
      results: [],
    });
    vi.mocked(updateAdminUser).mockResolvedValueOnce({
      id: 'user-1',
      username: 'member',
      display_name_zh: '成员',
      email: 'member@example.com',
      role: 'admin',
      is_active: true,
      is_admin: true,
      is_super_admin: false,
      created_at: '2026-03-20T10:00:00Z',
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'settings', section: 'users' }, '', navigate),
    );

    await waitFor(() => {
      expect(fetchAdminUsers).toHaveBeenCalledWith('token-1', expect.objectContaining({
        page: 1,
        role: 'all',
        status: 'all',
      }));
      expect(result.current.adminUsers[0]?.username).toBe('member');
    });

    await act(async () => {
      await result.current.updateAdminUserAccess('user-1', { isAdmin: true });
    });

    expect(updateAdminUser).toHaveBeenCalledWith('token-1', 'user-1', { isAdmin: true });
    expect(result.current.adminUsers[0]?.role).toBe('admin');
  });

  it('toggles skill likes from the detail page', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(fetchSkillDetail).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      like_count: 0,
      install_count: 0,
      rating_count: 0,
      latest_version: '1.0.0',
      viewer_liked: false,
      versions: [],
    });
    vi.mocked(likeSkill).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      like_count: 1,
      install_count: 0,
      rating_count: 0,
      latest_version: '1.0.0',
      viewer_liked: true,
      versions: [],
    });
    vi.mocked(unlikeSkill).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      like_count: 0,
      install_count: 0,
      rating_count: 0,
      latest_version: '1.0.0',
      viewer_liked: false,
      versions: [],
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.viewer_liked).toBe(false);
    });

    await act(async () => {
      await result.current.toggleSkillLike();
    });

    await waitFor(() => {
      expect(likeSkill).toHaveBeenCalledWith('token-1', 'testuser', 'github');
      expect(result.current.detailSkill?.viewer_liked).toBe(true);
      expect(result.current.detailSkill?.like_count).toBe(1);
    });

    await act(async () => {
      await result.current.toggleSkillLike();
    });

    await waitFor(() => {
      expect(unlikeSkill).toHaveBeenCalledWith('token-1', 'testuser', 'github');
      expect(result.current.detailSkill?.viewer_liked).toBe(false);
      expect(result.current.detailSkill?.like_count).toBe(0);
    });
  });

  it('records a share event after copying the detail link', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    });
    Object.defineProperty(navigator, 'share', {
      configurable: true,
      value: undefined,
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.name).toBe('github');
    });

    await act(async () => {
      await result.current.shareDetailSkill();
    });

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(expect.stringContaining('/skills/testuser/github'));
      expect(recordShareEvent).toHaveBeenCalledWith('token-1', 'testuser', 'github', 'copy-link');
      expect(result.current.skillShareStatus).toBe('详情链接已复制。');
    });
  });

  it('submits selected publish category and tags with the release payload', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(publishSkill).mockResolvedValue({
      namespace: 'testuser',
      name: 'github',
      version: '1.0.0',
      download_url: '/api/v1/download/testuser/github/1.0.0',
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'publish-new' }, '', navigate),
    );

    act(() => {
      result.current.setPublishForm((current) => ({
        ...current,
        namespace: 'testuser',
        name: 'github',
        description: 'Interact with GitHub using gh.',
        version: '1.0.0',
        license: 'MIT',
        category: 'integration',
        tags: ['api', 'automation'],
      }));
      result.current.setPublishFile(new File(['zip-binary'], 'github.zip', { type: 'application/zip' }));
    });

    await act(async () => {
      await result.current.submitPublish();
    });

    await waitFor(() => {
      expect(publishSkill).toHaveBeenCalledWith('token-1', expect.objectContaining({
        namespace: 'testuser',
        name: 'github',
        category: 'integration',
        tags: ['api', 'automation'],
      }));
    });
  });

  it('submits selected manage category and tags with the update payload', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(updateSkill).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Updated description',
      category: 'ops',
      tags: ['deployment', 'ci-cd'],
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      is_public: true,
      is_deprecated: false,
      versions: [],
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-settings', namespace: 'testuser', skillName: 'github', section: 'general' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.managedSkill?.name).toBe('github');
    });

    act(() => {
      result.current.setManageForm((current) => ({
        ...current,
        description: 'Updated description',
        category: 'ops',
        tags: ['deployment', 'ci-cd'],
        license: 'Apache-2.0',
      }));
    });

    await act(async () => {
      await result.current.submitManage();
    });

    await waitFor(() => {
      expect(updateSkill).toHaveBeenCalledWith(
        'token-1',
        'testuser',
        'github',
        expect.objectContaining({
          description: 'Updated description',
          category: 'ops',
          tags: ['deployment', 'ci-cd'],
          license: 'Apache-2.0',
        }),
      );
    });
  });

  it('submits recommendation changes through the admin recommendation endpoint', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(fetchCurrentUser).mockResolvedValue({
      id: 'admin-1',
      username: 'catalog-admin',
      email: 'admin@example.com',
      is_admin: true,
      is_super_admin: false,
      created_at: '2026-03-20T10:00:00Z',
    });
    vi.mocked(fetchSkillDetail).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      owner_id: 'owner-1',
      description: 'Interact with GitHub using gh.',
      category: 'ops',
      tags: ['deployment'],
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      is_public: true,
      is_recommended: false,
      versions: [],
    });
    vi.mocked(updateSkillRecommendation).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      owner_id: 'owner-1',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      is_public: true,
      is_recommended: true,
      versions: [],
    } as never);

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-settings', namespace: 'testuser', skillName: 'github', section: 'access' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.managedSkill?.name).toBe('github');
      expect(result.current.currentUser?.is_admin).toBe(true);
    });

    act(() => {
      result.current.setManageForm((current) => ({
        ...current,
        isRecommended: true,
      }));
    });

    await act(async () => {
      await result.current.submitManageRecommendation();
    });

    await waitFor(() => {
      expect(updateSkillRecommendation).toHaveBeenCalledWith('token-1', 'testuser', 'github', {
        isRecommended: true,
      });
    });
  });

  it('uses an absolute download_url directly on the object page', () => {
    render(
      <SkillOverviewPage
        model={{
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            download_url: 'https://oss.example.com/downloads/testuser/github/1.0.0.zip',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: false,
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'passed',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
        }}
        navigate={vi.fn()}
      />,
    );

    expect(screen.getByRole('link', { name: '下载 ZIP' })).toHaveAttribute(
      'href',
      'https://oss.example.com/downloads/testuser/github/1.0.0.zip',
    );
    expect(screen.getByRole('button', { name: '复制下载链接' })).toHaveAttribute(
      'data-value',
      'https://oss.example.com/downloads/testuser/github/1.0.0.zip',
    );
  });

  it('prefixes a relative download_url on the object page with the API base', () => {
    render(
      <SkillOverviewPage
        model={{
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            download_url: '/api/v1/download/testuser/github/1.0.0?format=zip',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: false,
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'passed',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
        }}
        navigate={vi.fn()}
      />,
    );

    expect(screen.getByRole('link', { name: '下载 ZIP' })).toHaveAttribute(
      'href',
      `${mockRegistryBase}/api/v1/download/testuser/github/1.0.0?format=zip`,
    );
    expect(screen.getByRole('button', { name: '复制下载链接' })).toHaveAttribute(
      'data-value',
      `${mockRegistryBase}/api/v1/download/testuser/github/1.0.0?format=zip`,
    );
  });

  it('does not prefix an absolute publish download_url with the API base', () => {
    render(
      <PublishNewPage
        model={{
          token: 'token-1',
          publishError: null,
          publishLoading: false,
          publishSuccess: {
            namespace: 'testuser',
            name: 'github',
            version: '1.0.0',
            download_url: 'https://oss.example.com/downloads/testuser/github/1.0.0.zip',
          },
          publishForm: {
            namespace: 'testuser',
            name: 'github',
            description: '',
            descriptionZh: '',
            category: '',
            version: '1.0.0',
            license: 'MIT',
            tags: [],
            isPublic: true,
            isOwnerOnly: false,
          },
          setPublishForm: vi.fn(),
          setPublishFile: vi.fn(),
          submitPublish: vi.fn(),
        }}
        navigate={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: '复制下载链接' })).toHaveAttribute(
      'data-value',
      'https://oss.example.com/downloads/testuser/github/1.0.0.zip',
    );
  });

  it('prefixes a relative publish download_url with the API base', () => {
    render(
      <PublishNewPage
        model={{
          token: 'token-1',
          publishError: null,
          publishLoading: false,
          publishSuccess: {
            namespace: 'testuser',
            name: 'github',
            version: '1.0.0',
            download_url: '/api/v1/download/testuser/github/1.0.0?format=zip',
          },
          publishForm: {
            namespace: 'testuser',
            name: 'github',
            description: '',
            descriptionZh: '',
            category: '',
            version: '1.0.0',
            license: 'MIT',
            tags: [],
            isPublic: true,
            isOwnerOnly: false,
          },
          setPublishForm: vi.fn(),
          setPublishFile: vi.fn(),
          submitPublish: vi.fn(),
        }}
        navigate={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: '复制下载链接' })).toHaveAttribute(
      'data-value',
      `${mockRegistryBase}/api/v1/download/testuser/github/1.0.0?format=zip`,
    );
  });

  it('submits a community tag against the current skill and updates the detail model', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(addCommunityTag).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      versions: [],
      community_tags: [{ tag: 'deployment', count: 1 }],
      viewer_tags: ['deployment'],
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.name).toBe('github');
    });

    await act(async () => {
      await result.current.submitCommunityTag('deployment');
    });

    await waitFor(() => {
      expect(addCommunityTag).toHaveBeenCalledWith('token-1', 'testuser', 'github', {
        tag: 'deployment',
      });
      expect(result.current.detailSkill?.community_tags?.[0].tag).toBe('deployment');
      expect(result.current.detailSkill?.viewer_tags).toEqual(['deployment']);
    });
  });

  it('submits a skill rating and updates the detail model with the user rating', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    mockedRateSkill.mockResolvedValue({
      skill: {
        id: 'skill-1',
        namespace: 'testuser',
        name: 'github',
        description: 'Interact with GitHub using gh.',
        download_count: 18,
        rating: 4.5,
        rating_count: 2,
        latest_version: '1.0.0',
        versions: [],
      },
      user_rating: {
        id: 'rating-1',
        skill_id: 'skill-1',
        user_id: 'user-1',
        rating: 5,
        comment: 'Great fit for deployment checks.',
      },
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.name).toBe('github');
    });

    await act(async () => {
      await (
        result.current as typeof result.current & {
          submitSkillRating: (rating: number, comment?: string) => Promise<void>;
        }
      ).submitSkillRating(5, 'Great fit for deployment checks.');
    });

    await waitFor(() => {
      expect(mockedRateSkill).toHaveBeenCalledWith('token-1', 'testuser', 'github', {
        rating: 5,
        comment: 'Great fit for deployment checks.',
      });
      expect(result.current.detailSkill?.rating).toBe(4.5);
      expect(result.current.detailSkill?.rating_count).toBe(2);
      expect(result.current.detailSkill?.user_rating?.rating).toBe(5);
      expect(result.current.detailSkill?.user_rating?.comment).toBe('Great fit for deployment checks.');
    });
  });

  it('redirects unauthenticated viewers to login when they try to rate a skill', async () => {
    window.history.pushState({}, '', '/skills/testuser/github?tag=automation');

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.name).toBe('github');
    });

    await act(async () => {
      await (
        result.current as typeof result.current & {
          submitSkillRating: (rating: number, comment?: string) => Promise<void>;
        }
      ).submitSkillRating(4, 'Looks promising');
    });

    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith('/login?redirect=%2Fskills%2Ftestuser%2Fgithub%3Ftag%3Dautomation');
      expect(
        (
          result.current as typeof result.current & {
            skillRatingError: string | null;
          }
        ).skillRatingError,
      ).toBe('请先登录，再为 skill 评分。');
    });
  });

  it('removes one of the viewer community tags from the current skill detail', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.mocked(fetchSkillDetail).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      versions: [],
      community_tags: [{ tag: 'deployment', count: 3 }],
      viewer_tags: ['deployment'],
    });
    vi.mocked(removeCommunityTag).mockResolvedValue({
      id: 'skill-1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      versions: [],
      community_tags: [{ tag: 'deployment', count: 2 }],
      viewer_tags: [],
    });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-tab', namespace: 'testuser', skillName: 'github', tab: 'overview' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.detailSkill?.name).toBe('github');
      expect(result.current.detailSkill?.viewer_tags).toEqual(['deployment']);
    });

    await act(async () => {
      await result.current.removeCommunityTag('deployment');
    });

    await waitFor(() => {
      expect(removeCommunityTag).toHaveBeenCalledWith('token-1', 'testuser', 'github', 'deployment');
      expect(result.current.detailSkill?.viewer_tags).toEqual([]);
    });
  });

  it('leaves deleted skills settings routes instead of staying on the removed object', async () => {
    window.localStorage.setItem('skill-home-web-token', 'token-1');
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.mocked(deleteSkill).mockResolvedValue({ message: 'deleted' });

    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp(
        { name: 'skill-settings', namespace: 'testuser', skillName: 'github', section: 'danger' },
        '',
        navigate,
      ),
    );

    await waitFor(() => {
      expect(result.current.managedSkill?.name).toBe('github');
    });

    await act(async () => {
      await result.current.removeManagedSkill();
    });

    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith('/settings/profile');
    });
  });
});
