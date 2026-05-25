import { useEffect, useState } from 'react';

import { API_BASE } from './api';
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
import { UserManagementSettingsPage } from './pages/settings/UserManagementSettingsPage';
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

export default function App() {
  const { route, location, navigate } = useRoute();
  const model = useRegistryApp(route, location.search, navigate);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);
  const [toast, setToast] = useState<{ tone: ToastTone; message: string } | null>(null);
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

  function handleHeaderSearchSubmit(query: string, event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMobileNavOpen(false);
    setMobileSearchOpen(false);

    if (route.name === 'skills') {
      model.setCatalogQuery(query);
      return;
    }

    navigate(buildHeaderSearch('/skills', query));
  }

  function handleSkillsNavigation() {
    setMobileNavOpen(false);
    setMobileSearchOpen(false);

    if (route.name === 'skills') {
      return;
    }

    if (route.name === 'skill-tab') {
      model.returnToCatalog();
      return;
    }

    navigate('/skills');
  }

  function buildHeaderSearch(searchPath: string, query: string) {
    return `${searchPath}${toCatalogSearch({
      ...model.catalogFilters,
      query,
    })}`;
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
        catalogQuery={model.catalogFilters.query}
        onSearchSuggestionSelect={(query, namespace, name) => {
          navigateInternal(buildHeaderSearch(buildSkillPath(namespace, name), query));
        }}
        onSearchSubmit={handleHeaderSearchSubmit}
        onSkills={handleSkillsNavigation}
        onToggleMobileNav={() => {
          setMobileNavOpen((value) => !value);
          setMobileSearchOpen(false);
        }}
        onToggleMobileSearch={() => {
          setMobileSearchOpen((value) => !value);
          setMobileNavOpen(false);
        }}
        skills={model.skills}
      />

      <main className="app-main">
        {route.name === 'home' ? <HomeRoutePage model={model} navigate={navigateInternal} /> : null}
        {route.name === 'skills' ? (
          <SkillsSearchPage
            model={{
              ...model,
              skills: model.catalogDisplaySkills ?? model.catalogSkills ?? model.skills,
              skillsError: model.catalogDisplayError ?? model.catalogError ?? model.skillsError,
              skillsLoading: model.catalogDisplayLoading ?? model.catalogLoading ?? model.skillsLoading,
              skillsTotal:
                model.catalogDisplayTotal ??
                model.catalogTotal ??
                model.catalogSkills?.length ??
                model.skills.length,
            }}
          />
        ) : null}
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
        {route.name === 'settings' && route.section === 'users' ? (
          <UserManagementSettingsPage model={model} navigate={navigateInternal} />
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
          <button className="footer-link" onClick={handleSkillsNavigation} type="button">
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
