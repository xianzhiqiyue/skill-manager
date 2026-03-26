import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { SettingsLayout } from './SettingsLayout';

describe('SettingsLayout', () => {
  it('renders a settings nav and isolates danger actions in a dedicated section', () => {
    render(
      <SettingsLayout
        description="Control visibility, releases, and destructive actions for this skill."
        navAriaLabel="Skill settings"
        navItems={[
          {
            current: true,
            href: '/settings/skills/testuser/github/general',
            label: 'General',
          },
          {
            href: '/settings/skills/testuser/github/versions',
            label: 'Versions',
          },
          {
            href: '/settings/skills/testuser/github/access',
            label: 'Access',
          },
          {
            href: '/settings/skills/testuser/github/danger',
            label: 'Danger Zone',
            tone: 'danger',
          },
        ]}
        title="General"
      >
        <section>Form body</section>
      </SettingsLayout>,
    );

    expect(screen.getByRole('navigation', { name: 'Skill settings' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'General' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Danger Zone' })).toHaveClass('is-danger');
    expect(screen.getByText('Form body')).toBeInTheDocument();
  });
});
