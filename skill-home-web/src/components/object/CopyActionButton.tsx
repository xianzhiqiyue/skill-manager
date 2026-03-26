import { useState } from 'react';

import { copyText } from '../../lib/format';
import { dispatchToast } from '../../lib/toast';

type CopyActionButtonProps = {
  className?: string;
  label?: string;
  value: string;
};

export function CopyActionButton({
  className = 'button button--secondary',
  label = '复制',
  value,
}: CopyActionButtonProps) {
  const [copied, setCopied] = useState(false);

  async function handleClick() {
    try {
      await copyText(value);
      setCopied(true);
      dispatchToast({
        tone: 'success',
        message: label === '复制' ? '内容已复制到剪贴板。' : `${label}已复制。`,
      });
      window.setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
      dispatchToast({
        tone: 'warning',
        message: '复制失败，请手动复制当前内容。',
      });
    }
  }

  return (
    <button className={className} onClick={() => void handleClick()} type="button">
      {copied ? '已复制' : label}
    </button>
  );
}
