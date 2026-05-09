/** @type {import('next').NextConfig} */
const nextConfig = {
  // Security: Disable powered-by header
  poweredByHeader: false,

  // Security: Explicitly disable browser source maps in production
  productionBrowserSourceMaps: false,

  // Remove deprecated experimental.appDir (default in Next.js 14+)

  async rewrites() {
    // 使用相对路径进行重写，由部署环境决定目标
    // 当使用 Docker Compose 或 Kubernetes 时，
    // 可以通过外部反向代理或 Next.js 的 API 路由来处理
    return [
      {
        source: '/api/v1/:path*',
        destination: '/api/v1/:path*',
      },
    ];
  },

  images: {
    domains: ['localhost'],
  },

  // Security: Add security response headers
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'X-XSS-Protection', value: '1; mode=block' },
          { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
          {
            key: 'Strict-Transport-Security',
            value: 'max-age=63072000; includeSubDomains; preload',
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
