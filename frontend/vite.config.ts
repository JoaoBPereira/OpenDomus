import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import checker from 'vite-plugin-checker'
import { visualizer } from 'rollup-plugin-visualizer'
import { readFileSync } from 'fs'

const { version } = JSON.parse(readFileSync('./package.json', 'utf-8'))

// https://vitejs.dev/config/
export default defineConfig(({ command }) => ({
  base: './', // Makes all script/CSS references relative to current folder
  envPrefix: ['VITE_', 'BACKEND_'], // expose BACKEND_SERVER_URL to the client
  plugins: [
    vue(),
    checker({ typescript: true }),
    // add visualizer only for build to avoid dev overhead
    ...(command === 'build' ? [visualizer({ filename: './dist/stats.html', open: false })] : [])
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@use "@/styles/mixins" as *;\n`
      }
    }
  },
  define: {
    __APP_VERSION__: JSON.stringify(version),
  },
  server: {
    port: 3000
  },
  build: {
    target: 'es2020',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (!id.includes('node_modules')) return;

          // extract the package name from node_modules path
          const parts = id.split('node_modules/')[1].split('/');
          let pkg = parts[0];
          // scoped packages have the form @scope/name
          if (pkg && pkg.startsWith('@') && parts.length > 1) pkg = `${pkg}/${parts[1]}`;

          // vue runtime packages
          if (pkg === 'vue' || pkg === 'vue-router') return 'vue-vendor';
          if (pkg === 'echarts' || pkg === 'vue-echarts') return 'charts-vendor';

          // All other packages go into `vendor` to avoid cross-import splitting
          return 'vendor';
        }
      }
    }
  }
}))
