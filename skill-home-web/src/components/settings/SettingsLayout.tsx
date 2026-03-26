import type { ReactNode } from 'react';

import type { SettingsNavItem } from '../../lib/settings';
import { SidebarLayout } from '../layout/SidebarLayout';
import { PageHeader } from '../layout/PageHeader';
import { SettingsNav } from './SettingsNav';

type SettingsLayoutProps = {
  actions?: ReactNode;
  aside?: ReactNode;
  children: ReactNode;
  description?: string;
  eyebrow?: string;
  meta?: ReactNode;
  navAriaLabel: string;
  navItems: SettingsNavItem[];
  onNavigate?: (path: string) => void;
  sidebarHeader?: ReactNode;
  title: string;
};

export function SettingsLayout({
  actions,
  aside,
  children,
  description,
  eyebrow,
  meta,
  navAriaLabel,
  navItems,
  onNavigate,
  sidebarHeader,
  title,
}: SettingsLayoutProps) {
  return (
    <div className="page-stack">
      <section className="surface-panel gh-settings-shell">
        <SidebarLayout
          aside={aside}
          className="gh-sidebar-layout--settings"
          content={(
            <div className="gh-settings-main">
              <PageHeader
                actions={actions}
                description={description}
                eyebrow={eyebrow}
                meta={meta}
                title={title}
              />
              <div className="gh-settings-content">{children}</div>
            </div>
          )}
          sidebar={(
            <div className="gh-settings-sidebar">
              {sidebarHeader ? <div className="gh-settings-sidebar__header">{sidebarHeader}</div> : null}
              <SettingsNav ariaLabel={navAriaLabel} items={navItems} onNavigate={onNavigate} />
            </div>
          )}
        />
      </section>
    </div>
  );
}
