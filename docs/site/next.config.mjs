import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

const basePath = process.env.BASE_PATH ?? '/go-udap';
const siteUrl = process.env.SITE_URL ?? 'https://yo61.github.io';

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  output: 'export',
  basePath,
  // Expose basePath + siteUrl to client AND server bundles. Used by:
  //   - components/search.tsx: prefix the static-search fetch URL
  //     (`<basePath>/api/search`) so it doesn't 404 on a basePath-
  //     mounted site.
  //   - lib/shared.ts: build basePath-prefixed URLs for the OG image
  //     and llms.mdx routes so meta tags resolve correctly.
  //   - app/layout.tsx: set metadataBase so relative URLs in
  //     openGraph.images become absolute production URLs in the
  //     emitted <meta property="og:image"> tag.
  env: {
    NEXT_PUBLIC_BASE_PATH: basePath,
    NEXT_PUBLIC_SITE_URL: siteUrl,
  },
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
};

export default withMDX(config);
