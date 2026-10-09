import { describe, expect, it } from "vitest";
import {
  type LeaveEvent,
  type LeavePhase,
  type LeaveStep,
  SENTINEL_STATE,
  isLeaveSentinel,
  nextLeaveStep,
} from "./prevent-leave-model";

function run(events: LeaveEvent[], phase: LeavePhase = "idle"): LeaveStep[] {
  const steps: LeaveStep[] = [];
  let current = phase;
  for (const event of events) {
    const step = nextLeaveStep(current, event);
    steps.push(step);
    current = step.phase;
  }
  return steps;
}

describe("nextLeaveStep", () => {
  it("pushes a sentinel when the guard turns on, unless one is already on top", () => {
    expect(nextLeaveStep("idle", { type: "guardOn", sentinelOnTop: false })).toEqual({
      phase: "idle",
      effects: ["pushSentinel"],
    });
    expect(nextLeaveStep("idle", { type: "guardOn", sentinelOnTop: true }).effects).toEqual([]);
  });

  it("goes back over its sentinel when the guard turns off", () => {
    expect(nextLeaveStep("idle", { type: "guardOff", sentinelOnTop: true })).toEqual({
      phase: "poppingSentinel",
      effects: ["disarmFallback", "historyBack"],
    });
  });

  it("asks on the first Back after the guard turns off and on again", () => {
    const steps = run([
      { type: "guardOn", sentinelOnTop: false },
      { type: "guardOff", sentinelOnTop: true },
      { type: "guardOn", sentinelOnTop: true },
      { type: "popstate", sentinel: false },
      { type: "popstate", sentinel: false },
    ]);
    expect(steps[3]).toEqual({ phase: "idle", effects: ["pushSentinel"] });
    expect(steps[4].effects).toEqual(["askBack"]);
  });

  it("asks on Back and closes the prompt on Forward to the sentinel", () => {
    expect(nextLeaveStep("idle", { type: "popstate", sentinel: false }).effects).toEqual([
      "askBack",
    ]);
    expect(nextLeaveStep("idle", { type: "popstate", sentinel: true }).effects).toEqual([
      "closePrompt",
    ]);
  });

  it("blocks reload and tab close until the user chooses to leave", () => {
    expect(nextLeaveStep("idle", { type: "beforeunload" }).effects).toEqual(["blockUnload"]);
    const [, unload] = run([{ type: "leave" }, { type: "beforeunload" }]);
    expect(unload.effects).toEqual([]);
  });

  it("lets a bypassed navigation through without asking", () => {
    const [, back] = run([{ type: "leave" }, { type: "popstate", sentinel: false }]);
    expect(back).toEqual({ phase: "idle", effects: ["disarmFallback"] });
  });

  it("keeps its sentinel when the guard turns off while leaving", () => {
    const [, off] = run([{ type: "leave" }, { type: "guardOff", sentinelOnTop: true }]);
    expect(off).toEqual({ phase: "idle", effects: ["disarmFallback"] });
  });

  it("goes back on discard and falls back when Back has nowhere to go", () => {
    const [discard, elapsed] = run([{ type: "leaveBack" }, { type: "fallbackElapsed" }]);
    expect(discard).toEqual({ phase: "leaving", effects: ["armFallback", "historyBack"] });
    expect(elapsed).toEqual({ phase: "idle", effects: ["goToFallback"] });
  });

  it("drops the fallback when Back lands or the page unloads", () => {
    expect(nextLeaveStep("leaving", { type: "popstate", sentinel: false }).effects).toEqual([
      "disarmFallback",
    ]);
    expect(nextLeaveStep("leaving", { type: "pagehide" })).toEqual({
      phase: "idle",
      effects: ["disarmFallback"],
    });
  });

  it("restores the sentinel when the user keeps editing after Back", () => {
    expect(nextLeaveStep("idle", { type: "keepEditing", sentinelOnTop: false }).effects).toEqual([
      "closePrompt",
      "pushSentinel",
    ]);
    expect(nextLeaveStep("idle", { type: "keepEditing", sentinelOnTop: true }).effects).toEqual([
      "closePrompt",
    ]);
  });
});

describe("isLeaveSentinel", () => {
  it("recognizes only the guard's history state", () => {
    expect(isLeaveSentinel(SENTINEL_STATE)).toBe(true);
    expect(isLeaveSentinel(null)).toBe(false);
    expect(isLeaveSentinel({ __NA: true })).toBe(false);
  });
});
