// SPDX-License-Identifier: AGPL-3.0-only

import tailwindcss from '@tailwindcss/vite';
import { tanstackRouter } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: 'react',
      routesDirectory: './src/app/routes',
      generatedRouteTree: './src/app/routeTree.gen.ts',
      routeTreeFileHeader: [
        '// SPDX-License-Identifier: AGPL-3.0-only',
        '/* eslint-disable */',
        '// @ts-nocheck',
        '// noinspection JSUnusedGlobalSymbols',
      ],
    }),
    react(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/api': { target: 'http://127.0.0.1:7317', ws: true },
    },
  },
  test: {
    environment: 'jsdom',
  },
});
