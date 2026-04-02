import { describe, expect, it } from 'vitest';

import {
  applyBasePath,
  normalizeBasePath,
  resolveAPIBase,
  stripBasePath,
} from './basePath';

describe('basePath helpers', () => {
  it('normalizes empty and root-like paths', () => {
    expect(normalizeBasePath('')).toBe('');
    expect(normalizeBasePath('/')).toBe('');
    expect(normalizeBasePath(' /skill-home/ ')).toBe('/skill-home');
    expect(normalizeBasePath('skill-home')).toBe('/skill-home');
  });

  it('applies and strips the app base path consistently', () => {
    expect(applyBasePath('/skills', '/skill-home')).toBe('/skill-home/skills');
    expect(applyBasePath('/', '/skill-home')).toBe('/skill-home/');
    expect(stripBasePath('/skill-home/skills', '/skill-home')).toBe('/skills');
    expect(stripBasePath('/skill-home', '/skill-home')).toBe('/');
  });

  it('resolves the public API base from the current origin and app base path', () => {
    expect(
      resolveAPIBase(
        { hostname: 'soulstore.ciqtek.com', origin: 'https://soulstore.ciqtek.com' },
        '',
        '/skill-home/',
      ),
    ).toBe('https://soulstore.ciqtek.com/skill-home');
  });

  it('prefers an explicit registry base URL override', () => {
    expect(
      resolveAPIBase(
        { hostname: 'soulstore.ciqtek.com', origin: 'https://soulstore.ciqtek.com' },
        'https://registry.example.test/skill-home/',
        '/skill-home/',
      ),
    ).toBe('https://registry.example.test/skill-home');
  });

  it('falls back to the local registry when running from localhost', () => {
    expect(
      resolveAPIBase(
        { hostname: '127.0.0.1', origin: 'http://127.0.0.1:5173' },
        '',
        '/skill-home/',
      ),
    ).toBe('http://127.0.0.1:8080/skill-home');
  });
});
