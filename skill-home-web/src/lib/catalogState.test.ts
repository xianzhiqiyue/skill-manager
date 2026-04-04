import { describe, expect, it } from 'vitest';

import { filterCatalogSkills, parseCatalogSearch, toCatalogSearch } from './catalogState';

describe('catalog search params', () => {
  it('reads known filters from the URL query string', () => {
    expect(
      parseCatalogSearch('?q=doc&namespace=testuser&sort=updated&view=cards'),
    ).toMatchObject({
      query: 'doc',
      namespace: 'testuser',
      tag: 'all',
      license: 'all',
      sort: 'updated',
      view: 'cards',
    });
  });

  it('serializes non-default filters into the URL query string', () => {
    expect(
      toCatalogSearch({
        query: 'github',
        namespace: 'testuser',
        tag: 'all',
        license: 'MIT',
        sort: 'updated',
        view: 'list',
      }),
    ).toBe('?q=github&namespace=testuser&license=MIT&sort=updated');
  });

  it('omits default values when serializing filters', () => {
    expect(
      toCatalogSearch({
        query: '',
        namespace: 'all',
        tag: 'all',
        license: 'all',
        sort: 'downloads',
        view: 'list',
      }),
    ).toBe('');
  });

  it('filters and sorts visible skills against the active catalog query', () => {
    const visible = filterCatalogSkills(
      [
        {
          id: '1',
          namespace: 'testuser',
          name: 'github',
          description: 'Interact with GitHub using gh.',
          tags: ['automation'],
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
      {
        query: 'fmea',
        namespace: 'all',
        tag: 'all',
        license: 'all',
        sort: 'downloads',
        view: 'list',
      },
    );

    expect(visible).toHaveLength(1);
    expect(visible[0]?.name).toBe('openclaw-fmea-cocreator');
  });

  it('keeps recommended skills ahead of other matches for the same sort mode', () => {
    const visible = filterCatalogSkills(
      [
        {
          id: '1',
          namespace: 'team',
          name: 'deploy-helper',
          download_count: 20,
          rating_count: 0,
          is_recommended: false,
          updated_at: '2026-04-01T00:00:00Z',
        },
        {
          id: '2',
          namespace: 'team',
          name: 'review-helper',
          download_count: 5,
          rating_count: 0,
          is_recommended: true,
          updated_at: '2026-03-01T00:00:00Z',
        },
      ],
      {
        query: '',
        namespace: 'all',
        tag: 'all',
        license: 'all',
        sort: 'downloads',
        view: 'list',
      },
    );

    expect(visible.map((skill) => skill.name)).toEqual(['review-helper', 'deploy-helper']);
  });
});
