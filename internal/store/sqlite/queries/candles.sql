-- SPDX-License-Identifier: AGPL-3.0-only

-- name: UpsertCandle :exec
INSERT INTO candles (instrument_id, tf, t, o, h, l, c, v, qv, n, rev, derived)
VALUES (
    sqlc.arg(instrument_id), sqlc.arg(tf), sqlc.arg(t), sqlc.arg(o), sqlc.arg(h), sqlc.arg(l),
    sqlc.arg(c), sqlc.arg(v), sqlc.narg(qv), sqlc.narg(n), sqlc.arg(rev), sqlc.arg(derived)
)
ON CONFLICT (instrument_id, tf, t) DO UPDATE SET
    o = excluded.o, h = excluded.h, l = excluded.l, c = excluded.c, v = excluded.v,
    qv = excluded.qv, n = excluded.n, rev = excluded.rev, derived = excluded.derived
WHERE excluded.rev >= candles.rev;

-- name: Candles :many
SELECT instrument_id, tf, t, o, h, l, c, v, qv, n, rev, derived
FROM candles
WHERE instrument_id = sqlc.arg(instrument_id) AND tf = sqlc.arg(tf)
    AND t >= sqlc.arg(from_t) AND t < sqlc.arg(to_t)
ORDER BY t
LIMIT sqlc.arg(row_limit);

-- name: LastCandleT :one
SELECT t
FROM candles
WHERE instrument_id = sqlc.arg(instrument_id) AND tf = sqlc.arg(tf)
ORDER BY t DESC
LIMIT 1;
