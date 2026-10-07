import { describe, expect, it } from "vitest";
import { formatReplicas, inheritedSummary } from "./format";

describe("formatReplicas", () => {
  it("writes one count, a range, or mixed", () => {
    expect(formatReplicas({ kind: "uniform", min: 2, max: 2 })).toBe("2");
    expect(formatReplicas({ kind: "uniform", min: 1, max: 3 })).toBe("1–3");
    expect(formatReplicas({ kind: "mixed", first: { min: 1, max: 3 } })).toBe("Mixed");
  });
});

describe("inheritedSummary", () => {
  it("names the instance count, size label, CPU and memory", () => {
    const view = {
      region: "us-east-1",
      sizeMode: "preset" as const,
      cpuMillicores: 500,
      memoryMib: 1024,
      storageMode: "preset" as const,
      storageMib: 0,
    };
    expect(inheritedSummary({ ...view, replicas: { kind: "uniform", min: 1, max: 1 } })).toBe(
      "1 instance · S · 1/2 vCPU · 1 GiB",
    );
    expect(inheritedSummary({ ...view, replicas: { kind: "uniform", min: 1, max: 4 } })).toBe(
      "1–4 instances · S · 1/2 vCPU · 1 GiB",
    );
  });
});
