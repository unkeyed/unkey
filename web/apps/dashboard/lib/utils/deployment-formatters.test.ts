import { describe, expect, it } from "vitest";
import { formatCpu, formatMemory, formatStorage } from "./deployment-formatters";

describe("formatCpu", () => {
  it("writes fractions of a vCPU, whole vCPUs and millicores", () => {
    expect([250, 500, 1000, 4000, 1250, 0].map(formatCpu)).toEqual([
      "1/4 vCPU",
      "1/2 vCPU",
      "1 vCPU",
      "4 vCPU",
      "1250m vCPU",
      "—",
    ]);
  });
});

describe("formatMemory", () => {
  it("switches to GiB at 1024 MiB", () => {
    expect([256, 1024, 1536, 0].map(formatMemory)).toEqual(["256 MiB", "1 GiB", "1.5 GiB", "—"]);
  });
});

describe("formatStorage", () => {
  it("reads zero as no disk", () => {
    expect([0, 512, 5120].map(formatStorage)).toEqual(["None", "512 MiB", "5 GiB"]);
  });
});
