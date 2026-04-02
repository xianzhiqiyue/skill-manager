import { act, cleanup, render, renderHook, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  addCommunityTag,
  deleteSkill,
  fetchSkillDetail,
  loginUser,
  publishSkill,
  removeCommunityTag,
  updateSkill,
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
    fetchCurrentUser: vi.fn().mockResolvedValue({
      id: 'user-1',
      username: 'testuser',
      email: 'test@example.com',
      created_at: '2026-03-20T10:00:00Z',
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
      description: 'Interact with GitHub using gh.',
      category: 'integration',
      tags: ['api'],
      download_count: 18,
      rating_count: 0,
      latest_version: '1.0.0',
      versions: [],
    }),
    fetchSkills: vi.fn().mockResolvedValue({
      total: 0,
      results: [],
    }),
    loginUser: vi.fn(),
    publishSkill: vi.fn(),
    rateSkill: mockedRateSkill,
    addCommunityTag: vi.fn(),
    removeCommunityTag: vi.fn(),
    registerUser: vi.fn(),
    revokeAPIKey: vi.fn(),
    updateSkill: vi.fn(),
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

  it('writes non-default catalog filters back to the skills URL', async () => {
    const navigate = vi.fn();
    const { result } = renderHook(() =>
      useRegistryApp({ name: 'skills' }, '?q=doc', navigate),
    );

    act(() => {
      result.current.updateCatalogFilter('sort', 'updated');
    });

    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith('/skills?q=doc&sort=updated', { replace: true });
    });
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
    const { result, rerender } = renderHook(
      ({ route, search }: { route: { name: 'home' } | { name: 'skills' }; search: string }) =>
        useRegistryApp(route, search, navigate),
      {
        initialProps: {
          route: { name: 'home' } as const,
          search: '',
        },
      },
    );

    rerender({
      route: { name: 'skills' } as const,
      search: '?q=fmea',
    });

    expect(result.current.catalogFilters.query).toBe('fmea');
    expect(navigate).not.toHaveBeenCalledWith('/skills', { replace: true });
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
            category: '',
            version: '1.0.0',
            license: 'MIT',
            tags: [],
            isPublic: true,
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
            category: '',
            version: '1.0.0',
            license: 'MIT',
            tags: [],
            isPublic: true,
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
