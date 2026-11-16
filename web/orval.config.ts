// SPDX-License-Identifier: AGPL-3.0-only

import { defineConfig } from 'orval';

export default defineConfig({
  wickframe: {
    input: '../api/openapi.yaml',
    output: {
      target: './src/api/generated/wickframe.ts',
      client: 'react-query',
      httpClient: 'fetch',
    },
  },
});
