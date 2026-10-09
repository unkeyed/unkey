import { describe, expect, it } from "vitest";
import { addEnvVarsInput, envVarRows } from "./add-plan";

const variable = (key: string) => ({ key, value: "v", kind: "recoverable" as const });

describe("addEnvVarsInput", () => {
  it("rejects a key listed twice", () => {
    const parsed = addEnvVarsInput.safeParse({
      appId: "app_1",
      environmentIds: ["env_1"],
      variables: [variable("A"), variable("A")],
    });

    expect(parsed.success).toBe(false);
  });
});

describe("envVarRows", () => {
  it("stores the ciphertext, not the value, under the given environment", () => {
    const rows = envVarRows({
      workspaceId: "ws_1",
      appId: "app_1",
      environmentId: "env_1",
      variables: [{ ...variable("A"), kind: "writeonly", id: "var_1", ciphertext: "enc" }],
    });

    expect(rows).toEqual([
      {
        id: "var_1",
        workspaceId: "ws_1",
        appId: "app_1",
        environmentId: "env_1",
        key: "A",
        value: "enc",
        type: "writeonly",
        description: null,
      },
    ]);
  });
});
