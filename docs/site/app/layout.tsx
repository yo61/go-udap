import { Inter } from 'next/font/google';
import type { Metadata } from 'next';
import { Provider } from '@/components/provider';
import './global.css';

const inter = Inter({
  subsets: ['latin'],
});

// metadataBase resolves relative URLs in metadata fields like
// openGraph.images. Without it, Next.js defaults to http://localhost:3000
// at build time and bakes that into <meta> tags shipped to production —
// social-card scrapers then try to fetch from localhost and fail.
// NEXT_PUBLIC_SITE_URL is set in next.config.mjs.
export const metadata: Metadata = {
  metadataBase: new URL(
    process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000',
  ),
};

export default function Layout({ children }: LayoutProps<'/'>) {
  return (
    <html lang="en" className={inter.className} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <Provider>{children}</Provider>
      </body>
    </html>
  );
}
