-- SPDX-License-Identifier: AGPL-3.0-only

-- +goose Up
CREATE TABLE candles (
    instrument_id integer          NOT NULL,
    tf            smallint         NOT NULL,
    t             bigint           NOT NULL,
    o             double precision NOT NULL,
    h             double precision NOT NULL,
    l             double precision NOT NULL,
    c             double precision NOT NULL,
    v             double precision NOT NULL,
    qv            double precision,
    n             integer,
    rev           smallint         NOT NULL DEFAULT 0,
    derived       boolean          NOT NULL DEFAULT false,
    PRIMARY KEY (instrument_id, tf, t)
);
