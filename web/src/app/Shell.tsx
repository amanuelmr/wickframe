// SPDX-License-Identifier: AGPL-3.0-only

import type { ReactNode } from 'react';

export function Shell({ children }: { children: ReactNode }) {
  return (
    <>
      <header className="border-b px-4 py-3">
        <h1 className="text-lg font-semibold">Wickframe</h1>
      </header>
      <main className="p-4">{children}</main>
    </>
  );
}
