import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { FilterRail } from './FilterRail';

const baseProps = {
  activeFilterLabels: [],
  filters: {
    query: 'fmea',
    namespace: 'all',
    tag: 'all',
    license: 'all',
    sort: 'downloads' as const,
    view: 'list' as const,
  },
  licenseOptions: ['MIT'],
  namespaceOptions: ['testuser'],
  onChange: vi.fn(),
  onQueryChange: vi.fn(),
  onReset: vi.fn(),
  skills: [
    {
      id: '1',
      namespace: 'testuser',
      name: 'openclaw-fmea-cocreator',
      description: 'Helps draft FMEA packages.',
      tags: ['analysis', 'docs'],
      license: 'MIT',
      download_count: 3,
      rating_count: 0,
      latest_version: '0.2.0',
      updated_at: '2026-04-02T00:00:00Z',
    },
  ],
  tagOptions: ['analysis', 'docs'],
};

describe('FilterRail keyword draft', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
  });

  it('debounces catalog query commits while the user is typing', () => {
    const onQueryChange = vi.fn();

    render(<FilterRail {...baseProps} onQueryChange={onQueryChange} />);

    fireEvent.change(screen.getByRole('textbox', { name: '关键词' }), {
      target: { value: 'fmea system' },
    });

    expect(onQueryChange).not.toHaveBeenCalled();

    vi.advanceTimersByTime(249);
    expect(onQueryChange).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(onQueryChange).toHaveBeenCalledWith('fmea system');
  });
});
