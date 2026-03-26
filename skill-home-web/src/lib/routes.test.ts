import { describe, expect, it } from 'vitest';

import { parseRoute } from './routes';

describe('GitHub-style routes', () => {
  it('parses bare skill paths as the overview tab', () => {
    expect(parseRoute('/skills/testuser/github')).toEqual({
      name: 'skill-tab',
      namespace: 'testuser',
      skillName: 'github',
      tab: 'overview',
    });
  });

  it('parses skill object tabs', () => {
    expect(parseRoute('/skills/testuser/github/install')).toEqual({
      name: 'skill-tab',
      namespace: 'testuser',
      skillName: 'github',
      tab: 'install',
    });
  });

  it('parses settings routes', () => {
    expect(parseRoute('/settings/skills/testuser/github/danger')).toEqual({
      name: 'skill-settings',
      namespace: 'testuser',
      skillName: 'github',
      section: 'danger',
    });
  });

  it('collapses legacy publish and console urls into canonical settings routes', () => {
    expect(parseRoute('/publish')).toEqual({
      name: 'publish-new',
    });
    expect(parseRoute('/publish/new')).toEqual({
      name: 'publish-new',
    });
    expect(parseRoute('/console')).toEqual({
      name: 'settings',
      section: 'profile',
    });
    expect(parseRoute('/console/skills')).toEqual({
      name: 'settings',
      section: 'profile',
    });
    expect(parseRoute('/console/api-keys')).toEqual({
      name: 'settings',
      section: 'api-keys',
    });
  });
});
