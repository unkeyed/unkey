import { describe, expect, it } from "vitest";
import { createBackgroundWork } from "./background-work";

function deferred() {
  let resolve: () => void = () => {};
  let reject: (error: Error) => void = () => {};
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const tick = () => new Promise((resolve) => setTimeout(resolve, 0));

describe("createBackgroundWork", () => {
  it("starts done and reports running until every task settles", async () => {
    const work = createBackgroundWork();
    expect(work.status()).toEqual({ kind: "done" });
    const link = deferred();
    const settings = deferred();
    work.run("link", () => link.promise);
    work.run("settings", () => settings.promise);
    expect(work.status()).toEqual({ kind: "running" });
    link.resolve();
    await tick();
    expect(work.status()).toEqual({ kind: "running" });
    settings.resolve();
    await tick();
    expect(work.status()).toEqual({ kind: "done" });
    await expect(work.settled()).resolves.toBeUndefined();
  });

  it("runs tasks with the same key in order and different keys at once", async () => {
    const work = createBackgroundWork();
    const order: string[] = [];
    const first = deferred();
    work.run("link", async () => {
      order.push("link:main start");
      await first.promise;
      order.push("link:main end");
    });
    work.run("settings", async () => {
      order.push("settings");
    });
    work.run("link", async () => {
      order.push("link:canary");
    });
    await tick();
    expect(order).toEqual(["link:main start", "settings"]);
    first.resolve();
    await work.settled();
    expect(order).toEqual(["link:main start", "settings", "link:main end", "link:canary"]);
  });

  it("reports the failure, rejects settled, and retry re-runs only the failed task", async () => {
    const work = createBackgroundWork();
    let attempts = 0;
    let settingsRuns = 0;
    work.run("link", async () => {
      attempts++;
      if (attempts === 1) {
        throw new Error("GitHub is down");
      }
    });
    work.run("settings", async () => {
      settingsRuns++;
    });
    await expect(work.settled()).rejects.toThrow("GitHub is down");
    expect(work.status()).toEqual({ kind: "failed", error: new Error("GitHub is down") });
    work.retry();
    expect(work.status()).toEqual({ kind: "running" });
    await expect(work.settled()).resolves.toBeUndefined();
    expect(attempts).toBe(2);
    expect(settingsRuns).toBe(1);
  });

  it("lets a newer task under the same key supersede a failed one", async () => {
    const work = createBackgroundWork();
    const runs: string[] = [];
    work.run("link", async () => {
      runs.push("main");
      throw new Error("no");
    });
    await expect(work.settled()).rejects.toThrow("no");
    work.run("link", async () => {
      runs.push("canary");
    });
    await expect(work.settled()).resolves.toBeUndefined();
    work.retry();
    await work.settled();
    expect(runs).toEqual(["main", "canary"]);
  });

  it("notifies subscribers on every status change", async () => {
    const work = createBackgroundWork();
    const seen: string[] = [];
    work.subscribe(() => seen.push(work.status().kind));
    work.run("link", async () => {});
    await work.settled();
    expect(seen).toEqual(["running", "done"]);
  });
});
