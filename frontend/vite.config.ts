import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/login': 'http://localhost:8080',
      '/register': 'http://localhost:8080',
      '/logout': 'http://localhost:8080',
      '/upload': 'http://localhost:8080',
      '/photo': 'http://localhost:8080',
      '/static': 'http://localhost:8080',
    },
  },
})
