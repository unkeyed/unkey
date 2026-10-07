import { describe, expect, it, vi } from "vitest";

let serverPort = 8080;
const listEnvironments = vi.fn(async () => {
  const port = serverPort;
  await new Promise((resolve) => setTimeout(resolve, 50));
  return { data: [{ id: "env_prod", runtime: { port } }] };
});

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { listEnvironments } }),
}));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { listAppEnvironments, markAppEnvironmentsChanged } = await import("./app-environments");

function portOf(environments: Awaited<ReturnType<typeof listAppEnvironments>>) {
  return environments[0]?.runtime?.port;
}

describe("listAppEnvironments", () => {
  it("shares one request between overlapping calls", async () => {
    listEnvironments.mockClear();
    await Promise.all([listAppEnvironments("proj", "app_a"), listAppEnvironments("proj", "app_a")]);
    expect(listEnvironments).toHaveBeenCalledTimes(1);
  });

  it("reads again after a write, even while an older request is in flight", async () => {
    serverPort = 8080;
    const beforeWrite = listAppEnvironments("proj", "app_b");
    await new Promise((resolve) => setTimeout(resolve, 10));
    serverPort = 9090;
    markAppEnvironmentsChanged("proj", "app_b");

    expect(portOf(await listAppEnvironments("proj", "app_b"))).toBe(9090);
    await beforeWrite;
  });
});
