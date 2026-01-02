import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // Output to the dist folder at the backend root for serving
    outDir: '../dist',
    emptyOutDir: true,
  },
  server: {
    port: 3000,
    // Proxy API calls to Go backend during development
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})

