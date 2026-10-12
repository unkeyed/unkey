import { describe, expect, it } from "vitest";
import type { BuildLogLine } from "./build-log-lines";
import {
  type HeaderLabel,
  type ScrollToLatest,
  type TailRow,
  buildLogsHeaderState,
  headerLabelText,
  scrollToLatest,
  tailRow,
  toToneIndices,
} from "./build-logs-header-state";

const base = {
  toneIndices: { error: [], warning: [] },
  cursor: null,
  pendingJump: null,
  hasMore: false,
  isFailed: false,
  entriesTotal: 44,
  entriesShownMax: 5000,
};

function entryLine(tone: "stdout" | "warning" | "error"): BuildLogLine {
  return {
    kind: "entry",
    tone,
    text: "x\n",
    entry: { message: "x\n", output: "stdout", step: "Build", stepId: "step_build", time: 1 },
  };
}

describe("toToneIndices", () => {
  it("lists the line indices of each tone", () => {
    const lines: BuildLogLine[] = [
      { kind: "step", step: "Build" },
      entryLine("warning"),
      entryLine("stdout"),
      entryLine("error"),
      entryLine("warning"),
    ];

    expect(toToneIndices(lines)).toEqual({ error: [3], warning: [1, 4] });
  });
});

describe("buildLogsHeaderState", () => {
  it("shows the line count and no counters for a clean build", () => {
    expect(buildLogsHeaderState(base)).toEqual({
      label: { type: "count", total: 44, hasMore: false },
      counters: [],
    });
  });

  it("hides a tone with no lines and keeps the others idle", () => {
    const state = buildLogsHeaderState({ ...base, toneIndices: { error: [], warning: [2, 5] } });

    expect(state.counters).toEqual([{ type: "idle", tone: "warning", count: 2, hasMore: false }]);
  });

  it("shows the error counter of a failed build before its error line loads", () => {
    const state = buildLogsHeaderState({ ...base, isFailed: true, hasMore: true });

    expect(state.counters).toEqual([{ type: "idle", tone: "error", count: 0, hasMore: true }]);
  });

  it("marks the counter waiting for the rest of the logs as pending", () => {
    const state = buildLogsHeaderState({
      ...base,
      toneIndices: { error: [], warning: [1] },
      pendingJump: "warning",
      hasMore: true,
    });

    expect(state.counters).toEqual([{ type: "pending", tone: "warning", count: 1, hasMore: true }]);
  });

  it("names the selected tone in the label and gives its counter the position", () => {
    const state = buildLogsHeaderState({
      ...base,
      toneIndices: { error: [9], warning: [1, 4] },
      cursor: { tone: "warning", position: 1 },
    });

    expect(state).toEqual({
      label: { type: "tone", tone: "warning", lines: 2 },
      counters: [
        { type: "idle", tone: "error", count: 1, hasMore: false },
        { type: "active", tone: "warning", count: 2, hasMore: false, position: 1 },
      ],
    });
  });

  it("says the list is cut when more lines exist than are shown", () => {
    const state = buildLogsHeaderState({ ...base, entriesTotal: 7200 });

    expect(state.label).toEqual({ type: "truncated", shown: 5000, total: 7200 });
  });
});

describe("headerLabelText", () => {
  it.each([
    {
      name: "one line of a tone",
      label: { type: "tone", tone: "error", lines: 1 },
      text: "1 error line",
    },
    {
      name: "several lines of a tone",
      label: { type: "tone", tone: "warning", lines: 2 },
      text: "2 warning lines",
    },
    {
      name: "a list cut to the newest lines",
      label: { type: "truncated", shown: 5000, total: 7200 },
      text: "Showing the last 5,000 of 7,200 lines",
    },
    { name: "a single line", label: { type: "count", total: 1, hasMore: false }, text: "1 line" },
    {
      name: "a count with more to load",
      label: { type: "count", total: 1, hasMore: true },
      text: "1+ lines",
    },
    { name: "a full count", label: { type: "count", total: 44, hasMore: false }, text: "44 lines" },
  ] satisfies { name: string; label: HeaderLabel; text: string }[])(
    "writes $name",
    ({ label, text }) => {
      expect(headerLabelText(label)).toBe(text);
    },
  );
});

describe("tailRow", () => {
  it.each([
    {
      name: "loading while more lines exist",
      hasMore: true,
      isBuilding: true,
      tail: { type: "loading-more" },
    },
    {
      name: "running once caught up on a live build",
      hasMore: false,
      isBuilding: true,
      tail: { type: "running" },
    },
    {
      name: "nothing for a finished build",
      hasMore: false,
      isBuilding: false,
      tail: { type: "none" },
    },
  ] satisfies { name: string; hasMore: boolean; isBuilding: boolean; tail: TailRow }[])(
    "shows $name",
    ({ hasMore, isBuilding, tail }) => {
      expect(tailRow({ hasMore, isBuilding })).toEqual(tail);
    },
  );
});

describe("scrollToLatest", () => {
  const settled = { followsTail: false, hasMore: false, isAtBottom: false, pendingJump: null };

  it.each([
    {
      name: "hidden while following the tail",
      input: { ...settled, followsTail: true },
      state: { type: "hidden" },
    },
    {
      name: "hidden at the end of a fully loaded log",
      input: { ...settled, isAtBottom: true },
      state: { type: "hidden" },
    },
    { name: "ready above the end", input: settled, state: { type: "ready" } },
    {
      name: "ready at the bottom while more lines exist",
      input: { ...settled, isAtBottom: true, hasMore: true },
      state: { type: "ready" },
    },
    {
      name: "pending while the rest of the log loads",
      input: { ...settled, hasMore: true, pendingJump: "latest" as const },
      state: { type: "pending" },
    },
  ] satisfies {
    name: string;
    input: Parameters<typeof scrollToLatest>[0];
    state: ScrollToLatest;
  }[])("is $name", ({ input, state }) => {
    expect(scrollToLatest(input)).toEqual(state);
  });
});
