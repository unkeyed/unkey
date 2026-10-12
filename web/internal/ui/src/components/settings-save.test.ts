import { describe, expect, it } from "vitest";
import { formSaveState, resolveGroupSave } from "./settings-save";

const name = { id: "name", dirty: true, saveState: { status: "ready" } } as const;
const prefix = { id: "prefix", dirty: true, saveState: { status: "ready" } } as const;
const invalidBytes = {
  id: "bytes",
  dirty: true,
  saveState: { status: "disabled", reason: "Fix the invalid fields to save" },
} as const;
const untouched = { id: "id", dirty: false, saveState: { status: "disabled" } } as const;

describe("resolveGroupSave", () => {
  it("is clean when no row is dirty", () => {
    expect(resolveGroupSave([untouched])).toEqual({ status: "clean" });
  });

  it("submits only the valid dirty rows and counts every dirty row", () => {
    expect(resolveGroupSave([name, invalidBytes, prefix, untouched])).toEqual({
      status: "ready",
      submit: [name, prefix],
      dirty: 3,
    });
  });

  it("is blocked when every dirty row is invalid", () => {
    expect(resolveGroupSave([invalidBytes, untouched])).toEqual({
      status: "blocked",
      reasons: ["Fix the invalid fields to save"],
    });
  });

  it("lists each reason once and drops empty reasons", () => {
    const admin = {
      id: "admin",
      dirty: true,
      saveState: { status: "disabled", reason: "Admin access required" },
    } as const;
    const silent = { id: "silent", dirty: true, saveState: { status: "disabled" } } as const;
    expect(resolveGroupSave([invalidBytes, admin, invalidBytes, silent])).toEqual({
      status: "blocked",
      reasons: ["Fix the invalid fields to save", "Admin access required"],
    });
  });

  it("is saving while any row is saving, even beside an invalid row", () => {
    const saving = { id: "saving", dirty: true, saveState: { status: "saving" } } as const;
    expect(resolveGroupSave([saving, invalidBytes])).toEqual({ status: "saving" });
  });
});

describe("formSaveState", () => {
  const idle = { isSubmitting: false, isValid: true, isDirty: true };

  it("is ready when dirty and valid", () => {
    expect(formSaveState(idle)).toEqual({ status: "ready" });
  });

  it("puts saving before blocked, invalid and unchanged", () => {
    expect(
      formSaveState({
        isSubmitting: true,
        isValid: false,
        isDirty: false,
        blockedReason: "No access",
      }),
    ).toEqual({ status: "saving" });
  });

  it("puts blocked before invalid and unchanged", () => {
    expect(
      formSaveState({ ...idle, isValid: false, isDirty: false, blockedReason: "No access" }),
    ).toEqual({ status: "disabled", reason: "No access" });
  });

  it("puts invalid before unchanged", () => {
    expect(formSaveState({ ...idle, isValid: false, isDirty: false })).toEqual({
      status: "disabled",
      reason: "Fix the invalid fields to save",
    });
  });

  it("is disabled without a reason when unchanged", () => {
    expect(formSaveState({ ...idle, isDirty: false })).toEqual({ status: "disabled" });
  });
});
