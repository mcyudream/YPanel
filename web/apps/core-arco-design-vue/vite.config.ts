import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import dayjs from 'dayjs'
import { defineConfig, loadEnv } from 'vite'
import { parseLoadedEnv } from 'vite-plugin-env-parse'
import pkg from './package.json' with { type: 'json' }
import createVitePlugins from './vite/plugins.ts'

export default defineConfig(({ mode, command }) => {
  const env = parseLoadedEnv(loadEnv(mode, process.cwd()))
  // 全局 scss 资源
  const scssResources: string[] = []
  fs.readdirSync('src/assets/styles/resources').forEach((dirname) => {
    if (fs.statSync(`src/assets/styles/resources/${dirname}`).isFile()) {
      scssResources.push(`@use "/src/assets/styles/resources/${dirname}" as *;`)
    }
  })
  return {
    // 开发服务器选项 https://cn.vitejs.dev/config/server-options
    server: {
      open: true,
      host: true,
      port: 9000,
      proxy: {
        '/proxy': {
          target: env.VITE_APP_API_BASEURL,
          changeOrigin: command === 'serve' && env.VITE_ENABLE_PROXY,
          rewrite: path => path.replace(/\/proxy/, ''),
        },
      },
    },
    // 构建选项 https://cn.vitejs.dev/config/build-options
    build: {
      outDir: mode === 'production' ? 'dist' : `dist-${mode}`,
      sourcemap: env.VITE_BUILD_SOURCEMAP,
      rollupOptions: {
        output: {
          // F12：第三方大件分包，避免单 vendor 巨块
          manualChunks(id: string) {
            if (!id.includes('node_modules')) {
              return undefined
            }
            if (id.includes('echarts') || id.includes('zrender')) {
              return 'vendor-echarts'
            }
            if (id.includes('xterm')) {
              return 'vendor-xterm'
            }
            if (id.includes('monaco-editor')) {
              return 'vendor-monaco'
            }
            if (id.includes('arco-design')) {
              return 'vendor-arco'
            }
            if (id.includes('vue') || id.includes('pinia') || id.includes('vue-router')) {
              return 'vendor-vue'
            }
            return 'vendor'
          },
        },
      },
    },
    define: {
      __SYSTEM_INFO__: JSON.stringify({
        pkg: {
          dependencies: pkg.dependencies,
          devDependencies: pkg.devDependencies,
        },
        lastBuildTime: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      }),
    },
    plugins: createVitePlugins(mode, command === 'build'),
    optimizeDeps: {
      exclude: [
        '@fantastic-admin/components',
        '@fantastic-admin/composables',
      ],
    },
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, 'src'),
        '#': path.resolve(import.meta.dirname, 'src/types'),
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: scssResources.join(''),
        },
      },
    },
    rollupOptions: {
      output: {
        // F12：第三方大件分包，避免单 vendor 巨块
        manualChunks(id: string) {
          if (!id.includes('node_modules')) {
            return undefined
          }
          if (id.includes('echarts') || id.includes('zrender')) {
            return 'vendor-echarts'
          }
          if (id.includes('xterm')) {
            return 'vendor-xterm'
          }
          if (id.includes('monaco-editor')) {
            return 'vendor-monaco'
          }
          if (id.includes('arco-design')) {
            return 'vendor-arco'
          }
          if (id.includes('vue') || id.includes('pinia') || id.includes('vue-router')) {
            return 'vendor-vue'
          }
          return 'vendor'
        },
      },
    },
  }
})
