'use client';

import { useEffect, useRef, useState } from 'react';

export function CopyCommand({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  async function copyCommand() {
    await navigator.clipboard.writeText(command);
    setCopied(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false);
    }, 2200);
  }

  return (
    <button
      className="eve-command"
      type="button"
      onClick={copyCommand}
      aria-label={copied ? 'Command copied' : 'Copy install command'}
      data-copied={copied ? 'true' : 'false'}
    >
      <code>
        <span>$</span> {command}
      </code>
      <span className="eve-copy-icon" aria-hidden="true" />
    </button>
  );
}
