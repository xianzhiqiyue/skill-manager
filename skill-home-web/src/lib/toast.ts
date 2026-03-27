export type ToastTone = 'success' | 'danger' | 'warning' | 'neutral';

export function dispatchToast(detail: { tone: ToastTone; message: string }) {
  if (typeof window === 'undefined') {
    return;
  }

  window.dispatchEvent(new CustomEvent('skill-home-toast', { detail }));
}
