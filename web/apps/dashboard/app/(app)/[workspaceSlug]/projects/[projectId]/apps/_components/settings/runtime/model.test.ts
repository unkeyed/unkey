import {
  type EnvironmentSettings,
  buildSettingsUpdate,
} from "@/lib/collections/deploy/environment-settings";
import { describe, expect, it } from "vitest";
import {
  ADVANCED_PATHS,
  HEALTHCHECK_FIELDS,
  hasAdvancedOverrides,
  readRuntime,
  runtimeFormSchema,
  runtimeSchema,
  writeRuntime,
} from "./model";

const stored: EnvironmentSettings = {
  environmentId: "env_prod",
  projectId: "proj_1",
  appId: "app_1",
  autoDeploy: true,
  dockerfile: "",
  dockerContext: ".",
  buildCommand: "",
  watchPaths: [],
  port: 8080,
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  command: ["sh", "-c", "node a.js && node b.js"],
  healthcheck: null,
  regions: [{ name: "us-east-1", replicasMin: 1, replicasMax: 1 }],
  shutdownSignal: "SIGTERM",
  upstreamProtocol: "http1",
  openapiSpecPath: null,
};

function save(edit: (values: ReturnType<typeof readRuntime>) => void): EnvironmentSettings {
  const saved = readRuntime(stored);
  const values = structuredClone(saved);
  edit(values);
  const draft = structuredClone(stored);
  writeRuntime(draft, runtimeSchema.parse(values), saved);
  return draft;
}

describe("writeRuntime", () => {
  it("keeps a stored command with spaced arguments when only compute changes", () => {
    const draft = save((values) => {
      values.compute.replicas = { min: 1, max: 3 };
    });
    expect(draft.command).toEqual(["sh", "-c", "node a.js && node b.js"]);
    expect(draft.regions).toEqual([{ name: "us-east-1", replicasMin: 1, replicasMax: 3 }]);
  });

  it("splits an edited command on whitespace", () => {
    const draft = save((values) => {
      values.command = "  node   server.js --port 3000 ";
    });
    expect(draft.command).toEqual(["node", "server.js", "--port", "3000"]);
  });

  it("keeps the stored command when the edit only adds surrounding spaces", () => {
    const draft = save((values) => {
      values.command = ` ${values.command} `;
    });
    expect(draft.command).toEqual(["sh", "-c", "node a.js && node b.js"]);
  });

  it("leaves the health check off when only the port changes", () => {
    const draft = save((values) => {
      values.port = 3000;
    });
    expect(draft.port).toBe(3000);
    expect(draft.healthcheck).toBeNull();
  });

  describe("health check", () => {
    const healthcheck = {
      method: "POST" as const,
      path: "/ready",
      intervalSeconds: 27,
      timeoutSeconds: 8,
      failureThreshold: 9,
      initialDelaySeconds: 14,
    };
    const withCheck: EnvironmentSettings = { ...stored, healthcheck };
    const saved = readRuntime(withCheck);

    it("keeps each environment's own timing when only the path changes", () => {
      const preview: EnvironmentSettings = {
        ...withCheck,
        environmentId: "env_preview",
        healthcheck: { ...healthcheck, timeoutSeconds: 3 },
      };
      const drafts = [structuredClone(withCheck), structuredClone(preview)];
      for (const draft of drafts) {
        writeRuntime(
          draft,
          { ...saved, healthcheck: { ...saved.healthcheck, path: "/live" } },
          saved,
        );
      }
      expect(drafts.map((draft) => draft.healthcheck)).toEqual([
        { ...healthcheck, path: "/live" },
        { ...healthcheck, path: "/live", timeoutSeconds: 3 },
      ]);
    });

    it("sends all health check settings", () => {
      const draft = save((values) => {
        values.healthcheck = healthcheck;
      });
      expect(buildSettingsUpdate(stored, draft)).toEqual({
        project: stored.projectId,
        app: stored.appId,
        environment: stored.environmentId,
        healthcheck,
      });
      expect(readRuntime(draft).healthcheck).toEqual(healthcheck);
    });

    it("sends an initial delay of 0", () => {
      const draft = structuredClone(withCheck);
      writeRuntime(
        draft,
        { ...saved, healthcheck: { ...saved.healthcheck, initialDelaySeconds: 0 } },
        saved,
      );
      expect(buildSettingsUpdate(withCheck, draft)?.healthcheck).toEqual({
        ...healthcheck,
        initialDelaySeconds: 0,
      });
    });

    it("removes the check when the path is cleared", () => {
      const draft = structuredClone(withCheck);
      writeRuntime(draft, { ...saved, healthcheck: { ...saved.healthcheck, path: "" } }, saved);
      expect(buildSettingsUpdate(withCheck, draft)?.healthcheck).toBeNull();
    });

    it("writes null for an empty path with empty timing values", () => {
      const draft = structuredClone(withCheck);
      writeRuntime(
        draft,
        runtimeSchema.parse({
          ...saved,
          healthcheck: {
            ...saved.healthcheck,
            path: "",
            intervalSeconds: Number.NaN,
            timeoutSeconds: Number.NaN,
          },
        }),
        saved,
      );
      expect(draft.healthcheck).toBeNull();
    });
  });

  it.each(["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"] as const)(
    "sends %s without changing other runtime settings",
    (shutdownSignal) => {
      const original: EnvironmentSettings = {
        ...stored,
        shutdownSignal: shutdownSignal === "SIGTERM" ? "SIGINT" : "SIGTERM",
      };
      const saved = readRuntime(original);
      const draft = structuredClone(original);
      writeRuntime(draft, runtimeSchema.parse({ ...saved, shutdownSignal }), saved);
      expect(buildSettingsUpdate(original, draft)).toEqual({
        project: stored.projectId,
        app: stored.appId,
        environment: stored.environmentId,
        shutdownSignal,
      });
      expect(readRuntime(draft).shutdownSignal).toBe(shutdownSignal);
      expect(draft.command).toEqual(stored.command);
      expect(draft.healthcheck).toBeNull();
    },
  );
});

describe("hasAdvancedOverrides", () => {
  const defaultCheck = {
    method: "GET" as const,
    path: "/health",
    intervalSeconds: 10,
    timeoutSeconds: 5,
    failureThreshold: 3,
    initialDelaySeconds: 0,
  };

  it("is false for default settings with or without a health check", () => {
    expect(hasAdvancedOverrides(readRuntime(stored))).toBe(false);
    expect(hasAdvancedOverrides(readRuntime({ ...stored, healthcheck: defaultCheck }))).toBe(false);
  });

  it("is true for a non-default shutdown signal", () => {
    expect(hasAdvancedOverrides(readRuntime({ ...stored, shutdownSignal: "SIGINT" }))).toBe(true);
  });

  it.each(HEALTHCHECK_FIELDS.map(({ name }) => name))("is true for a non-default %s", (name) => {
    expect(
      hasAdvancedOverrides(
        readRuntime({ ...stored, healthcheck: { ...defaultCheck, [name]: 42 } }),
      ),
    ).toBe(true);
  });

  it("covers every timing error the schema can raise", () => {
    const result = runtimeSchema.safeParse({
      ...readRuntime(stored),
      healthcheck: {
        ...readRuntime(stored).healthcheck,
        path: "/health",
        intervalSeconds: 0,
        timeoutSeconds: 0,
        failureThreshold: 0,
        initialDelaySeconds: -1,
      },
    });
    const paths: string[] = result.error?.issues.map((issue) => issue.path.join(".")) ?? [];
    expect(paths).toHaveLength(HEALTHCHECK_FIELDS.length);
    expect(paths.every((path) => ADVANCED_PATHS.some((watched) => watched === path))).toBe(true);
  });
});

describe("runtimeSchema", () => {
  const values = readRuntime(stored);

  it("accepts an empty interval while the health check path is empty", () => {
    const result = runtimeSchema.safeParse({
      ...values,
      healthcheck: { ...values.healthcheck, path: "", intervalSeconds: Number.NaN },
    });
    expect(result.success).toBe(true);
  });

  it("rejects an empty interval once a health check path is set", () => {
    const result = runtimeSchema.safeParse({
      ...values,
      healthcheck: { ...values.healthcheck, path: "/health", intervalSeconds: Number.NaN },
    });
    expect(result.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
      [["healthcheck", "intervalSeconds"], "Use 1 to 3600 seconds"],
    ]);
  });

  const withField = (
    path: string,
    field: (typeof HEALTHCHECK_FIELDS)[number]["name"],
    value: number,
  ) => ({
    ...values,
    healthcheck: { ...values.healthcheck, path, [field]: value },
  });

  it.each([
    { field: "intervalSeconds", value: 1 },
    { field: "intervalSeconds", value: 3600 },
    { field: "timeoutSeconds", value: 1 },
    { field: "timeoutSeconds", value: 3600 },
    { field: "failureThreshold", value: 1 },
    { field: "failureThreshold", value: 100 },
    { field: "initialDelaySeconds", value: 0 },
    { field: "initialDelaySeconds", value: 3600 },
  ] as const)("accepts $field = $value with a path set", ({ field, value }) => {
    expect(runtimeSchema.safeParse(withField("/health", field, value)).success).toBe(true);
  });

  it.each([
    { field: "intervalSeconds", value: 0 },
    { field: "intervalSeconds", value: 3601 },
    { field: "intervalSeconds", value: 1.5 },
    { field: "intervalSeconds", value: Number.NaN },
    { field: "timeoutSeconds", value: 0 },
    { field: "timeoutSeconds", value: 3601 },
    { field: "timeoutSeconds", value: 1.5 },
    { field: "timeoutSeconds", value: Number.NaN },
    { field: "failureThreshold", value: 0 },
    { field: "failureThreshold", value: 101 },
    { field: "failureThreshold", value: 1.5 },
    { field: "failureThreshold", value: Number.NaN },
    { field: "initialDelaySeconds", value: -1 },
    { field: "initialDelaySeconds", value: 3601 },
    { field: "initialDelaySeconds", value: 1.5 },
    { field: "initialDelaySeconds", value: Number.NaN },
  ] as const)("rejects $field = $value only with a path set", ({ field, value }) => {
    const result = runtimeSchema.safeParse(withField("/health", field, value));
    expect(result.error?.issues.map((issue) => issue.path)).toEqual([["healthcheck", field]]);
    expect(runtimeSchema.safeParse(withField("", field, value)).success).toBe(true);
  });
});

describe("runtimeFormSchema", () => {
  it("lets the start command save while the stored regions are empty", () => {
    const values = {
      ...readRuntime(stored),
      compute: { ...readRuntime(stored).compute, regions: [] },
    };
    expect(runtimeFormSchema("all", 4).safeParse(values).success).toBe(false);
    expect(runtimeFormSchema(["command"], 4).safeParse(values).success).toBe(true);
  });

  it("lets compute save while the stored port is out of range", () => {
    const values = { ...readRuntime(stored), port: 0 };
    expect(runtimeFormSchema(["compute"], 4).safeParse(values).success).toBe(true);
  });

  it("rejects a max above the plan limit with the limit in the message", () => {
    const values = readRuntime(stored);
    values.compute.replicas = { min: 1, max: 9 };
    const result = runtimeFormSchema(["compute"], 4).safeParse(values);
    expect(result.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
      [["compute", "replicas"], "Up to 4 per region on your plan"],
    ]);
  });

  it("rejects an empty instance count with one message", () => {
    const values = readRuntime(stored);
    values.compute.replicas = { min: Number.NaN, max: 2 };
    const result = runtimeFormSchema(["compute"], 4).safeParse(values);
    expect(result.error?.issues.map((issue) => issue.message)).toEqual([
      "Use a whole number from 1 to 4",
    ]);
  });
});
