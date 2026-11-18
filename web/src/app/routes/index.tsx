// SPDX-License-Identifier: AGPL-3.0-only

import { createFileRoute } from '@tanstack/react-router';
import { LiveChart } from '../../features/chart/LiveChart';

export const Route = createFileRoute('/')({
  component: () => <LiveChart instrument="fake:DEMO" tf="1m" />,
});
