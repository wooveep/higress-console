import path from 'node:path';
import { webcrypto } from 'node:crypto';
import { fileURLToPath, URL } from 'node:url';
import vue from '@vitejs/plugin-vue';
import Components from 'unplugin-vue-components/vite';
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers';
import { defineConfig } from 'vite';

const rootDir = fileURLToPath(new URL('.', import.meta.url));
const devServerHost = process.env.AIGATEWAY_CONSOLE_DEV_HOST || '0.0.0.0';
const devServerPort = Number(process.env.AIGATEWAY_CONSOLE_FRONTEND_PORT || 3000);
const devApiTarget = process.env.AIGATEWAY_CONSOLE_DEV_API_TARGET || 'http://127.0.0.1:8080';

// Vite 6 reads global Web Crypto during config resolution. The Maven-installed
// Node 16 runtime may expose a partial `globalThis.crypto` without
// `getRandomValues`, while `node:crypto.webcrypto` is complete. Patch both the
// missing-global and missing-method cases for build compatibility.
if (!globalThis.crypto || typeof globalThis.crypto.getRandomValues !== 'function') {
  (globalThis as typeof globalThis & { crypto: Crypto }).crypto = webcrypto as Crypto;
}

export default defineConfig({
  plugins: [
    vue(),
    Components({
      dts: path.resolve(rootDir, 'src/components.d.ts'),
      resolvers: [AntDesignVueResolver({ importStyle: false })],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: devServerHost,
    port: devServerPort,
    proxy: {
      '/api': {
        target: devApiTarget,
        changeOrigin: true,
        rewrite: (requestPath) => requestPath.replace(/^\/api/, ''),
      },
    },
  },
  build: {
    outDir: 'build',
    sourcemap: false,
  },
});
