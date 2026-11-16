// SPDX-License-Identifier: AGPL-3.0-only

import { createRootRoute, Outlet } from '@tanstack/react-router';
import { Shell } from '../Shell';

export const Route = createRootRoute({
  component: () => (
    <Shell>
      <Outlet />
    </Shell>
  ),
});
