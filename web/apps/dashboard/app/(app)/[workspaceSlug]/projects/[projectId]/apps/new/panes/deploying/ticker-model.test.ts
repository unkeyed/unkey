import { describe, expect, it } from "vitest";
import type { LogGroup, Stage } from "./run-model";
import {
  type TickerEvent,
  type TickerLine,
  emptyTicker,
  stageTickerLine,
  tickerReducer,
} from "./ticker-model";

const line = (id: string): TickerLine => ({ id, text: `line ${id}`, tone: "plain" });
const receive = (id: string, animate = true): TickerEvent => ({
  type: "receive",
  line: line(id),
  animate,
});
const run = (events: TickerEvent[]) => events.reduce(tickerReducer, emptyTicker);
const ids = (state: ReturnType<typeof run>) => ({
  shown: state.shown?.id ?? null,
  incoming: state.incoming?.id ?? null,
  queued: state.queued?.id ?? null,
});
const clear: TickerEvent = { type: "receive", line: null, animate: true };

describe("tickerReducer", () => {
  it("shows the first line without a swap", () => {
    expect(ids(run([receive("a")]))).toEqual({ shown: "a", incoming: null, queued: null });
  });

  it("starts one swap for a new line", () => {
    expect(ids(run([receive("a"), receive("b")]))).toEqual({
      shown: "a",
      incoming: "b",
      queued: null,
    });
  });

  it("keeps only the newest line while a swap runs", () => {
    expect(
      ids(run([receive("a"), receive("b"), receive("c"), receive("d"), receive("e")])),
    ).toEqual({ shown: "a", incoming: "b", queued: "e" });
  });

  it("jumps to the newest queued line when the swap ends", () => {
    const events = [receive("a"), receive("b"), receive("c"), receive("d")];
    expect(ids(run([...events, { type: "settle" }]))).toEqual({
      shown: "b",
      incoming: "d",
      queued: null,
    });
    expect(ids(run([...events, { type: "settle" }, { type: "settle" }]))).toEqual({
      shown: "d",
      incoming: null,
      queued: null,
    });
  });

  it("ignores a repeat of the line it is heading to", () => {
    expect(ids(run([receive("a"), receive("b"), receive("b")]))).toEqual({
      shown: "a",
      incoming: "b",
      queued: null,
    });
  });

  it("swaps in place when motion is reduced", () => {
    expect(ids(run([receive("a", false), receive("b", false), receive("c", false)]))).toEqual({
      shown: "c",
      incoming: null,
      queued: null,
    });
  });

  it("fades the line out before clearing it", () => {
    const leaving = run([receive("a"), clear]);
    expect(leaving.leaving).toBe(true);
    expect(leaving.shown?.id).toBe("a");
    expect(tickerReducer(leaving, { type: "settle" })).toEqual(emptyTicker);
  });

  it("shows a new line straight away while fading out", () => {
    expect(ids(run([receive("a"), clear, receive("b")]))).toEqual({
      shown: "b",
      incoming: null,
      queued: null,
    });
  });
});

const stage = (key: Stage["key"], state: Stage["state"], error: string | null = null): Stage => ({
  key,
  label: key,
  state,
  durationMs: null,
  note: null,
  error,
});

const groups: LogGroup[] = [
  {
    id: "0-build",
    title: "build",
    failed: false,
    lines: [{ id: "b1", offsetMs: 0, text: "exporting to image", tone: "plain" }],
  },
  {
    id: "runtime",
    title: "Runtime logs",
    failed: false,
    lines: [{ id: "r1", offsetMs: 0, text: "listening on :8080", tone: "plain" }],
  },
];

describe("stageTickerLine", () => {
  it("shows build lines only while building", () => {
    expect(stageTickerLine(stage("building", "active"), groups, null)?.text).toBe(
      "exporting to image",
    );
    expect(stageTickerLine(stage("building", "done"), groups, null)).toBeNull();
  });

  it("shows runtime lines while instances start", () => {
    expect(stageTickerLine(stage("deploying", "active"), groups, null)?.text).toBe(
      "listening on :8080",
    );
    expect(stageTickerLine(stage("deploying", "active"), groups.slice(0, 1), null)?.text).toBe(
      "Scheduling instances…",
    );
  });

  it("shows nothing for stages without their own logs", () => {
    expect(stageTickerLine(stage("network", "active"), groups, null)).toBeNull();
  });

  it("keeps the error on a failed stage", () => {
    expect(stageTickerLine(stage("building", "failed", "exit 1"), groups, null)).toEqual({
      id: "building-error",
      text: "exit 1",
      tone: "error",
    });
    expect(stageTickerLine(stage("deploying", "failed"), groups, "oom")?.text).toBe("oom");
  });
});

describe("stageTickerLine before real data arrives", () => {
  it("shows a waiting line while the build has no steps yet", () => {
    expect(stageTickerLine(stage("building", "active"), [], null)).toEqual({
      id: "building-waiting",
      text: "Preparing the build…",
      tone: "plain",
      waiting: true,
    });
  });

  it("rolls into the first real build step when it arrives", () => {
    const first: LogGroup = {
      id: "0-load",
      title: "[internal] load build definition from Dockerfile",
      failed: false,
      lines: [],
    };
    expect(stageTickerLine(stage("building", "active"), [first], null)?.text).toBe(
      "[internal] load build definition from Dockerfile",
    );
  });

  it("shows a waiting line while instances start without runtime logs", () => {
    expect(stageTickerLine(stage("deploying", "active"), [], null)?.text).toBe(
      "Scheduling instances…",
    );
  });
});
