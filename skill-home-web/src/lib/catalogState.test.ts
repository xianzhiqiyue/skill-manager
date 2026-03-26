import { describe, expect, it } from 'vitest';

import { parseCatalogSearch, toCatalogSearch } from './catalogState';

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
});
