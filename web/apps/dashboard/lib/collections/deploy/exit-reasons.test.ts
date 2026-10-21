import { describe, expect, test } from "vitest";
import { isFatalWaitingReason, reasonCopy } from "./exit-reasons";

describe("reasonCopy", () => {
  test("gives plain copy for known reasons", () => {
    expect(reasonCopy("CrashLoopBackOff")?.label).toBe("Crashing");
    expect(reasonCopy("ErrImagePull")?.label).toBe("Can't pull image");
  });

  test("is null for unknown, inherited or missing reasons", () => {
    expect(reasonCopy("RunContainerError")).toBe(null);
    expect(reasonCopy("toString")).toBe(null);
    expect(reasonCopy(null)).toBe(null);
  });
});

describe("isFatalWaitingReason", () => {
  test("is true only for waiting reasons that stop an instance", () => {
    expect(isFatalWaitingReason("CrashLoopBackOff")).toBe(true);
    expect(isFatalWaitingReason("CreateContainerConfigError")).toBe(true);
    expect(isFatalWaitingReason("OOMKilled")).toBe(false);
    expect(isFatalWaitingReason("ContainerCreating")).toBe(false);
    expect(isFatalWaitingReason(null)).toBe(false);
  });
});
