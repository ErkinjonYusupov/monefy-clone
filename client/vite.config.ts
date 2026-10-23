import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    // Telegram'da sinash uchun tunnel (ngrok, cloudflared) orqali ochiladi
    allowedHosts: true,
    proxy: {
      '/api': 'http://localhost:8090',
    },
  },
})
