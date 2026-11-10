import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        cookieDomainRewrite: { '127.0.0.1': 'localhost' },
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const cookie = req.headers.cookie;
            if (cookie) {
              proxyReq.setHeader('Cookie', cookie);
            }
          });
          proxy.on('proxyRes', (proxyRes, _req, res) => {
            const setCookie = proxyRes.headers['set-cookie'];
            if (setCookie) {
              const rewritten = Array.isArray(setCookie)
                ? setCookie.map((c: string) => c.replace(/Domain=127\.0\.0\.1/gi, 'Domain=localhost'))
                : (setCookie as string).replace(/Domain=127\.0\.0\.1/gi, 'Domain=localhost');
              res.setHeader('Set-Cookie', rewritten);
            }
          });
        },
      },
    },
  },
})