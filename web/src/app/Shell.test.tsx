// SPDX-License-Identifier: AGPL-3.0-only

import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { Shell } from './Shell';

test('Shell shows the app name and renders its content in the main landmark', () => {
  render(
    <Shell>
      <p>content</p>
    </Shell>,
  );

  expect(screen.getByRole('heading', { level: 1, name: 'Wickframe' })).toBeDefined();
  expect(screen.getByRole('main').textContent).toBe('content');
});
