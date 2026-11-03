import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { newId } from "@unkey/id";
import { describe, expect, it, vi } from "vitest";

const listEnvironments = vi.fn();
vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { listEnvironments } }),
}));

const { loadAppEnvironments } = await import("./environments");

describe("loadAppEnvironments", () => {
  it("tags every environment with its project and app", async () => {
    const projectId = newId("project");
    const appId = newId("app");
    const production = { id: newId("environment"), slug: "production", kind: "production" };
    listEnvironments.mockResolvedValue({ data: [production] });

    expect(await loadAppEnvironments(projectId, appId)).toEqual([
      { ...production, projectId, appId },
    ]);
    expect(listEnvironments).toHaveBeenCalledWith(
      { project: projectId, app: appId },
      { signal: undefined },
    );
  });

  it("lists nothing for an app that no longer exists", async () => {
    listEnvironments.mockRejectedValue(
      new NotFoundErrorResponse(
        {
          meta: { requestId: newId("request") },
          error: {
            detail: "The requested app does not exist.",
            status: 404,
            title: "Not Found",
            type: "https://unkey.com/docs/errors/unkey/data/app_not_found",
          },
        },
        {
          request: new Request("https://api.unkey.com/v2/environments.listEnvironments", {
            method: "POST",
          }),
          response: new Response(null, { status: 404 }),
          body: "{}",
        },
      ),
    );

    expect(await loadAppEnvironments(newId("project"), newId("app"))).toEqual([]);
  });
});
