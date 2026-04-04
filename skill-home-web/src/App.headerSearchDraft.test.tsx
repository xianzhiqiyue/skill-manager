import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { renderSkillsPage } = vi.hoisted(() => ({
  renderSkillsPage: vi.fn(),
}));

const baseModel = {
  health: { service: 'skill-home', status: 'ok', version: '1.0.0' },
  healthError: null,
  token: '',
  currentUser: null,
  authLoading: false,
  authError: null,
  authSuccess: null,
  authForm: { username: '', email: '', password: '' },
  setAuthForm: vi.fn(),
  submitAuth: vi.fn(),
  handleLogout: vi.fn(),
  skills: [
    {
      id: 'skill-1',
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
  ],
  skillsTotal: 1,
  skillsLoading: false,
  skillsError: null,
  catalogFilters: {
    query: '',
    namespace: 'all',
    tag: 'all',
    license: 'all',
    sort: 'downloads',
    view: 'list',
  },
  namespaceOptions: ['testuser'],
  tagOptions: ['automation', 'github'],
  licenseOptions: ['MIT'],
  quickStats: { namespaceCount: 1, licenseCount: 1, tagCount: 2 },
  featuredSkills: [],
  latestSkills: [],
  setCatalogQuery: vi.fn(),
  updateCatalogFilter: vi.fn(),
  resetCatalogFilters: vi.fn(),
  refreshCatalog: vi.fn(),
  openSkill: vi.fn(),
  returnToCatalog: vi.fn(),
  detailSkill: null,
  detailLoading: false,
  detailError: null,
  refreshDetail: vi.fn(),
  accountLoading: false,
  accountError: null,
  mySkills: [],
  apiKeys: [],
  apiKeysLoading: false,
  apiKeysError: null,
  apiKeysSuccess: null,
  apiKeyCreating: false,
  apiKeyRevoking: null,
  revealedAPIKey: null,
  setRevealedAPIKey: vi.fn(),
  apiKeyForm: { name: '', expiryPreset: 'never', customExpiresAt: '' },
  setAPIKeyForm: vi.fn(),
  submitAPIKeyCreate: vi.fn(),
  removeAPIKey: vi.fn(),
  refreshAPIKeys: vi.fn(),
  accountStats: { total: 0, publicCount: 0, privateCount: 0 },
  apiKeyStats: { total: 0, active: 0, expiringSoon: 0 },
  managedSkillKey: null,
  setManagedSkillKey: vi.fn(),
  managedSkill: null,
  manageLoading: false,
  manageSaving: false,
  manageRecommendationSaving: false,
  manageDeletingSkill: false,
  manageDeletingVersion: null,
  manageError: null,
  manageSuccess: null,
  manageForm: {
    description: '',
    category: '',
    license: 'MIT',
    tags: [],
    isPublic: true,
    isDeprecated: false,
    isRecommended: false,
  },
  setManageForm: vi.fn(),
  submitManage: vi.fn(),
  submitManageRecommendation: vi.fn(),
  removeManagedSkill: vi.fn(),
  removeManagedVersion: vi.fn(),
  publishLoading: false,
  publishError: null,
  publishSuccess: null,
  publishForm: {
    namespace: 'testuser',
    name: '',
    description: '',
    category: '',
    version: '0.1.0',
    license: 'MIT',
    tags: [],
    isPublic: true,
  },
  setPublishForm: vi.fn(),
  publishFile: null,
  setPublishFile: vi.fn(),
  submitPublish: vi.fn(),
  relatedSkills: [],
};

const mockUseRoute = vi.fn();
const mockUseRegistryApp = vi.fn();

vi.mock('./pages/SkillsSearchPage', () => ({
  SkillsSearchPage: () => {
    renderSkillsPage();
    return <div data-testid="skills-page">skills page</div>;
  },
}));

vi.mock('./hooks/useRoute', () => ({
  useRoute: () => mockUseRoute(),
}));

vi.mock('./hooks/useRegistryApp', () => ({
  useRegistryApp: (...args: unknown[]) => mockUseRegistryApp(...args),
}));

import App from './App';

describe('App header search draft state', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRoute.mockReturnValue({
      route: { name: 'skills' },
      location: { pathname: '/skills', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue(baseModel);
  });

  afterEach(() => {
    cleanup();
  });

  it('does not rerender the skills page when typing into the global header search draft', () => {
    render(<App />);

    expect(screen.getByTestId('skills-page')).toBeInTheDocument();
    expect(renderSkillsPage).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByRole('searchbox', { name: '搜索 skill、能力、场景' }), {
      target: { value: 'g' },
    });

    expect(renderSkillsPage).toHaveBeenCalledTimes(1);
  });
});
