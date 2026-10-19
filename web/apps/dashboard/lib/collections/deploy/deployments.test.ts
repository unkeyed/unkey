import { createLiveQueryCollection, eq } from "@tanstack/react-db";
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

const listDeployments = vi.fn(async (_input: { project: string; limit: number }) => {
  throw projectNotFound();
});

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ deployments: { listDeployments } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return {
    queryClient: new QueryClient(),
    trpcClient: {
      deploy: { deployment: { listDetails: { query: vi.fn(async () => ({})) } } },
    },
  };
});

const { deployments } = await import("./deployments");

describe("deployments collection", () => {
  it("loads an empty list when the API reports the project as not found", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ deployment: deployments })
        .where(({ deployment }) => eq(deployment.projectId, "proj_1")),
    );

    await query.preload();

    await vi.waitFor(() => expect(listDeployments).toHaveBeenCalledTimes(1));
    expect(query.toArray).toEqual([]);
    expect(listDeployments).toHaveBeenCalledWith(expect.objectContaining({ project: "proj_1" }));
  });
});
