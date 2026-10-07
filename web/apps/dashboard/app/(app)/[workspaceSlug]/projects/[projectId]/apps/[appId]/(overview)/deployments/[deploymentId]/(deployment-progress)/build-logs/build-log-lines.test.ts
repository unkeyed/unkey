import type { BuildLogEntry } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import { toBuildLogLines, toBuildLogText } from "./build-log-lines";

function entry(message: string, overrides: Partial<BuildLogEntry> = {}): BuildLogEntry {
  return {
    message,
    output: "stdout",
    step: "Install dependencies",
    stepId: "step_install",
    time: 1,
    ...overrides,
  };
}

function entryTexts(lines: ReturnType<typeof toBuildLogLines>) {
  return lines.flatMap((line) => (line.kind === "entry" ? [[line.tone, line.text]] : []));
}

describe("toBuildLogLines", () => {
  it("marks warning lines on either stream and leaves other stderr plain", () => {
    const lines = toBuildLogLines(
      [
        entry("\x1b[33mWARN\x1b[0m  deprecated inflight@1.0.6\n", { output: "stderr" }),
        entry("WARNING! corepack is off\n"),
        entry("remote: Counting objects\n", { output: "stderr" }),
        entry("ERROR: failed to solve\n", { output: "stderr" }),
        entry('Error: Command "pnpm install" exited with 1\n'),
      ],
      new Set(),
    );

    expect(entryTexts(lines).map(([tone]) => tone)).toEqual([
      "warning",
      "warning",
      "stderr",
      "error",
      "error",
    ]);
  });

  it("keeps only matching entries under their step names", () => {
    const lines = toBuildLogLines(
      [
        entry("Cloning into '/workspace'\n", { step: "Clone", stepId: "step_clone" }),
        entry("Packages: +1412\n"),
        entry("Done in 16.4s\n"),
      ],
      new Set(),
      "  PACKAGES ",
    );

    expect(lines).toEqual([
      { kind: "step", step: "Install dependencies" },
      expect.objectContaining({ kind: "entry", text: "Packages: +1412\n" }),
    ]);
  });

  it("does not fold a long step while searching", () => {
    const entries = Array.from({ length: 60 }, (_, index) => entry(`line ${index}\n`));
    const withLaterStep = [...entries, entry("next\n", { step: "Build", stepId: "step_build" })];

    expect(toBuildLogLines(withLaterStep, new Set()).some((line) => line.kind === "fold")).toBe(
      true,
    );
    expect(
      toBuildLogLines(withLaterStep, new Set(), "line").some((line) => line.kind === "fold"),
    ).toBe(false);
  });
});

describe("toBuildLogText", () => {
  it("prints every entry under its step name without ANSI codes", () => {
    const entries = [
      ...Array.from({ length: 60 }, (_, index) => entry(`line ${index}\n`)),
      entry("\x1b[31mfailed\x1b[0m\n", { step: "Build", stepId: "step_build" }),
    ];
    const text = toBuildLogText(toBuildLogLines(entries, "all"));

    expect(text.split("\n")).toHaveLength(63);
    expect(text.startsWith("Install dependencies\nline 0\n")).toBe(true);
    expect(text.endsWith("line 59\nBuild\nfailed")).toBe(true);
  });
});
