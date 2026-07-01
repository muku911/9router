import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/postcss'

// Vite config for the go-9router dashboard.
// - Dev server proxies /api, /v1 and /health to the Go backend (default :20128).
// - Build output is written straight into ../web/dist so //go:embed can pick it up.
export default defineConfig({
  plugins: [react()],
  css: {
    postcss: { plugins: [tailwindcss()] },
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    sourcemap: false,
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks: {
          recharts: ['recharts'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:20128',
      '/v1': 'http://localhost:20128',
      '/health': 'http://localhost:20128',
    },
  },
})
