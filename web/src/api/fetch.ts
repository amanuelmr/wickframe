// SPDX-License-Identifier: AGPL-3.0-only

import type { Problem } from './generated/wickframe';

/** An API error response; branch on problem.type. */
export class ProblemError extends Error {
  constructor(readonly problem: Problem) {
    super(problem.detail || problem.title);
  }
}

/** The fetch that orval's generated functions use: it returns the JSON body and throws on errors. */
export async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (res.ok) return res.json();
  if (res.headers.get('content-type')?.startsWith('application/problem+json')) {
    throw new ProblemError(await res.json());
  }
  throw new Error(`${init?.method ?? 'GET'} ${url}: HTTP ${res.status}`);
}

/** The error type orval gives the generated hooks. */
export type ErrorType<P extends Problem> = ProblemError & { problem: P };
