import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { SkillOverviewPage } from './SkillOverviewPage';

afterEach(() => {
  cleanup();
});

describe('SkillOverviewPage', () => {
  it('renders the object header, object tabs, and metadata sidebar', () => {
    const navigate = vi.fn();

    render(
      <SkillOverviewPage
        model={{
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            license: 'MIT',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: false,
            tags: ['automation', 'github'],
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'passed',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
        }}
        navigate={navigate}
        search="?q=github&tag=automation"
      />,
    );

    expect(screen.getByRole('heading', { name: 'testuser / github' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Versions' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Install' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Activity' })).toBeInTheDocument();
    const metadata = screen.getByRole('region', { name: 'Metadata' });
    expect(metadata).toHaveTextContent('Metadata');
    expect(metadata).toHaveTextContent('Namespace');
    expect(metadata).toHaveTextContent('testuser');
    expect(metadata).toHaveTextContent('Latest version');
    expect(metadata).toHaveTextContent('1.0.0');

    fireEvent.click(screen.getByRole('link', { name: 'Versions' }));

    expect(navigate).toHaveBeenCalledWith('/skills/testuser/github/versions?q=github&tag=automation');
  });

  it('keeps scan status separate from the deprecated lifecycle flag in metadata', () => {
    render(
      <SkillOverviewPage
        model={{
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            license: 'MIT',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: true,
            tags: ['automation', 'github'],
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'pass',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
        }}
        navigate={vi.fn()}
      />
    );

    const metadata = screen.getByRole('region', { name: 'Metadata' });

    expect(metadata).toHaveTextContent('State');
    expect(metadata).toHaveTextContent('扫描通过');
    expect(metadata).not.toHaveTextContent('Deprecated');
  });

  it('routes official tags back into a catalog tag filter', () => {
    const navigate = vi.fn();

    render(
      <SkillOverviewPage
        model={{
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            license: 'MIT',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: false,
            tags: ['automation', 'github'],
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'passed',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
        }}
        navigate={navigate}
      />,
    );

    fireEvent.click(screen.getAllByRole('button', { name: 'automation' })[0]);

    expect(navigate).toHaveBeenCalledWith('/skills?tag=automation');
  });

  it('renders community tags and lets authenticated viewers submit or remove their own tags', () => {
    const submitCommunityTag = vi.fn();
    const removeCommunityTag = vi.fn();

    render(
      <SkillOverviewPage
        model={{
          currentUser: {
            id: 'user-1',
            username: 'testuser',
            email: 'test@example.com',
          },
          communityTagError: null,
          communityTagRemoving: null,
          communityTagSaving: false,
          communityTagSuccess: null,
          detailError: null,
          detailLoading: false,
          detailSkill: {
            id: 'skill-1',
            namespace: 'testuser',
            name: 'github',
            description: 'Interact with GitHub using gh.',
            license: 'MIT',
            download_count: 18,
            rating_count: 0,
            latest_version: '1.0.0',
            updated_at: '2026-03-22T21:32:00Z',
            is_public: true,
            is_deprecated: false,
            tags: ['automation', 'github'],
            community_tags: [{ tag: 'deployment', count: 2 }],
            viewer_tags: ['deployment'],
            versions: [
              {
                id: 'v1',
                version: '1.0.0',
                size_bytes: 4096,
                scan_status: 'passed',
                created_at: '2026-03-22T21:32:00Z',
              },
            ],
            owner: {
              username: 'testuser',
            },
          },
          removeCommunityTag,
          submitCommunityTag,
        }}
        navigate={vi.fn()}
      />,
    );

    expect(screen.getByRole('heading', { name: 'Community tags' })).toBeInTheDocument();
    expect(screen.getAllByText('deployment')).toHaveLength(2);

    const input = screen.getByRole('textbox', { name: 'Add tag' });
    fireEvent.change(input, { target: { value: 'ci' } });
    fireEvent.submit(input.closest('form')!);

    expect(submitCommunityTag).toHaveBeenCalledWith('ci');

    fireEvent.click(screen.getByRole('button', { name: 'deployment' }));

    expect(removeCommunityTag).toHaveBeenCalledWith('deployment');
  });
});
