import type { Metadata, Viewport } from 'next';
import { RootProvider } from 'fumadocs-ui/provider/next';
import 'fumadocs-ui/style.css';
import './global.css';

export const metadata: Metadata = {
  title: {
    default: 'eve - git tracks code, eve tracks product',
    template: '%s | eve',
  },
  description:
    'eve records completed product changes next to Git history so developers and agents can understand what changed, why, and how it was verified.',
  icons: {
    icon: '/eve.svg',
  },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
