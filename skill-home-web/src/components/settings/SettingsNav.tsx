import type { SettingsNavItem } from '../../lib/settings';

type SettingsNavProps = {
  ariaLabel: string;
  items: SettingsNavItem[];
  onNavigate?: (path: string) => void;
};

export function SettingsNav({ ariaLabel, items, onNavigate }: SettingsNavProps) {
  return (
    <nav aria-label={ariaLabel} className="gh-settings-nav">
      <ul className="gh-settings-nav__list">
        {items.map((item) => (
          <li className="gh-settings-nav__item" key={item.href}>
            <a
              aria-current={item.current ? 'page' : undefined}
              className={`gh-settings-nav__link ${item.current ? 'is-active' : ''} ${
                item.tone === 'danger' ? 'is-danger' : ''
              }`.trim()}
              href={item.href}
              onClick={(event) => {
                if (!onNavigate) {
                  return;
                }

                event.preventDefault();
                onNavigate(item.href);
              }}
            >
              {item.label}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
