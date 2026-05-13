import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    minify: 'esbuild',
    target: 'es2020',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return;
          if (id.includes('gridstack')) return 'vendor-grid';
          if (id.includes('primevue') || id.includes('primeicons') || id.includes('@primeuix')) return 'vendor-ui';
          if (id.includes('/vue/') || id.includes('vue-router') || id.includes('pinia')) return 'vendor-vue';
          if (id.includes('vue-i18n') || id.includes('vue-draggable-plus')) return 'vendor-utils';
        },
        // Prevent underscore-prefixed filenames that break Go's go:embed
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        // Strip leading underscores from sanitized filenames
        sanitizeFileName(name) {
          // Replace null bytes and colons (colons are invalid on NTFS and rejected by upload-artifact)
          const sanitized = name.replace(/\0/g, '_').replace(/:/g, '_')
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
