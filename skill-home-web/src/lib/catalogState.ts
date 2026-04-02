import type { SkillSummary } from '../api';

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

function compareCatalogSkills(left: SkillSummary, right: SkillSummary, sort: CatalogSort) {
  if (sort === 'updated') {
    return (
      new Date(right.updated_at || 0).getTime() - new Date(left.updated_at || 0).getTime() ||
      right.download_count - left.download_count ||
      left.name.localeCompare(right.name)
    );
  }

  if (sort === 'rating') {
    return (
      (right.rating || 0) - (left.rating || 0) ||
      right.rating_count - left.rating_count ||
      right.download_count - left.download_count ||
      left.name.localeCompare(right.name)
    );
  }

  if (sort === 'name') {
    return left.name.localeCompare(right.name) || left.namespace.localeCompare(right.namespace);
  }

  return (
    right.download_count - left.download_count ||
    new Date(right.updated_at || 0).getTime() - new Date(left.updated_at || 0).getTime() ||
    left.name.localeCompare(right.name)
  );
}

export function filterCatalogSkills(skills: SkillSummary[], filters: CatalogFilters) {
  const query = filters.query.trim().toLowerCase();

  return skills
    .filter((skill) => {
      if (filters.namespace !== defaultCatalogFilters.namespace && skill.namespace !== filters.namespace) {
        return false;
      }

      if (filters.tag !== defaultCatalogFilters.tag && !(skill.tags || []).includes(filters.tag)) {
        return false;
      }

      const license = skill.license || '';
      if (filters.license !== defaultCatalogFilters.license && license !== filters.license) {
        return false;
      }

      if (!query) {
        return true;
      }

      const reference = `${skill.namespace}/${skill.name}`.toLowerCase();
      const description = (skill.description || '').toLowerCase();
      const tags = (skill.tags || []).map((tag) => tag.toLowerCase());

      return (
        skill.name.toLowerCase().includes(query) ||
        reference.includes(query) ||
        description.includes(query) ||
        tags.some((tag) => tag.includes(query))
      );
    })
    .sort((left, right) => compareCatalogSkills(left, right, filters.sort));
}
