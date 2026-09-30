import { createRootKey } from "@/lib/root-keys-api";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createRootKeyFromForm } from "./create-root-key";

vi.mock("@/lib/root-keys-api", () => ({ createRootKey: vi.fn() }));

describe("createRootKeyFromForm", () => {
  beforeEach(() => {
    vi.mocked(createRootKey).mockReset();
  });

  it("creates a root key with workspace-scoped URNs", async () => {
    vi.mocked(createRootKey).mockResolvedValue({ keyId: "key_1", key: "unkey_secret" });

    await expect(
      createRootKeyFromForm("ws_1", {
        name: " CI key ",
        policies: [
          {
            scope: "projects",
            instances: ["proj_1"],
            selection: { project: ["read"] },
          },
        ],
      }),
    ).resolves.toEqual({ keyId: "key_1", secret: "unkey_secret" });

    expect(createRootKey).toHaveBeenCalledWith({
      name: "CI key",
      permissions: ["unkey:v1:ws_1:projects/proj_1#read_project"],
    });
  });
});
