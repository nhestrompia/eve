import Image from 'next/image';
import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export function baseOptions(): BaseLayoutProps {
  return {
    githubUrl: 'https://github.com/nhestrompia/eve',
    nav: {
      title: (
        <span className="eve-docs-brand" aria-label="eve">
          <Image src="/eve.svg" alt="" width={88} height={36} unoptimized priority />
        </span>
      ),
    },
    links: [
      {
        text: 'Docs',
        url: '/docs',
        active: 'nested-url',
      },
      {
        text: 'GitHub',
        url: 'https://github.com/nhestrompia/eve',
        external: true,
      },
    ],
  };
}
