import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwind from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwind()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://127.0.0.1:8787', changeOrigin: true, ws: true }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('@xterm')) return 'xterm'
          if (id.includes('@ant-design/x')) return 'antd-x'
          if (id.includes('node_modules')) return 'vendor'
        },
      },
    },
  },
})
