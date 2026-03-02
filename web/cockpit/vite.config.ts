import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [preact(), tailwindcss()],
  server: {
    proxy: {
      '/api/ws': {
        target: process.env.VITE_WS_TARGET ?? 'ws://localhost:8080',
        ws: true,
      },
      '/api': {
        target: process.env.VITE_API_TARGET ?? 'http://localhost:8080',
      },
    },
  },
})
