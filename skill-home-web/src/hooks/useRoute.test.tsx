import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';

import { useRoute } from './useRoute';

describe('useRoute', () => {
  beforeEach(() => {
    window.history.replaceState(null, '', '/skills?q=doc');
  });

  it('exposes the current pathname and search string', () => {
    const { result } = renderHook(() => useRoute());

    expect(result.current.route.name).toBe('skills');
    expect(result.current.location.pathname).toBe('/skills');
    expect(result.current.location.search).toBe('?q=doc');
  });

  it('updates pathname and search when navigating with query params', () => {
    const { result } = renderHook(() => useRoute());

    act(() => {
      result.current.navigate('/skills?q=github&sort=updated');
    });

    expect(window.location.pathname).toBe('/skills');
    expect(window.location.search).toBe('?q=github&sort=updated');
    expect(result.current.location.search).toBe('?q=github&sort=updated');
    expect(result.current.route.name).toBe('skills');
  });

  it('keeps deep skill tab paths intact when navigating', () => {
    const { result } = renderHook(() => useRoute());

    act(() => {
      result.current.navigate('/skills/testuser/github/install?view=list');
    });

    expect(window.location.pathname).toBe('/skills/testuser/github/install');
    expect(window.location.search).toBe('?view=list');
    expect(result.current.location.pathname).toBe('/skills/testuser/github/install');
    expect(result.current.location.search).toBe('?view=list');
    expect(result.current.route).toEqual({
      name: 'skill-tab',
      namespace: 'testuser',
      skillName: 'github',
      tab: 'install',
    });
  });

  it('maps bare skill paths to the overview tab', () => {
    const { result } = renderHook(() => useRoute());

    act(() => {
      result.current.navigate('/skills/testuser/github');
    });

    expect(result.current.route).toEqual({
      name: 'skill-tab',
      namespace: 'testuser',
      skillName: 'github',
      tab: 'overview',
    });
  });
});
