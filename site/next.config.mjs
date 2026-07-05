import { createMDX } from 'fumadocs-mdx/next';

const basePath = process.env.SITE_BASE_PATH || undefined;

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  trailingSlash: true,
  basePath,
  images: {
    unoptimized: true,
  },
};

const withMDX = createMDX();

export default withMDX(config);
