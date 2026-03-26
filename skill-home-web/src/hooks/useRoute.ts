import { useEffect, useState } from 'react';

import { parseRoute } from '../lib/routes';

type NavigateOptions = {
  replace?: boolean;
};

function getPathname() {
  if (typeof window === 'undefined') {
    return '/';
  }
  return window.location.pathname || '/';
}

export function useRoute() {
  const [route, setRoute] = useState(() => parseRoute(getPathname()));

  useEffect(() => {
    function handlePopState() {
      setRoute(parseRoute(getPathname()));
    }

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  function navigate(path: string, options: NavigateOptions = {}) {
    if (typeof window === 'undefined') {
      return;
    }

    const nextPath = path || '/';
    const method = options.replace ? 'replaceState' : 'pushState';

    if (window.location.pathname !== nextPath) {
      window.history[method](null, '', nextPath);
      window.scrollTo({ top: 0, behavior: 'auto' });
    }

    setRoute(parseRoute(nextPath));
  }

  return { route, navigate };
}
