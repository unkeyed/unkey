import { afterEach, expect, it, vi } from "vitest";
import {
  listWorkspaceFlags,
  removeWorkspaceFlagOverride,
  setWorkspaceFlagOverride,
} from "./workspace-flags-api";

afterEach(() => vi.unstubAllGlobals());

it("loads boolean effective values through the authenticated proxy", async () => {
  const flag = {
    slug: "preview",
    description: "Preview deployments",
    defaultValue: true,
    value: false,
    hasOverride: true,
    allowOptIn: true,
    allowOptOut: false,
  };
  const fetch = vi
    .fn<typeof globalThis.fetch>()
    .mockResolvedValue(new Response(JSON.stringify({ data: [flag] }), { status: 200 }));
  vi.stubGlobal("fetch", fetch);
  await expect(listWorkspaceFlags()).resolves.toEqual([flag]);
  expect(fetch).toHaveBeenCalledWith(
    "/proxy/v2/flags.listFlags",
    expect.objectContaining({
      method: "POST",
      body: "{}",
    }),
  );
});

it("sets false and restores a true default through API endpoints without a workspace id", async () => {
  const flag = {
    slug: "preview",
    description: "Preview deployments",
    defaultValue: true,
    value: false,
    hasOverride: true,
    allowOptIn: true,
    allowOptOut: true,
  };
  const fetch = vi
    .fn<typeof globalThis.fetch>()
    .mockResolvedValueOnce(new Response(JSON.stringify({ data: flag })))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ data: { ...flag, value: true, hasOverride: false } })),
    );
  vi.stubGlobal("fetch", fetch);
  await expect(setWorkspaceFlagOverride("preview", false)).resolves.toEqual(flag);
  await expect(removeWorkspaceFlagOverride("preview")).resolves.toMatchObject({
    value: true,
    hasOverride: false,
  });
  expect(fetch).toHaveBeenNthCalledWith(
    1,
    "/proxy/v2/flags.setOverride",
    expect.objectContaining({ body: '{"slug":"preview","value":false}' }),
  );
  expect(fetch).toHaveBeenNthCalledWith(
    2,
    "/proxy/v2/flags.removeOverride",
    expect.objectContaining({ body: '{"slug":"preview"}' }),
  );
});

it("reports the API denial instead of claiming a value was saved", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { detail: "Only workspace admins can change platform features." },
          }),
          { status: 403 },
        ),
      ),
  );
  await expect(setWorkspaceFlagOverride("preview", true)).rejects.toThrow(
    "Only workspace admins can change platform features.",
  );
});

it.each(["false", null, 0])("rejects a boolean flag with invalid value %j", async (value) => {
  vi.stubGlobal(
    "fetch",
    vi.fn<typeof globalThis.fetch>().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: [
            {
              slug: "preview",
              description: "Preview",
              defaultValue: false,
              value,
              hasOverride: true,
              allowOptIn: true,
              allowOptOut: true,
            },
          ],
        }),
      ),
    ),
  );
  await expect(listWorkspaceFlags()).rejects.toThrow();
});
