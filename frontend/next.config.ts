import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /** Output standalone build for Docker production deployment. */
  output: "standalone",
};

export default nextConfig;
