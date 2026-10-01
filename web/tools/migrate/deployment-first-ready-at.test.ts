import assert from "node:assert/strict";
import test from "node:test";
import mysql, { type RowDataPacket } from "mysql2/promise";
import { backfillDeploymentFirstReadyAt } from "./deployment-first-ready-at";

const databaseUrl = process.env.MIGRATION_TEST_DATABASE_URL;

test(
  "backfills only deployments with ready proof in bounded batches",
  { skip: databaseUrl ? false : "MIGRATION_TEST_DATABASE_URL is not set" },
  async (t) => {
    assert.ok(databaseUrl);
    const url = new URL(databaseUrl);
    assert.ok(["localhost", "127.0.0.1", "::1"].includes(url.hostname));

    const database = `migration_first_ready_at_${process.pid}`;
    const admin = mysql.createPool(databaseUrl);
    await admin.query(`CREATE DATABASE \`${database}\``);
    url.pathname = `/${database}`;
    const pool = mysql.createPool(url.toString());

    t.after(async () => {
      await pool.end();
      await admin.query(`DROP DATABASE IF EXISTS \`${database}\``);
      await admin.end();
    });

    await pool.query(`CREATE TABLE deployments (
      pk BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      id VARCHAR(48) NOT NULL UNIQUE,
      status ENUM('pending', 'ready', 'failed', 'awaiting_approval', 'stopped') NOT NULL,
      first_ready_at BIGINT NULL,
      created_at BIGINT NOT NULL,
      updated_at BIGINT NULL
    )`);
    await pool.query(`CREATE TABLE deployment_steps (
      pk BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      deployment_id VARCHAR(48) NOT NULL,
      step ENUM('building', 'finalizing') NOT NULL,
      ended_at BIGINT UNSIGNED NULL,
      error VARCHAR(512) NULL,
      UNIQUE KEY unique_step_per_deployment (deployment_id, step)
    )`);

    await pool.query(
      `INSERT INTO deployments (id, status, first_ready_at, created_at, updated_at) VALUES
       ('ready-no-history', 'ready', NULL, 100, 110),
       ('ready-no-updated-at', 'ready', NULL, 120, NULL),
       ('stopped-successful-history', 'stopped', NULL, 200, 250),
       ('stopped-unsuccessful-history', 'stopped', NULL, 270, 280),
       ('failed-finalizing', 'failed', NULL, 300, 350),
       ('pending-no-history', 'pending', NULL, 400, 450),
       ('unapproved-no-history', 'awaiting_approval', NULL, 500, 550),
       ('preserved', 'ready', 42, 600, 650)`,
    );
    await pool.query(
      `INSERT INTO deployment_steps (deployment_id, step, ended_at, error) VALUES
       ('stopped-successful-history', 'finalizing', 260, NULL),
       ('stopped-unsuccessful-history', 'finalizing', 290, 'failed to promote'),
       ('failed-finalizing', 'finalizing', 360, 'failed to promote')`,
    );

    const firstRun = await backfillDeploymentFirstReadyAt(pool, 2);
    assert.deepEqual(firstRun, { batches: [2, 1], updated: 3 });

    const [rows] = await pool.query<
      (RowDataPacket & { id: string; first_ready_at: number | null })[]
    >("SELECT id, first_ready_at FROM deployments ORDER BY pk");
    assert.deepEqual(Object.fromEntries(rows.map((row) => [row.id, row.first_ready_at])), {
      "ready-no-history": 110,
      "ready-no-updated-at": 120,
      "stopped-successful-history": 260,
      "stopped-unsuccessful-history": null,
      "failed-finalizing": null,
      "pending-no-history": null,
      "unapproved-no-history": null,
      preserved: 42,
    });

    assert.deepEqual(await backfillDeploymentFirstReadyAt(pool, 2), {
      batches: [0],
      updated: 0,
    });
  },
);
