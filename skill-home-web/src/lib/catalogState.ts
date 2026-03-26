export type CatalogSort = 'downloads' | 'updated' | 'rating' | 'name';
export type CatalogView = 'cards' | 'list';

export type CatalogFilters = {
  query: string;
  namespace: string;
  tag: string;
  license: string;
  sort: CatalogSort;
  view: CatalogView;
};

export const defaultCatalogFilters: CatalogFilters = {
  query: '',
  namespace: 'all',
  tag: 'all',
  license: 'all',
  sort: 'downloads',
  view: 'list',
};

function readParamValue(value: string | null, fallback: string) {
  return value?.trim() || fallback;
}

export function parseCatalogSearch(search: string): CatalogFilters {
  const params = new URLSearchParams(search);
  const sort = readParamValue(params.get('sort'), defaultCatalogFilters.sort);
  const view = readParamValue(params.get('view'), defaultCatalogFilters.view);

  return {
    query: readParamValue(params.get('q'), defaultCatalogFilters.query),
    namespace: readParamValue(params.get('namespace'), defaultCatalogFilters.namespace),
    tag: readParamValue(params.get('tag'), defaultCatalogFilters.tag),
    license: readParamValue(params.get('license'), defaultCatalogFilters.license),
    sort: sort === 'updated' || sort === 'rating' || sort === 'name' ? sort : 'downloads',
    view: view === 'cards' ? 'cards' : 'list',
  };
}

export function toCatalogSearch(filters: CatalogFilters) {
  const params = new URLSearchParams();

  if (filters.query.trim()) {
    params.set('q', filters.query.trim());
  }
  if (filters.namespace !== defaultCatalogFilters.namespace) {
    params.set('namespace', filters.namespace);
  }
  if (filters.tag !== defaultCatalogFilters.tag) {
    params.set('tag', filters.tag);
  }
  if (filters.license !== defaultCatalogFilters.license) {
    params.set('license', filters.license);
  }
  if (filters.sort !== defaultCatalogFilters.sort) {
    params.set('sort', filters.sort);
  }
  if (filters.view !== defaultCatalogFilters.view) {
    params.set('view', filters.view);
  }

  const serialized = params.toString();
  return serialized ? `?${serialized}` : '';
}
