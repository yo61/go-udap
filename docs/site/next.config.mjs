import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

const basePath = process.env.BASE_PATH ?? '/go-udap';

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  output: 'export',
  basePath,
  // Expose basePath to client bundles so the static search client can
  // prefix its fetch URL (`<basePath>/api/search`). Without this the
  // default `/api/search` 404s on the basePath-mounted site, the search
  // dialog opens but never returns results, and Enter does nothing.
  env: {
    NEXT_PUBLIC_BASE_PATH: basePath,
  },
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
};

export default withMDX(config);
