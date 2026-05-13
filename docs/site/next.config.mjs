import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  output: 'export',
  basePath: process.env.BASE_PATH ?? '/go-udap',
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
};

export default withMDX(config);
