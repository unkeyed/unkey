import type { BuildLogEntry } from "@unkey/api/models/components";
import { describe, expect, it } from "vitest";
import { toBuildLogLines } from "../(deployment-progress)/build-logs/build-log-lines";
import type { Stage, StageKey } from "./stages-model";
import {
  type TickerEvent,
  type TickerLine,
  type TickerSources,
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

  it("fades out the incoming line when cleared mid-swap", () => {
    const leaving = run([receive("a"), receive("b"), clear]);
    expect(leaving.leaving).toBe(true);
    expect(ids(leaving)).toEqual({ shown: "b", incoming: null, queued: null });
  });

  it("shows a new line straight away while fading out", () => {
    expect(ids(run([receive("a"), clear, receive("b")]))).toEqual({
      shown: "b",
      incoming: null,
      queued: null,
    });
  });
});

const stage = (
  key: StageKey,
  status: Stage["resolution"]["status"],
  description = "",
): Pick<Stage, "key" | "resolution"> => ({
  key,
  resolution: { status, description, duration: undefined },
});

const entry = (
  message: string,
  { step = "build", output = "stdout", time = 1 }: Partial<BuildLogEntry> = {},
): BuildLogEntry => ({ time, stepId: step, step, output, message: `${message}\n` });

const sources = (overrides: Partial<TickerSources> = {}): TickerSources => ({
  buildLines: toBuildLogLines(
    [
      entry("resolve docker.io/library/node:22", { step: "[internal] load metadata" }),
      entry("#14 writing image sha256:6f1c2b9e", { step: "exporting to image", time: 2 }),
    ],
    "all",
  ),
  buildLogsFailed: false,
  runtimeLogs: [
    { time: 5, severity: "INFO", message: "listening on :8080", instance_id: "i1", region: "r" },
    { time: 3, severity: "INFO", message: "booting", instance_id: "i1", region: "r" },
  ],
  instances: null,
  ...overrides,
});

describe("stageTickerLine", () => {
  it("shows the current build step while building", () => {
    expect(stageTickerLine(stage("build", "started"), sources())?.text).toBe("exporting to image");
    expect(stageTickerLine(stage("build", "completed"), sources())).toBeNull();
  });

  it("names a cached step instead of printing CACHED", () => {
    const buildLines = toBuildLogLines([entry("CACHED", { step: "[2/4] RUN pnpm i" })], "all");
    expect(stageTickerLine(stage("build", "started"), sources({ buildLines }))?.text).toBe(
      "[2/4] RUN pnpm i (cached)",
    );
  });

  it("skips a CACHED line from a step without a name", () => {
    const buildLines = toBuildLogLines(
      [entry("added 312 packages", { step: "RUN pnpm i" }), entry("CACHED", { step: "", time: 2 })],
      "all",
    );
    expect(stageTickerLine(stage("build", "started"), sources({ buildLines }))?.text).toBe(
      "RUN pnpm i",
    );
  });

  it("shows a build error line as it is", () => {
    const buildLines = toBuildLogLines(
      [
        entry("added 312 packages", { step: "RUN pnpm i" }),
        entry("ERROR: process did not complete", {
          step: "RUN pnpm build",
          output: "stderr",
          time: 2,
        }),
      ],
      "all",
    );
    expect(stageTickerLine(stage("build", "started"), sources({ buildLines }))).toMatchObject({
      text: "ERROR: process did not complete",
      tone: "error",
    });
  });

  it("shows the newest runtime line while instances start", () => {
    expect(stageTickerLine(stage("instances", "started"), sources())?.text).toBe(
      "listening on :8080",
    );
  });

  it("reads WARN and WARNING runtime lines as warnings", () => {
    for (const severity of ["WARN", "WARNING"]) {
      const runtimeLogs = [{ time: 9, severity, message: "slow", instance_id: "i1", region: "r" }];
      expect(stageTickerLine(stage("instances", "started"), sources({ runtimeLogs }))?.tone).toBe(
        "warn",
      );
    }
  });

  it("falls back to the instance count, then a waiting line", () => {
    const instances = { state: "starting" as const, running: 1, total: 3 };
    expect(
      stageTickerLine(stage("instances", "started"), sources({ runtimeLogs: [], instances }))?.text,
    ).toBe("1 of 3 running");
    expect(stageTickerLine(stage("instances", "started"), sources({ runtimeLogs: [] }))).toEqual({
      id: "instances-waiting",
      text: "Scheduling instances…",
      tone: "plain",
      waiting: true,
    });
  });

  it("shows nothing for stages without their own logs", () => {
    expect(stageTickerLine(stage("domains", "started"), sources())).toBeNull();
  });

  it("shows the stage error on a failed stage", () => {
    expect(stageTickerLine(stage("build", "error", "exit 1"), sources())).toEqual({
      id: "build-error",
      text: "exit 1",
      tone: "error",
    });
  });

  it("shows a waiting line while the build has no output yet", () => {
    expect(stageTickerLine(stage("build", "started"), sources({ buildLines: [] }))).toEqual({
      id: "build-waiting",
      text: "Preparing the build…",
      tone: "plain",
      waiting: true,
    });
  });

  it("falls back to the stage description when the build logs fail to load", () => {
    expect(
      stageTickerLine(
        stage("build", "started"),
        sources({ buildLines: [], buildLogsFailed: true }),
      ),
    ).toBe(null);
  });
});
