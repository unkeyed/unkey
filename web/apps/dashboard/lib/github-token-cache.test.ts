import { describe, expect, it } from "vitest";
import { createInstallationTokenCache } from "./github-token-cache";

const HOUR_MS = 60 * 60 * 1000;

function setup() {
  let clock = 0;
  const minted: number[] = [];
  const get = createInstallationTokenCache(
    async (installationId) => {
      minted.push(installationId);
      return {
        token: `t${minted.length}`,
        expires_at: new Date(clock + HOUR_MS).toISOString(),
      };
    },
    () => clock,
  );
  return {
    get,
    minted,
    advance: (ms: number) => {
      clock += ms;
    },
  };
}

describe("createInstallationTokenCache", () => {
  it("reuses a token per installation until it nears expiry", async () => {
    const { get, minted, advance } = setup();
    expect((await get(1)).token).toBe("t1");
    expect((await get(1)).token).toBe("t1");
    expect((await get(2)).token).toBe("t2");
    advance(54 * 60 * 1000);
    expect((await get(1)).token).toBe("t1");
    advance(2 * 60 * 1000);
    expect((await get(1)).token).toBe("t3");
    expect(minted).toEqual([1, 2, 1]);
  });

  it("shares one mint between concurrent callers", async () => {
    const { get, minted } = setup();
    const [a, b] = await Promise.all([get(1), get(1)]);
    expect([a.token, b.token]).toEqual(["t1", "t1"]);
    expect(minted).toEqual([1]);
  });

  it("mints again after a failed mint", async () => {
    let calls = 0;
    const get = createInstallationTokenCache(
      async () => {
        calls++;
        if (calls === 1) {
          throw new Error("github down");
        }
        return { token: "ok", expires_at: new Date(HOUR_MS).toISOString() };
      },
      () => 0,
    );
    await expect(get(1)).rejects.toThrow("github down");
    expect((await get(1)).token).toBe("ok");
  });
});
