import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  addCommunityTag,
  deleteSkill,
  fetchSkillDetail,
  loginUser,
  publishSkill,
  removeCommunityTag,
} from '../api';
import { useRegistryApp } from './useRegistryApp';

vi.mock('../api', () => ({
  API_BASE: 'http://example.com',
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
  addCommunityTag: vi.fn(),
  removeCommunityTag: vi.fn(),
  registerUser: vi.fn(),
  revokeAPIKey: vi.fn(),
  updateSkill: vi.fn(),
}));

describe('useRegistryApp catalog URL sync', () => {
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

  it('submits parsed publish tags with the release payload', async () => {
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
        tags: 'automation, github',
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
        tags: ['automation', 'github'],
      }));
    });
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
