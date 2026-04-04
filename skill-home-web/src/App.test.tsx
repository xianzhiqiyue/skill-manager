import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import App from './App';

const mockLocalRegistryBase = 'http://127.0.0.1:8080';

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
  ],
  skillsTotal: 12,
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
  quickStats: { namespaceCount: 3, licenseCount: 1, tagCount: 2 },
  featuredSkills: [
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
  ],
  latestSkills: [
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
  ],
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

vi.mock('./hooks/useRoute', () => ({
  useRoute: () => mockUseRoute(),
}));

vi.mock('./hooks/useRegistryApp', () => ({
  useRegistryApp: (...args: unknown[]) => mockUseRegistryApp(...args),
}));

function renderApp() {
  return render(<App />);
}

describe('App shell', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('renders a dark global header and no longer uses the old soft hero shell', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'home' },
      location: { pathname: '/', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    expect(screen.getByRole('banner')).toHaveClass('gh-header');
    expect(screen.queryByText('把 skill 变成能被搜索、安装和持续运营的产品入口')).not.toBeInTheDocument();
  });

  it('keeps the home page concise and routes users into search instead of repeating long explainer sections', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'home' },
      location: { pathname: '/', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    expect(screen.getByRole('heading', { name: 'Find and ship skills faster' })).toBeInTheDocument();
    expect(screen.queryByText('精选技能')).not.toBeInTheDocument();
  });

  it('surfaces the CLI install command on the home page and links to the install workspace', () => {
    const navigate = vi.fn();
    mockUseRoute.mockReturnValue({
      route: { name: 'home' },
      location: { pathname: '/', search: '' },
      navigate,
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    expect(screen.getByRole('heading', { name: 'Install CLI' })).toBeInTheDocument();
    expect(screen.getByText(`curl -fsSL ${mockLocalRegistryBase}/install.sh | bash`)).toBeInTheDocument();
    expect(screen.getByText('Skill Home /releases')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Open install guide' }));

    expect(navigate).toHaveBeenCalledWith('/install');
  });

  it('treats the header search as a draft until submit and only then applies the catalog query', () => {
    const navigate = vi.fn();
    mockUseRoute.mockReturnValue({
      route: { name: 'home' },
      location: { pathname: '/', search: '' },
      navigate,
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    const input = screen.getByRole('searchbox', { name: '搜索 skill、能力、场景' });
    fireEvent.change(input, { target: { value: 'github' } });

    expect(baseModel.setCatalogQuery).not.toHaveBeenCalled();

    fireEvent.submit(input.closest('form')!);

    expect(baseModel.setCatalogQuery).not.toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith('/skills?q=github');
  });

  it('updates the in-place catalog query instead of re-navigating when the header search submits on /skills', () => {
    const navigate = vi.fn();
    const setCatalogQuery = vi.fn();
    mockUseRoute.mockReturnValue({
      route: { name: 'skills' },
      location: { pathname: '/skills', search: '' },
      navigate,
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      setCatalogQuery,
    });

    renderApp();

    const input = screen.getByRole('searchbox', { name: '搜索 skill、能力、场景' });
    fireEvent.change(input, { target: { value: 'fmea' } });
    fireEvent.submit(input.closest('form')!);

    expect(setCatalogQuery).toHaveBeenCalledWith('fmea');
    expect(navigate).not.toHaveBeenCalled();
  });

  it('renders the GitHub-style object shell for the canonical skill-tab route', () => {
    mockUseRoute.mockReturnValue({
      route: {
        name: 'skill-tab',
        namespace: 'testuser',
        skillName: 'github',
        tab: 'overview',
      },
      location: { pathname: '/skills/testuser/github', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      detailSkill: {
        ...baseModel.skills[0],
        versions: [
          {
            id: 'v1',
            version: '1.0.0',
            size_bytes: 4096,
            scan_status: 'passed',
            created_at: '2026-03-22T21:32:00Z',
          },
        ],
      },
    });

    renderApp();

    expect(screen.getByRole('heading', { name: 'testuser / github' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Versions' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Install' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Activity' })).toBeInTheDocument();
    const metadata = screen.getByRole('region', { name: 'Metadata' });
    expect(metadata).toHaveTextContent('Namespace');
    expect(metadata).toHaveTextContent('Latest version');
    expect(
      screen.queryByText('先打通 CLI 和 registry，再优先复制主安装命令；其余命令只做排查和辅助。'),
    ).not.toBeInTheDocument();
  });

  it('renders the install tab inside the same object shell instead of the legacy detail page', () => {
    mockUseRoute.mockReturnValue({
      route: {
        name: 'skill-tab',
        namespace: 'testuser',
        skillName: 'github',
        tab: 'install',
      },
      location: { pathname: '/skills/testuser/github/install', search: '?q=github&tag=automation' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      detailSkill: {
        ...baseModel.skills[0],
        versions: [
          {
            id: 'v1',
            version: '1.0.0',
            size_bytes: 4096,
            scan_status: 'passed',
            created_at: '2026-03-22T21:32:00Z',
          },
        ],
      },
    });

    renderApp();

    expect(screen.getByRole('heading', { name: 'testuser / github' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Install' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText('Install command set')).toBeInTheDocument();
    expect(screen.queryByText('AI 安装指引')).not.toBeInTheDocument();
  });

  it('renders the GitHub-style skills search workspace on the /skills route', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'skills' },
      location: { pathname: '/skills', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    expect(screen.getAllByText('Filter by')[0]).toBeInTheDocument();
    expect(screen.getByText('1 结果')).toBeInTheDocument();
    expect(screen.queryByText('查看详情')).not.toBeInTheDocument();
  });

  it('does not reset the active search workspace when clicking 技能中心 from a filtered catalog page', () => {
    const navigate = vi.fn();
    mockUseRoute.mockReturnValue({
      route: { name: 'skills' },
      location: { pathname: '/skills', search: '?q=fmea' },
      navigate,
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      catalogFilters: {
        ...baseModel.catalogFilters,
        query: 'fmea',
      },
    });

    renderApp();

    fireEvent.click(screen.getAllByRole('button', { name: '技能中心' })[0]);

    expect(navigate).not.toHaveBeenCalled();
  });

  it('returns to the filtered catalog when clicking 技能中心 from a skill detail page', () => {
    const navigate = vi.fn();
    const returnToCatalog = vi.fn();
    mockUseRoute.mockReturnValue({
      route: {
        name: 'skill-tab',
        namespace: 'testuser',
        skillName: 'github',
        tab: 'overview',
      },
      location: { pathname: '/skills/testuser/github', search: '?q=fmea&tag=analysis' },
      navigate,
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      returnToCatalog,
      detailSkill: {
        ...baseModel.skills[0],
        versions: [],
      },
    });

    renderApp();

    fireEvent.click(screen.getAllByRole('button', { name: '技能中心' })[0]);

    expect(returnToCatalog).toHaveBeenCalledTimes(1);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('renders an account settings shell on /settings/profile instead of the legacy console dashboard', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'settings', section: 'profile' },
      location: { pathname: '/settings/profile', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      token: 'token',
      currentUser: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        created_at: '2026-03-20T10:00:00Z',
      },
      mySkills: baseModel.skills,
      managedSkillKey: 'testuser/github',
      managedSkill: {
        ...baseModel.skills[0],
        versions: [
          {
            id: 'v1',
            version: '1.0.0',
            size_bytes: 4096,
            scan_status: 'passed',
            created_at: '2026-03-22T21:32:00Z',
          },
        ],
      },
      accountStats: { total: 1, publicCount: 1, privateCount: 0 },
    });

    renderApp();

    expect(screen.getByRole('navigation', { name: 'Settings' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Profile' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText('Your skills')).toBeInTheDocument();
    expect(screen.queryByText('我的技能控制台')).not.toBeInTheDocument();
  });

  it('renders a dedicated skill danger settings page on the canonical skill-settings route', () => {
    mockUseRoute.mockReturnValue({
      route: {
        name: 'skill-settings',
        namespace: 'testuser',
        skillName: 'github',
        section: 'danger',
      },
      location: { pathname: '/settings/skills/testuser/github/danger', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      token: 'token',
      currentUser: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
        created_at: '2026-03-20T10:00:00Z',
      },
      mySkills: baseModel.skills,
      managedSkillKey: 'testuser/github',
      managedSkill: {
        ...baseModel.skills[0],
        versions: [
          {
            id: 'v1',
            version: '1.0.0',
            size_bytes: 4096,
            scan_status: 'passed',
            created_at: '2026-03-22T21:32:00Z',
          },
        ],
      },
      accountStats: { total: 1, publicCount: 1, privateCount: 0 },
    });

    renderApp();

    expect(screen.getByRole('navigation', { name: 'Skill settings' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Danger Zone' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText('Delete this skill')).toBeInTheDocument();
    expect(screen.queryByText('基础信息')).not.toBeInTheDocument();
  });

  it('renders API key creation and existing keys inside one unified card with creation first', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'settings', section: 'api-keys' },
      location: { pathname: '/settings/api-keys', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      token: 'token',
      currentUser: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
      },
      apiKeys: [
        {
          id: 'key-1',
          name: 'ci',
          prefix: 'sk_test',
          created_at: '2026-03-26T10:00:00Z',
          last_used_at: '2026-03-26T11:00:00Z',
          expires_at: null,
        },
      ],
      apiKeyStats: { total: 1, active: 1, expiringSoon: 0 },
    });

    const { container } = renderApp();

    const createHeading = screen.getByRole('heading', { name: 'Create new key' });
    const existingHeading = screen.getByRole('heading', { name: 'Existing keys' });
    const cards = container.querySelectorAll('.gh-settings-card');

    expect(cards).toHaveLength(1);
    expect(createHeading.closest('.gh-settings-card')).toBe(existingHeading.closest('.gh-settings-card'));
    expect(createHeading.compareDocumentPosition(existingHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('renders publish as a focused creation form under /publish/new', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'publish-new' },
      location: { pathname: '/publish/new', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      token: 'token',
      currentUser: {
        id: 'user-1',
        username: 'testuser',
        email: 'test@example.com',
      },
    });

    renderApp();

    expect(screen.getByRole('heading', { name: 'Create a new release' })).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Category' })).toBeInTheDocument();
    expect(screen.getAllByText('Official tags')).toHaveLength(2);
    expect(screen.queryByText('发布清单')).not.toBeInTheDocument();
  });

  it('renders auth inside the GitHub-style shell instead of the old compact marketing card', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'auth', mode: 'login' },
      location: { pathname: '/login', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue(baseModel);

    renderApp();

    expect(screen.getByText('Sign in to Skill Home')).toBeInTheDocument();
    expect(screen.queryByText('登录 Skill Home')).not.toBeInTheDocument();
  });

  it('renders install docs as an operational workspace under /install', () => {
    mockUseRoute.mockReturnValue({
      route: { name: 'install' },
      location: { pathname: '/install', search: '' },
      navigate: vi.fn(),
    });
    mockUseRegistryApp.mockReturnValue({
      ...baseModel,
      detailSkill: {
        ...baseModel.skills[0],
        versions: [
          {
            id: 'v1',
            version: '1.0.0',
            size_bytes: 4096,
            scan_status: 'passed',
            created_at: '2026-03-22T21:32:00Z',
          },
        ],
      },
    });

    renderApp();

    expect(screen.getByRole('heading', { name: 'Install from the registry' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '安装指南' })).not.toBeInTheDocument();
  });
});
