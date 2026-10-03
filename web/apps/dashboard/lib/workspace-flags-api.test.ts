import { afterEach, expect, it, vi } from "vitest";
import {
  listWorkspaceFlags,
  removeWorkspaceFlagOverride,
  setWorkspaceFlagOverride,
} from "./workspace-flags-api";

afterEach(() => vi.unstubAllGlobals());

it("loads typed effective values through the authenticated proxy", async () => {
  const flag = {
    slug: "routing-strategy",
    description: "Select routing behavior",
    type: "string",
    defaultValue: "balanced",
    value: "",
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

it("sets zero and removes an override through API endpoints without a workspace id", async () => {
  const flag = {
    slug: "timeout",
    description: "Timeout",
    type: "number",
    defaultValue: 37,
    value: 0,
    hasOverride: true,
    allowOptIn: true,
    allowOptOut: true,
  };
  const fetch = vi
    .fn<typeof globalThis.fetch>()
    .mockResolvedValueOnce(new Response(JSON.stringify({ data: flag })))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ data: { ...flag, value: 37, hasOverride: false } })),
    );
  vi.stubGlobal("fetch", fetch);
  await expect(setWorkspaceFlagOverride("timeout", 0)).resolves.toEqual(flag);
  await expect(removeWorkspaceFlagOverride("timeout")).resolves.toMatchObject({
    value: 37,
    hasOverride: false,
  });
  expect(fetch).toHaveBeenNthCalledWith(
    1,
    "/proxy/v2/flags.setOverride",
    expect.objectContaining({ body: '{"slug":"timeout","value":0}' }),
  );
  expect(fetch).toHaveBeenNthCalledWith(
    2,
    "/proxy/v2/flags.removeOverride",
    expect.objectContaining({ body: '{"slug":"timeout"}' }),
  );
});

it("reports the API denial instead of claiming a value was saved", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ error: { detail: "Only workspace admins can change flag overrides." } }),
          { status: 403 },
        ),
      ),
  );
  await expect(setWorkspaceFlagOverride("timeout", 4)).rejects.toThrow(
    "Only workspace admins can change flag overrides.",
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
              type: "boolean",
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
