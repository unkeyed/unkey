import type { RollbackEvent } from "@/lib/collections/deploy/rollback";
import type { Loadable } from "@/lib/loadable/loadable";
import { describe, expect, it } from "vitest";
import { liveFactsOf } from "./deploy-stamp";

const live = {
  id: "d_live",
  environmentId: "env_prod",
  status: "ready" as const,
  createdAt: 1_000,
};
const named = { ...live, id: "d_named", createdAt: 1_500 };
const newest = { ...live, id: "d_newest", createdAt: 3_000 };
const deployments = [newest, named, live];

const author = { handle: "octocat", avatarUrl: "https://example.com/octocat.png" };
const event: RollbackEvent = {
  at: 2_000,
  actor: { name: "Ada Lovelace", avatarUrl: "https://example.com/ada.png" },
  fromDeploymentId: "d_named",
};
const ready = (data: RollbackEvent | null): Loadable<RollbackEvent | null> => ({
  status: "ready",
  data,
  stale: null,
});

describe("liveFactsOf", () => {
  it("shows nothing for a deployment that is not live", () => {
    expect(liveFactsOf({ kind: "not-live" }, live, deployments)).toEqual({
      stamp: null,
      replaced: null,
    });
  });

  it("credits the commit author with the build time", () => {
    expect(liveFactsOf({ kind: "live", author }, live, deployments)).toEqual({
      stamp: {
        verb: "Deployed",
        at: 1_000,
        actor: { name: "octocat", avatarUrl: "https://example.com/octocat.png" },
      },
      replaced: null,
    });
  });

  it("has no actor when the commit author has no handle", () => {
    const anonymous = { handle: null, avatarUrl: null };
    expect(liveFactsOf({ kind: "live", author: anonymous }, live, deployments).stamp?.actor).toBe(
      null,
    );
  });

  it("credits whoever rolled back and names the replaced deployment", () => {
    const facts = liveFactsOf({ kind: "rolled-back", event: ready(event) }, live, deployments);
    expect(facts).toEqual({
      stamp: {
        verb: "Rolled back",
        at: 2_000,
        actor: { name: "Ada Lovelace", avatarUrl: "https://example.com/ada.png" },
      },
      replaced: named,
    });
  });

  it("shows nothing while the rollback event loads", () => {
    const loading: Loadable<RollbackEvent | null> = { status: "loading" };
    expect(liveFactsOf({ kind: "rolled-back", event: loading }, live, deployments)).toEqual({
      stamp: null,
      replaced: null,
    });
  });

  it("hides the stamp and infers the replaced deployment when the event is missing", () => {
    expect(liveFactsOf({ kind: "rolled-back", event: ready(null) }, live, deployments)).toEqual({
      stamp: null,
      replaced: newest,
    });
  });

  it("treats a failed lookup like a missing event", () => {
    const failed: Loadable<RollbackEvent | null> = {
      status: "failed",
      error: new Error("boom"),
      retry: () => {},
    };
    expect(liveFactsOf({ kind: "rolled-back", event: failed }, live, deployments)).toEqual({
      stamp: null,
      replaced: newest,
    });
  });
});
