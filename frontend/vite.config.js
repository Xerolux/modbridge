import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    minify: 'esbuild',
    target: 'esnext',
    rollupOptions: {
      output: {
        // Prevent underscore-prefixed filenames that break Go's go:embed
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        // Sanitize filenames: replace characters invalid on NTFS / in GitHub Actions artifacts
        sanitizeFileName(name) {
          // Replace \0 and NTFS-invalid characters (:, ", <, >, |, *, ?) with hyphens
          const sanitized = name.replace(/[\0:"<>|*?]/g, '-')
          // Strip leading underscores/hyphens
          return sanitized.replace(/^[-_]+/, '')
        }
      }
    }
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})
