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
        // Strip leading underscores from sanitized filenames
        sanitizeFileName(name) {
          // Default Rollup sanitization replaces \0 with '_', we strip leading underscores
          const sanitized = name.replace(/\0/g, '_')
          return sanitized.replace(/^_+/, '')
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
