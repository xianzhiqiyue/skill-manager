import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { GlobalHeader } from './GlobalHeader';

const noop = vi.fn();

function renderHeader({
  catalogQuery = '',
  mobileNavOpen = false,
  onSearchSubmit = noop,
  onSearchSuggestionSelect = noop,
  skills = [],
}: {
  catalogQuery?: string;
  mobileNavOpen?: boolean;
  onSearchSubmit?: (query: string, event: React.FormEvent<HTMLFormElement>) => void;
  onSearchSuggestionSelect?: (query: string, namespace: string, name: string) => void;
  skills?: Array<{
    id: string;
    namespace: string;
    name: string;
    description?: string;
    latestVersion?: string;
  }>;
} = {}) {
  return render(
    <GlobalHeader
      activeNav="home"
      catalogQuery={catalogQuery}
      currentUser={{ username: 'zhuyuxiao314' }}
      mobileNavOpen={mobileNavOpen}
      mobileSearchOpen={false}
      skills={skills}
      onConsole={noop}
      onHome={noop}
      onInstall={noop}
      onLogin={noop}
      onLogout={noop}
      onPublish={noop}
      onRegister={noop}
      onSearchSuggestionSelect={onSearchSuggestionSelect}
      onSearchSubmit={onSearchSubmit}
      onSkills={noop}
      onToggleMobileNav={noop}
      onToggleMobileSearch={noop}
    />,
  );
}

describe('GlobalHeader', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('uses the username chip as the only settings entry in the desktop header', () => {
    renderHeader();

    expect(screen.getByRole('button', { name: 'zhuyuxiao314' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '控制台' })).not.toBeInTheDocument();
    expect(screen.queryByText('控制台')).not.toBeInTheDocument();
  });

  it('keeps the mobile menu free of duplicate console labels', () => {
    renderHeader({ mobileNavOpen: true });

    expect(screen.queryByRole('button', { name: '控制台' })).not.toBeInTheDocument();
    expect(screen.queryByText('查看控制台')).not.toBeInTheDocument();
  });

  it('removes the explicit search button but still submits on Enter', () => {
    const handleSearchSubmit = vi.fn((_: string, event: React.FormEvent<HTMLFormElement>) => event.preventDefault());
    const { container } = renderHeader({ onSearchSubmit: handleSearchSubmit });

    expect(screen.queryByRole('button', { name: '搜索' })).not.toBeInTheDocument();

    const form = container.querySelector('form.gh-header__search');
    expect(form).not.toBeNull();
    fireEvent.submit(form!);
    expect(handleSearchSubmit).toHaveBeenCalledTimes(1);
  });

  it('shows search suggestions and opens the selected skill on click', () => {
    const handleSearchSuggestionSelect = vi.fn();
    renderHeader({
      onSearchSuggestionSelect: handleSearchSuggestionSelect,
      catalogQuery: 'git',
      skills: [
        {
          id: '1',
          namespace: 'testuser',
          name: 'github',
          description: 'Interact with GitHub using gh.',
          latestVersion: '1.0.0',
        },
      ],
    });

    const input = screen.getByRole('searchbox', { name: '搜索 skill、能力、场景' });
    fireEvent.focus(input);

    expect(screen.getByRole('listbox', { name: '搜索联想' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('option', { name: 'testuser / github Interact with GitHub using gh.' }));
    expect(handleSearchSuggestionSelect).toHaveBeenCalledWith('git', 'testuser', 'github');
  });

  it('uses the highlighted suggestion on Enter instead of submitting the search form', () => {
    const handleSearchSubmit = vi.fn((_: string, event: React.FormEvent<HTMLFormElement>) => event.preventDefault());
    const handleSearchSuggestionSelect = vi.fn();
    renderHeader({
      onSearchSubmit: handleSearchSubmit,
      onSearchSuggestionSelect: handleSearchSuggestionSelect,
      catalogQuery: 'g',
      skills: [
        {
          id: '1',
          namespace: 'testuser',
          name: 'github',
          description: 'Interact with GitHub using gh.',
        },
        {
          id: '2',
          namespace: 'testuser',
          name: 'doc',
          description: 'Work with .docx files.',
        },
      ],
    });

    const input = screen.getByRole('searchbox', { name: '搜索 skill、能力、场景' });
    fireEvent.focus(input);
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(handleSearchSuggestionSelect).toHaveBeenCalledWith('g', 'testuser', 'github');
    expect(handleSearchSubmit).not.toHaveBeenCalled();
  });
});
