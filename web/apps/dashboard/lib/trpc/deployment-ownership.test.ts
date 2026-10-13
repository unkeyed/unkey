import { describe, expect, it } from "vitest";
import {
  type OwnedDeployment,
  createDeploymentOwnershipCache,
  loadOwnedDeployment,
} from "./deployment-ownership";

const row: OwnedDeployment = {
  id: "dep_1",
  projectId: "prj_1",
  environmentId: "env_1",
};

describe("loadOwnedDeployment", () => {
  it("runs one lookup when the same deployment is checked more than once", async () => {
    const cache = createDeploymentOwnershipCache();
    let calls = 0;
    const find = async () => {
      calls += 1;
      return row;
    };

    const [first, second] = await Promise.all([
      loadOwnedDeployment(cache, "ws_1", "dep_1", find),
      loadOwnedDeployment(cache, "ws_1", "dep_1", find),
    ]);

    expect(calls).toBe(1);
    expect(first).toBe(row);
    expect(second).toBe(row);
  });

  it("joins an in-flight lookup instead of starting another", async () => {
    const cache = createDeploymentOwnershipCache();
    let calls = 0;
    let release: (deployment: OwnedDeployment) => void = () => {};
    const find = () => {
      calls += 1;
      return new Promise<OwnedDeployment>((resolve) => {
        release = resolve;
      });
    };

    const first = loadOwnedDeployment(cache, "ws_1", "dep_1", find);
    const second = loadOwnedDeployment(cache, "ws_1", "dep_1", find);
    expect(calls).toBe(1);

    release(row);
    await expect(first).resolves.toBe(row);
    await expect(second).resolves.toBe(row);
  });

  it("looks up each deployment separately", async () => {
    const cache = createDeploymentOwnershipCache();
    const seen: string[] = [];
    const find = async (_workspaceId: string, deploymentId: string) => {
      seen.push(deploymentId);
      return { ...row, id: deploymentId };
    };

    await loadOwnedDeployment(cache, "ws_1", "dep_1", find);
    await loadOwnedDeployment(cache, "ws_1", "dep_2", find);

    expect(seen).toEqual(["dep_1", "dep_2"]);
  });

  it("does not reuse an ownership lookup for a different workspace", async () => {
    // The cache key includes the workspace. A hit for one workspace must not
    // satisfy an ownership check for another.
    const cache = createDeploymentOwnershipCache();
    const seen: string[] = [];
    const find = async (workspaceId: string) => {
      seen.push(workspaceId);
      return row;
    };

    await loadOwnedDeployment(cache, "ws_1", "dep_1", find);
    await loadOwnedDeployment(cache, "ws_2", "dep_1", find);

    expect(seen).toEqual(["ws_1", "ws_2"]);
  });

  it("does not reuse a lookup from another request", async () => {
    let calls = 0;
    const find = async () => {
      calls += 1;
      return row;
    };

    await loadOwnedDeployment(createDeploymentOwnershipCache(), "ws_1", "dep_1", find);
    await loadOwnedDeployment(createDeploymentOwnershipCache(), "ws_1", "dep_1", find);

    expect(calls).toBe(2);
  });

  it("remembers a missing deployment", async () => {
    const cache = createDeploymentOwnershipCache();
    let calls = 0;
    const find = async () => {
      calls += 1;
      return undefined;
    };

    await expect(loadOwnedDeployment(cache, "ws_1", "dep_missing", find)).resolves.toBeUndefined();
    await expect(loadOwnedDeployment(cache, "ws_1", "dep_missing", find)).resolves.toBeUndefined();
    expect(calls).toBe(1);
  });

  it("shares a failed lookup instead of retrying it per procedure", async () => {
    const cache = createDeploymentOwnershipCache();
    let calls = 0;
    const find = async () => {
      calls += 1;
      throw new Error("db down");
    };

    const first = loadOwnedDeployment(cache, "ws_1", "dep_1", find);
    const second = loadOwnedDeployment(cache, "ws_1", "dep_1", find);

    await expect(first).rejects.toThrow("db down");
    await expect(second).rejects.toThrow("db down");
    expect(calls).toBe(1);
  });
});
