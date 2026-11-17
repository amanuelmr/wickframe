-- SPDX-License-Identifier: AGPL-3.0-only

-- +goose Up
CREATE TABLE candles (
    instrument_id INTEGER NOT NULL,
    tf            INTEGER NOT NULL,
    t             INTEGER NOT NULL,
    o             REAL    NOT NULL,
    h             REAL    NOT NULL,
    l             REAL    NOT NULL,
    c             REAL    NOT NULL,
    v             REAL    NOT NULL,
    qv            REAL,
    n             INTEGER,
    rev           INTEGER NOT NULL DEFAULT 0,
    derived       INTEGER NOT NULL DEFAULT 0 CHECK (derived IN (0, 1)),
    PRIMARY KEY (instrument_id, tf, t)
) STRICT, WITHOUT ROWID;
