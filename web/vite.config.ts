import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Em desenvolvimento, o frontend e a API são servidos na mesma origem
// (via proxy do Vite), evitando qualquer complicação de CORS ou de cookies
// entre domínios diferentes — o cookie de sessão HttpOnly funciona de
// forma transparente, como funcionaria em produção com o mesmo domínio.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
