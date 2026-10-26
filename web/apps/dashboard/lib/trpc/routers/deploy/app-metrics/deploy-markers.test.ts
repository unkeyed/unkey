import { describe, expect, it } from "vitest";
import { toDeployMarkers } from "./deploy-markers";

const grid = { startMs: 1_000, endMs: 2_000 };

function row(id: string, firstReadyAt: number | null) {
  return {
    id,
    firstReadyAt,
    gitCommitSha: null,
    gitCommitMessage: null,
    gitCommitAuthorHandle: null,
    gitCommitAuthorAvatarUrl: null,
    imageRequested: null,
  };
}

const routed = { isRolledBack: false, currentReadyAtMs: 1_800 };

function spans(rows: ReturnType<typeof row>[], routing = routed) {
  return toDeployMarkers(rows, grid, routing).map((m) => [
    m.deploymentId,
    m.readyAtMs,
    m.serving?.untilMs,
  ]);
}

describe("toDeployMarkers", () => {
  it("orders by ready time and chains each serving span to the next marker", () => {
    expect(spans([row("d_c", 1_800), row("d_a", 1_200), row("d_b", 1_500)])).toEqual([
      ["d_a", 1_200, 1_500],
      ["d_b", 1_500, 1_800],
      ["d_c", 1_800, null],
    ]);
  });

  it("keeps only the last deploy before the range start, not an older rollback target", () => {
    expect(spans([row("d_rollback", 100), row("d_serving", 900), row("d_new", 1_400)])).toEqual([
      ["d_serving", 900, 1_400],
      ["d_new", 1_400, null],
    ]);
  });

  it("returns the serving deploy alone when none became ready in range", () => {
    expect(spans([row("d_old", 100), row("d_serving", 900)])).toEqual([["d_serving", 900, null]]);
  });

  it("drops deploys that never became ready or became ready after the range", () => {
    expect(spans([row("d_failed", null), row("d_in", 1_000), row("d_after", 2_000)])).toEqual([
      ["d_in", 1_000, null],
    ]);
  });

  it("returns nothing when no deploy is ready", () => {
    expect(spans([row("d_failed", null)])).toEqual([]);
  });

  it("breaks ready-time ties by id", () => {
    expect(spans([row("d_b", 1_500), row("d_a", 1_500)])).toEqual([
      ["d_a", 1_500, 1_500],
      ["d_b", 1_500, null],
    ]);
  });

  it("drops serving spans while the app is rolled back", () => {
    expect(
      spans([row("d_a", 1_200), row("d_b", 1_500)], {
        isRolledBack: true,
        currentReadyAtMs: 1_200,
      }),
    ).toEqual([
      ["d_a", 1_200, undefined],
      ["d_b", 1_500, undefined],
    ]);
  });

  it("drops serving spans when an older deploy was promoted over a newer ready one", () => {
    expect(
      spans([row("d_a", 1_200), row("d_b", 1_500)], {
        isRolledBack: false,
        currentReadyAtMs: 1_200,
      }),
    ).toEqual([
      ["d_a", 1_200, undefined],
      ["d_b", 1_500, undefined],
    ]);
  });

  it("keeps serving spans when the current deploy became ready after the range", () => {
    expect(
      spans([row("d_a", 1_200), row("d_b", 1_500)], {
        isRolledBack: false,
        currentReadyAtMs: 2_400,
      }),
    ).toEqual([
      ["d_a", 1_200, 1_500],
      ["d_b", 1_500, null],
    ]);
  });

  it("gives preview deploys markers without serving spans", () => {
    expect(toDeployMarkers([row("d_old", 900), row("d_new", 1_400)], grid, null)).toEqual([
      { deploymentId: "d_old", readyAtMs: 900, serving: null, source: null },
      { deploymentId: "d_new", readyAtMs: 1_400, serving: null, source: null },
    ]);
  });

  it("describes a git deploy by its commit", () => {
    const [marker] = toDeployMarkers(
      [
        {
          ...row("d_git", 1_200),
          gitCommitSha: "c7bd7700d1b4b4db1c0b0b2ab5bfab323a06ba03",
          gitCommitMessage: "fix checkout totals",
          gitCommitAuthorHandle: "acme-dev",
          gitCommitAuthorAvatarUrl: "https://avatars.example.com/acme-dev",
        },
      ],
      grid,
      routed,
    );
    expect(marker.source).toEqual({
      kind: "git",
      sha: "c7bd7700d1b4b4db1c0b0b2ab5bfab323a06ba03",
      message: "fix checkout totals",
      authorHandle: "acme-dev",
      avatarUrl: "https://avatars.example.com/acme-dev",
    });
  });

  it("describes a prebuilt image deploy by its requested ref", () => {
    const [marker] = toDeployMarkers(
      [{ ...row("d_image", 1_200), imageRequested: "ghcr.io/acme/checkout-api:1.4.2" }],
      grid,
      routed,
    );
    expect(marker.source).toEqual({ kind: "image", ref: "ghcr.io/acme/checkout-api:1.4.2" });
  });
});
