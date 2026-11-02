import { QueryClient } from "@tanstack/react-query";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { describe, expect, it, vi } from "vitest";

function projectNotFound(): NotFoundErrorResponse {
  return new NotFoundErrorResponse(
    {
      meta: { requestId: "req_test" },
      error: {
        detail: "The requested project does not exist.",
        status: 404,
        title: "Not Found",
        type: "https://unkey.com/docs/errors/unkey/data/project_not_found",
      },
    },
    {
      request: new Request("https://api.unkey.com/v2/deployments.listDeployments", {
        method: "POST",
      }),
      response: new Response(null, { status: 404 }),
      body: "{}",
    },
  );
}

const listDeployments = vi.fn(
  async (_input: { project: string; limit: number }, _options: { signal?: AbortSignal }) => {
    throw projectNotFound();
  },
);

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ deployments: { listDeployments } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));

const { deploymentsPollIntervalMs, recentDeploymentsOptions } = await import("./deployments");

describe("recentDeploymentsOptions", () => {
  it("lists nothing when the API reports the project as not found", async () => {
    const projectId = `proj_${crypto.randomUUID()}`;
    const queryClient = new QueryClient();

    const rows = await queryClient.fetchQuery(
      recentDeploymentsOptions(queryClient, projectId, undefined, null, 5),
    );

    expect(rows).toEqual([]);
    expect(listDeployments).toHaveBeenCalledWith(
      expect.objectContaining({ project: projectId }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });
});

describe("deploymentsPollIntervalMs", () => {
  it("picks the fastest cadence that applies", () => {
    const idle = { settling: false, requestedMs: null, msSinceChange: 0 };
    expect(deploymentsPollIntervalMs(idle)).toBe(20_000);
    expect(deploymentsPollIntervalMs({ ...idle, requestedMs: 10_000 })).toBe(10_000);
    expect(deploymentsPollIntervalMs({ ...idle, requestedMs: 10_000, settling: true })).toBe(5_000);
  });

  it("backs off polls that change nothing, up to a minute", () => {
    const idle = { settling: false, requestedMs: null };
    expect(deploymentsPollIntervalMs({ ...idle, msSinceChange: 20_000 })).toBe(40_000);
    expect(deploymentsPollIntervalMs({ ...idle, msSinceChange: 60_000 })).toBe(60_000);
    expect(deploymentsPollIntervalMs({ ...idle, msSinceChange: 600_000 })).toBe(60_000);
  });

  it("never backs off while something moves or a page asks for a cadence", () => {
    expect(
      deploymentsPollIntervalMs({ settling: true, requestedMs: null, msSinceChange: 600_000 }),
    ).toBe(5_000);
    expect(
      deploymentsPollIntervalMs({ settling: false, requestedMs: 10_000, msSinceChange: 600_000 }),
    ).toBe(10_000);
  });
});
