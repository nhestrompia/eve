import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export function baseOptions(): BaseLayoutProps {
  return {
    githubUrl: 'https://github.com/nhestrompia/eve',
    nav: {
      title: 'EVE',
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
