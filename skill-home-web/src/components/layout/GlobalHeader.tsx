import { useEffect, useId, useState } from 'react';

import { getSkillDescription, type SkillSummary } from '../../api';

type HeaderNav = 'home' | 'skills' | 'install' | 'publish' | 'settings' | null;

type HeaderUser = {
  username: string;
} | null;

export type HeaderSearchSuggestion = {
  id: string;
  namespace: string;
  name: string;
  description?: string;
  latestVersion?: string;
};

type GlobalHeaderProps = {
  activeNav: HeaderNav;
  catalogQuery: string;
  currentUser: HeaderUser;
  mobileNavOpen: boolean;
  mobileSearchOpen: boolean;
  skills: SkillSummary[];
  onConsole: () => void;
  onHome: () => void;
  onInstall: () => void;
  onLogin: () => void;
  onLogout: () => void;
  onPublish: () => void;
  onRegister: () => void;
  onSearchSuggestionSelect: (query: string, namespace: string, name: string) => void;
  onSearchSubmit: (query: string, event: React.FormEvent<HTMLFormElement>) => void;
  onSkills: () => void;
  onToggleMobileNav: () => void;
  onToggleMobileSearch: () => void;
};

function NavButton({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-current={active ? 'page' : undefined}
      className={`gh-nav-button ${active ? 'is-active' : ''}`.trim()}
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}

function buildHeaderSearchSuggestions(skills: SkillSummary[], query: string): HeaderSearchSuggestion[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return [];
  }

  return skills
    .map((skill) => {
      const reference = `${skill.namespace}/${skill.name}`.toLowerCase();
      const name = skill.name.toLowerCase();
      const description = [skill.description, skill.description_zh, getSkillDescription(skill)]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
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
      description: getSkillDescription(skill),
      latestVersion: skill.latest_version,
    }));
}

export function GlobalHeader({
  activeNav,
  catalogQuery,
  currentUser,
  mobileNavOpen,
  mobileSearchOpen,
  skills,
  onConsole,
  onHome,
  onInstall,
  onLogin,
  onLogout,
  onPublish,
  onRegister,
  onSearchSuggestionSelect,
  onSearchSubmit,
  onSkills,
  onToggleMobileNav,
  onToggleMobileSearch,
}: GlobalHeaderProps) {
  const [searchValue, setSearchValue] = useState(catalogQuery);
  const [searchFocused, setSearchFocused] = useState(false);
  const [activeSuggestionIndex, setActiveSuggestionIndex] = useState(-1);
  const suggestionListId = useId();
  const searchSuggestions = buildHeaderSearchSuggestions(skills, searchValue);
  const suggestionsVisible = searchFocused && searchValue.trim().length > 0 && searchSuggestions.length > 0;

  useEffect(() => {
    setSearchValue(catalogQuery);
  }, [catalogQuery]);

  useEffect(() => {
    setActiveSuggestionIndex(-1);
    if (!searchValue.trim()) {
      setSearchFocused(false);
    }
  }, [searchValue]);

  function selectSuggestion(index: number) {
    const suggestion = searchSuggestions[index];
    if (!suggestion) {
      return;
    }

    setSearchFocused(false);
    setActiveSuggestionIndex(-1);
    onSearchSuggestionSelect(searchValue, suggestion.namespace, suggestion.name);
  }

  return (
    <header className="gh-header">
      <div className="gh-header__inner">
        <div className="gh-header__brand">
          <button aria-label="返回首页" className="gh-header__mark" onClick={onHome} type="button">
            <span>SH</span>
          </button>
          <div className="gh-header__brand-copy">
            <strong>Skill Home</strong>
            <span>Registry</span>
          </div>
        </div>

        <nav className="gh-header__nav" aria-label="Primary">
          <NavButton active={activeNav === 'home'} label="首页" onClick={onHome} />
          <NavButton active={activeNav === 'skills'} label="技能中心" onClick={onSkills} />
          <NavButton active={activeNav === 'install'} label="安装指南" onClick={onInstall} />
          <NavButton active={activeNav === 'publish'} label="发布" onClick={onPublish} />
        </nav>

        <form
          className={`gh-header__search ${mobileSearchOpen ? 'is-open' : ''}`.trim()}
          onBlur={(event) => {
            const nextTarget = event.relatedTarget;
            if (!(nextTarget instanceof Node) || !event.currentTarget.contains(nextTarget)) {
              setSearchFocused(false);
              setActiveSuggestionIndex(-1);
            }
          }}
          onSubmit={(event) => onSearchSubmit(searchValue, event)}
        >
          <input
            aria-activedescendant={
              suggestionsVisible && activeSuggestionIndex >= 0
                ? `${suggestionListId}-option-${activeSuggestionIndex}`
                : undefined
            }
            aria-label="搜索 skill、能力、场景"
            aria-autocomplete="list"
            aria-controls={suggestionsVisible ? suggestionListId : undefined}
            aria-expanded={suggestionsVisible}
            autoComplete="off"
            onChange={(event) => {
              const nextValue = event.target.value;
              setSearchValue(nextValue);
              setSearchFocused(Boolean(nextValue.trim()));
            }}
            onFocus={() => {
              if (searchValue.trim()) {
                setSearchFocused(true);
              }
            }}
            onKeyDown={(event) => {
              if (!searchSuggestions.length || !searchValue.trim()) {
                return;
              }

              if (event.key === 'ArrowDown') {
                event.preventDefault();
                setSearchFocused(true);
                setActiveSuggestionIndex((current) =>
                  current < searchSuggestions.length - 1 ? current + 1 : 0,
                );
                return;
              }

              if (event.key === 'ArrowUp') {
                event.preventDefault();
                setSearchFocused(true);
                setActiveSuggestionIndex((current) =>
                  current > 0 ? current - 1 : searchSuggestions.length - 1,
                );
                return;
              }

              if (event.key === 'Escape') {
                setSearchFocused(false);
                setActiveSuggestionIndex(-1);
                return;
              }

              if (event.key === 'Enter' && suggestionsVisible && activeSuggestionIndex >= 0) {
                event.preventDefault();
                selectSuggestion(activeSuggestionIndex);
              }
            }}
            placeholder="搜索 skill、能力、场景"
            type="search"
            value={searchValue}
          />
          {suggestionsVisible ? (
            <div
              aria-label="搜索联想"
              className="gh-header__search-dropdown"
              id={suggestionListId}
              role="listbox"
            >
              {searchSuggestions.map((suggestion, index) => (
                <button
                  aria-selected={index === activeSuggestionIndex}
                  className={`gh-header__search-suggestion ${index === activeSuggestionIndex ? 'is-active' : ''}`.trim()}
                  id={`${suggestionListId}-option-${index}`}
                  key={suggestion.id}
                  onClick={() => selectSuggestion(index)}
                  onMouseDown={(event) => event.preventDefault()}
                  role="option"
                  type="button"
                >
                  <strong>{suggestion.namespace} / {suggestion.name}</strong>
                  {suggestion.description ? <span>{suggestion.description}</span> : null}
                  {suggestion.latestVersion ? (
                    <span aria-hidden="true" className="gh-header__search-suggestion-meta">
                      v{suggestion.latestVersion}
                    </span>
                  ) : null}
                </button>
              ))}
            </div>
          ) : null}
        </form>

        <div className="gh-header__account">
          {currentUser ? (
            <>
              <button
                className={`gh-account-chip ${activeNav === 'settings' ? 'is-active' : ''}`.trim()}
                onClick={onConsole}
                type="button"
              >
                <strong>{currentUser.username}</strong>
              </button>
              <button className="button button--quiet" onClick={onLogout} type="button">
                退出
              </button>
            </>
          ) : (
            <>
              <button className="button button--quiet" onClick={onLogin} type="button">
                登录
              </button>
              <button className="button button--secondary" onClick={onRegister} type="button">
                注册
              </button>
            </>
          )}
        </div>

        <div className="gh-header__mobile-actions">
          <button
            aria-label="切换搜索"
            className="gh-mobile-toggle"
            onClick={onToggleMobileSearch}
            type="button"
          >
            {mobileSearchOpen ? '收起搜索' : '搜索'}
          </button>
          <button
            aria-label="切换导航"
            className="gh-mobile-toggle"
            onClick={onToggleMobileNav}
            type="button"
          >
            {mobileNavOpen ? '关闭' : '菜单'}
          </button>
        </div>
      </div>

      {mobileNavOpen ? (
        <div className="gh-mobile-nav">
          <NavButton active={activeNav === 'home'} label="首页" onClick={onHome} />
          <NavButton active={activeNav === 'skills'} label="技能中心" onClick={onSkills} />
          <NavButton active={activeNav === 'install'} label="安装指南" onClick={onInstall} />
          <NavButton active={activeNav === 'publish'} label="发布" onClick={onPublish} />
          <div className="gh-mobile-nav__account">
            {currentUser ? (
              <>
                <button
                  className={`gh-account-chip ${activeNav === 'settings' ? 'is-active' : ''}`.trim()}
                  onClick={onConsole}
                  type="button"
                >
                  <strong>{currentUser.username}</strong>
                </button>
                <button className="button button--quiet" onClick={onLogout} type="button">
                  退出登录
                </button>
              </>
            ) : (
              <>
                <button className="button button--quiet" onClick={onLogin} type="button">
                  登录
                </button>
                <button className="button button--secondary" onClick={onRegister} type="button">
                  注册
                </button>
              </>
            )}
          </div>
        </div>
      ) : null}
    </header>
  );
}
