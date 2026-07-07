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
          if (id.includes('@primeuix/themes') || id.includes('primeicons')) return 'vendor-ui-theme';
          if (id.includes('primevue/datatable') || id.includes('primevue/column') || id.includes('primevue/paginator')) return 'vendor-ui-data';
          if (id.includes('primevue/dialog') || id.includes('primevue/confirmdialog') || id.includes('primevue/menu')) return 'vendor-ui-overlay';
          if (id.includes('primevue/inputtext') || id.includes('primevue/inputnumber') || id.includes('primevue/checkbox') || id.includes('primevue/dropdown') || id.includes('primevue/chips') || id.includes('primevue/password') || id.includes('primevue/toggleswitch') || id.includes('primevue/selectbutton')) return 'vendor-ui-form';
          if (id.includes('primevue/tabs') || id.includes('primevue/tablist') || id.includes('primevue/tabpanel') || id.includes('primevue/tabpanels')) return 'vendor-ui-tabs';
          if (id.includes('primevue/button') || id.includes('primevue/tag') || id.includes('primevue/badge') || id.includes('primevue/card') || id.includes('primevue/popover')) return 'vendor-ui-widgets';
          if (id.includes('primevue/toast') || id.includes('primevue/toastservice') || id.includes('primevue/usetoast') || id.includes('primevue/confirmationservice') || id.includes('primevue/useconfirm') || id.includes('primevue/tooltip')) return 'vendor-ui-services';
          if (id.includes('primevue')) return 'vendor-ui-core';
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
