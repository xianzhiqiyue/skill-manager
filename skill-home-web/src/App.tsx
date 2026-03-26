import { useEffect, useState, type FormEvent } from 'react';

import { API_BASE, type SkillSummary } from './api';
import { GlobalHeader } from './components/layout/GlobalHeader';
import { useRegistryApp } from './hooks/useRegistryApp';
import { useRoute } from './hooks/useRoute';
import { toCatalogSearch } from './lib/catalogState';
import { buildAuthPath, buildSkillPath } from './lib/routes';
import { type ToastTone } from './lib/toast';
import { AuthPage as AuthRoutePage } from './pages/AuthPage';
import { HomePage as HomeRoutePage } from './pages/HomePage';
import { InstallDocsPage } from './pages/InstallDocsPage';
import { PublishNewPage } from './pages/PublishNewPage';
import { SkillsSearchPage } from './pages/SkillsSearchPage';
import { APIKeysSettingsPage } from './pages/settings/APIKeysSettingsPage';
import { ProfileSettingsPage } from './pages/settings/ProfileSettingsPage';
import { SkillAccessSettingsPage } from './pages/settings/SkillAccessSettingsPage';
import { SkillDangerSettingsPage } from './pages/settings/SkillDangerSettingsPage';
import { SkillGeneralSettingsPage } from './pages/settings/SkillGeneralSettingsPage';
import { SkillVersionsSettingsPage } from './pages/settings/SkillVersionsSettingsPage';
import { SkillActivityPage } from './pages/skill/SkillActivityPage';
import { SkillInstallPage } from './pages/skill/SkillInstallPage';
import { SkillOverviewPage } from './pages/skill/SkillOverviewPage';
import { SkillVersionsPage } from './pages/skill/SkillVersionsPage';

function StatusBanner({
  tone,
  message,
}: {
  tone: 'success' | 'danger' | 'warning' | 'neutral';
  message: string;
}) {
  return <div className={`status-banner status-banner--${tone}`}>{message}</div>;
}

function buildHeaderSearchSuggestions(skills: SkillSummary[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return [];
  }

  return skills
    .map((skill) => {
      const reference = `${skill.namespace}/${skill.name}`.toLowerCase();
      const name = skill.name.toLowerCase();
      const description = skill.description?.toLowerCase() || '';
      const tags = (skill.tags || []).map((tag) => tag.toLowerCase());

      let score = Number.POSITIVE_INFINITY;
      if (name === normalizedQuery || reference === normalizedQuery) {
        score = 0;
      } else if (name.startsWith(normalizedQuery)) {
        score = 1;
      } else if (reference.startsWith(normalizedQuery)) {
        score = 2;
      } else if (tags.some((tag) => tag.startsWith(normalizedQuery))) {
        score = 3;
      } else if (description.includes(normalizedQuery)) {
        score = 4;
      } else if (reference.includes(normalizedQuery) || tags.some((tag) => tag.includes(normalizedQuery))) {
        score = 5;
      }

      return Number.isFinite(score) ? { score, skill } : null;
    })
    .filter((entry): entry is { score: number; skill: SkillSummary } => entry !== null)
    .sort(
      (left, right) =>
        left.score - right.score ||
        right.skill.download_count - left.skill.download_count ||
        left.skill.name.localeCompare(right.skill.name),
    )
    .slice(0, 5)
    .map(({ skill }) => ({
      id: skill.id,
      namespace: skill.namespace,
      name: skill.name,
      description: skill.description,
      latestVersion: skill.latest_version,
    }));
}

export default function App() {
  const { route, location, navigate } = useRoute();
  const model = useRegistryApp(route, location.search, navigate);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);
  const [headerSearchValue, setHeaderSearchValue] = useState(model.catalogFilters.query);
  const [toast, setToast] = useState<{ tone: ToastTone; message: string } | null>(null);
  const headerSearchSuggestions = buildHeaderSearchSuggestions(model.skills, headerSearchValue);
  const activeNav =
    route.name === 'skill-tab'
      ? 'skills'
      : route.name === 'settings' || route.name === 'skill-settings'
        ? 'settings'
        : route.name === 'publish-new'
          ? 'publish'
          : route.name === 'home' || route.name === 'skills' || route.name === 'install'
            ? route.name
            : null;

  useEffect(() => {
    let timer = 0;

    function handleToast(event: Event) {
      const detail = (event as CustomEvent<{ tone: ToastTone; message: string }>).detail;
      if (!detail) {
        return;
      }

      setToast(detail);
      window.clearTimeout(timer);
      timer = window.setTimeout(() => setToast(null), 2200);
    }

    window.addEventListener('skill-home-toast', handleToast);

    return () => {
      window.removeEventListener('skill-home-toast', handleToast);
      window.clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    setHeaderSearchValue(model.catalogFilters.query);
  }, [location.search, model.catalogFilters.query, route.name]);

  function navigateInternal(path: string) {
    setMobileNavOpen(false);
    setMobileSearchOpen(false);
    navigate(path);
  }

  function handleLogout() {
    setMobileNavOpen(false);
    setMobileSearchOpen(false);
    model.handleLogout();
  }

  function buildHeaderSearch(searchPath: string) {
    return `${searchPath}${toCatalogSearch({
      ...model.catalogFilters,
      query: headerSearchValue,
    })}`;
  }

  function submitGlobalSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigateInternal(buildHeaderSearch('/skills'));
  }

  return (
    <div className="app-frame">
      {toast ? (
        <div className="toast-stack" aria-live="polite">
          <StatusBanner message={toast.message} tone={toast.tone} />
        </div>
      ) : null}

      <GlobalHeader
        activeNav={activeNav}
        currentUser={model.currentUser}
        mobileNavOpen={mobileNavOpen}
        mobileSearchOpen={mobileSearchOpen}
        onConsole={() => navigateInternal('/settings/profile')}
        onHome={() => navigateInternal('/')}
        onInstall={() => navigateInternal('/install')}
        onLogin={() => navigateInternal(buildAuthPath('login', `${location.pathname}${location.search}`))}
        onLogout={handleLogout}
        onPublish={() => navigateInternal('/publish/new')}
        onRegister={() => navigateInternal(buildAuthPath('register', `${location.pathname}${location.search}`))}
        onSearchChange={(nextValue) => setHeaderSearchValue(nextValue)}
        onSearchSuggestionSelect={(namespace, name) => {
          navigateInternal(buildHeaderSearch(buildSkillPath(namespace, name)));
        }}
        onSearchSubmit={submitGlobalSearch}
        onSkills={() => navigateInternal('/skills')}
        onToggleMobileNav={() => {
          setMobileNavOpen((value) => !value);
          setMobileSearchOpen(false);
        }}
        onToggleMobileSearch={() => {
          setMobileSearchOpen((value) => !value);
          setMobileNavOpen(false);
        }}
        searchSuggestions={headerSearchSuggestions}
        searchValue={headerSearchValue}
      />

      <main className="app-main">
        {route.name === 'home' ? <HomeRoutePage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'skills' ? <SkillsSearchPage model={model} /> : null}
        {route.name === 'skill-tab' && route.tab === 'overview' ? (
          <SkillOverviewPage model={model} navigate={navigateInternal} search={location.search} />
        ) : null}
        {route.name === 'skill-tab' && route.tab === 'versions' ? (
          <SkillVersionsPage model={model} navigate={navigateInternal} search={location.search} />
        ) : null}
        {route.name === 'skill-tab' && route.tab === 'install' ? (
          <SkillInstallPage model={model} navigate={navigateInternal} search={location.search} />
        ) : null}
        {route.name === 'skill-tab' && route.tab === 'activity' ? (
          <SkillActivityPage model={model} navigate={navigateInternal} search={location.search} />
        ) : null}
        {route.name === 'publish-new' ? (
          <PublishNewPage model={model} navigate={navigateInternal} />
        ) : null}
        {route.name === 'settings' && route.section === 'profile' ? (
          <ProfileSettingsPage model={model} navigate={navigateInternal} />
        ) : null}
        {route.name === 'settings' && route.section === 'api-keys' ? (
          <APIKeysSettingsPage model={model} navigate={navigateInternal} />
        ) : null}
        {route.name === 'skill-settings' && route.section === 'general' ? (
          <SkillGeneralSettingsPage
            model={model}
            namespace={route.namespace}
            navigate={navigateInternal}
            skillName={route.skillName}
          />
        ) : null}
        {route.name === 'skill-settings' && route.section === 'versions' ? (
          <SkillVersionsSettingsPage
            model={model}
            namespace={route.namespace}
            navigate={navigateInternal}
            skillName={route.skillName}
          />
        ) : null}
        {route.name === 'skill-settings' && route.section === 'access' ? (
          <SkillAccessSettingsPage
            model={model}
            namespace={route.namespace}
            navigate={navigateInternal}
            skillName={route.skillName}
          />
        ) : null}
        {route.name === 'skill-settings' && route.section === 'danger' ? (
          <SkillDangerSettingsPage
            model={model}
            namespace={route.namespace}
            navigate={navigateInternal}
            skillName={route.skillName}
          />
        ) : null}
        {route.name === 'install' ? <InstallDocsPage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'auth' ? (
          <AuthRoutePage
            locationSearch={location.search}
            model={model}
            mode={route.mode}
            navigate={navigateInternal}
          />
        ) : null}
      </main>

      <footer className="footer-bar">
        <div>
          <strong>Skill Home</strong>
          <span>统一 skill 的发布、发现和安装入口。</span>
        </div>
        <div className="footer-bar__links">
          <a href={API_BASE} rel="noreferrer" target="_blank">
            Registry API
          </a>
          <button className="footer-link" onClick={() => navigateInternal('/skills')} type="button">
            技能中心
          </button>
          <button className="footer-link" onClick={() => navigateInternal('/publish/new')} type="button">
            发布
          </button>
          <button className="footer-link" onClick={() => navigateInternal('/install')} type="button">
            安装指南
          </button>
        </div>
      </footer>
    </div>
  );
}
