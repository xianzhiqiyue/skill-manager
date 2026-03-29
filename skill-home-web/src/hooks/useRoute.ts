import { useEffect, useState } from 'react';

import { APP_BASE_PATH, applyBasePath, stripBasePath } from '../basePath';
import { parseRoute } from '../lib/routes';

type NavigateOptions = {
  replace?: boolean;
};

function getLocationState() {
  if (typeof window === 'undefined') {
    return {
      pathname: '/',
      search: '',
    };
  }

  return {
    pathname: stripBasePath(window.location.pathname || '/', APP_BASE_PATH),
    search: window.location.search || '',
  };
}

function parseNextLocation(path: string) {
  const target = path || '/';
  const url = new URL(target, typeof window === 'undefined' ? 'http://localhost' : window.location.origin);

  return {
    pathname: stripBasePath(url.pathname || '/', APP_BASE_PATH),
    search: url.search || '',
  };
}

export function useRoute() {
  const [location, setLocation] = useState(getLocationState);
  const [route, setRoute] = useState(() => parseRoute(getLocationState().pathname));

  function syncRoute(nextLocation: ReturnType<typeof getLocationState>) {
    setLocation(nextLocation);
    setRoute(parseRoute(nextLocation.pathname));
  }

  useEffect(() => {
    function handlePopState() {
      syncRoute(getLocationState());
    }

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  function navigate(path: string, options: NavigateOptions = {}) {
    if (typeof window === 'undefined') {
      return;
    }

    const nextLocation = parseNextLocation(path);
    const nextPath = `${applyBasePath(nextLocation.pathname, APP_BASE_PATH)}${nextLocation.search}`;
    const method = options.replace ? 'replaceState' : 'pushState';
    const currentPath = `${window.location.pathname || '/'}${window.location.search || ''}`;

    if (currentPath !== nextPath) {
      window.history[method](null, '', nextPath);
      if (typeof window.scrollTo === 'function') {
        window.scrollTo({ top: 0, behavior: 'auto' });
      }
    }

    syncRoute(nextLocation);
  }

  return { route, location, navigate };
}
