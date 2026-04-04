import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { SkillsSearchPage, type SkillsSearchPageModel } from '../../pages/SkillsSearchPage';

const model: SkillsSearchPageModel = {
  skills: [
    {
      id: '1',
      namespace: 'testuser',
      name: 'github',
      description: 'Interact with GitHub using gh.',
      tags: ['automation', 'github'],
      license: 'MIT',
      download_count: 18,
      rating: 4.8,
      rating_count: 12,
      latest_version: '1.0.0',
      is_recommended: true,
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
  setCatalogQuery: vi.fn(),
  updateCatalogFilter: vi.fn(),
  resetCatalogFilters: vi.fn(),
  refreshCatalog: vi.fn(),
  openSkill: vi.fn(),
};

describe('SkillsSearchPage', () => {
  afterEach(() => {
    cleanup();
  });

  it('shows a left filter rail and compact result rows', () => {
    render(<SkillsSearchPage model={model} />);

    expect(screen.getAllByText('Filter by')[0]).toBeInTheDocument();
    expect(screen.getByText('1 结果')).toBeInTheDocument();
    expect(screen.queryByText('查看详情')).not.toBeInTheDocument();
  });

  it('shows rating signals in search results and weak copy for unrated skills', () => {
    render(
      <SkillsSearchPage
        model={{
          ...model,
          skills: [
            ...model.skills,
            {
              id: '2',
              namespace: 'testuser',
              name: 'docs-helper',
              description: 'Keeps docs tidy.',
              tags: ['docs'],
              license: 'MIT',
              download_count: 5,
              rating_count: 0,
              latest_version: '0.4.0',
              updated_at: '2026-03-20T21:32:00Z',
            },
          ],
          skillsTotal: 2,
        }}
      />,
    );

    expect(screen.getAllByText('4.8 分').length).toBeGreaterThan(0);
    expect(screen.getAllByText('12 人评分').length).toBeGreaterThan(0);
    expect(screen.getAllByText('暂无评分').length).toBeGreaterThan(0);
  });

  it('marks recommended results explicitly in the search list', () => {
    render(<SkillsSearchPage model={model} />);

    expect(screen.getByText('推荐')).toBeInTheDocument();
  });

  it('renders the committed catalog result set from the model as-is', () => {
    render(
      <SkillsSearchPage
        model={{
          ...model,
          skills: [
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
          catalogFilters: {
            ...model.catalogFilters,
            query: 'fmea',
          },
          skillsTotal: 1,
        }}
      />,
    );

    expect(screen.getByText('@zhuyuxiao314/openclaw-fmea-cocreator')).toBeInTheDocument();
    expect(screen.queryByText('@testuser/github')).not.toBeInTheDocument();
    expect(screen.getByText('1 结果')).toBeInTheDocument();
    expect(screen.getByText('关键词：fmea')).toBeInTheDocument();
  });

  it('keeps current results visible while a filtered refresh is in flight', () => {
    render(
      <SkillsSearchPage
        model={{
          ...model,
          skillsLoading: true,
        }}
      />,
    );

    expect(screen.getAllByText('@testuser/github').length).toBeGreaterThan(0);
    expect(screen.queryByText('正在读取技能目录...')).not.toBeInTheDocument();
  });
});
