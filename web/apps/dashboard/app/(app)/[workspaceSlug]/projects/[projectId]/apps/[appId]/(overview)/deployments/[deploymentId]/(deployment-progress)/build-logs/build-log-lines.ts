import type { BuildLogEntry } from "@unkey/api/models/components";

// printed is the output without ANSI codes, before carriage returns redraw it.
// text is how a terminal shows printed
type EntryLine = {
  kind: "entry";
  entry: BuildLogEntry;
  printed: string;
  text: string;
  tone: "stdout" | "stderr" | "error";
  isEvent: boolean;
};

export type BuildLogLine =
  | { kind: "step"; step: string }
  | EntryLine
  | { kind: "fold"; runKey: string; entriesHidden: number };

// ctrl writes "ERROR: <reason>" to stderr when a build step fails
const STEP_ERROR_PREFIX = "ERROR: ";

// CSI (colors, cursor moves, erase), OSC (titles, links), and two-byte escapes
// biome-ignore lint/suspicious/noControlCharactersInRegex: matching escape sequences is the point
const ANSI_ESCAPE = /\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/g;

// The lines ctrl adds when a step ends, see processBuildStatus in build.go
const STEP_DONE_LINE = /^DONE \d+\.\ds$/;

export const FOLD_ENTRIES_MIN = 40;
export const FOLD_EDGE_ENTRIES = 5;

// A run is the entries one step printed between two other steps. A long run
// folds its middle unless it holds an error, the viewer expanded it, or it is
// the last run, which can still grow
export function toBuildLogLines(
  entries: BuildLogEntry[],
  expandedRunKeys: ReadonlySet<string>,
): BuildLogLine[] {
  const runs: EntryLine[][] = [];
  for (const entry of entries) {
    const isStepError = entry.output === "stderr" && entry.message.startsWith(STEP_ERROR_PREFIX);
    const isEvent =
      isStepError ||
      (entry.output === "stdout" &&
        (entry.message === "CACHED" || STEP_DONE_LINE.test(entry.message)));
    const run = runs.at(-1);
    const isSameStep = run !== undefined && run[0].entry.stepId === entry.stepId;
    const previous = isSameStep ? run.at(-1) : undefined;
    // BuildKit cuts output into chunks anywhere, so a chunk after an unfinished
    // line of the same stream continues that line
    const continuesPrevious =
      previous !== undefined &&
      !previous.isEvent &&
      !isEvent &&
      previous.entry.output === entry.output &&
      !previous.printed.endsWith("\n");
    const printed =
      (continuesPrevious ? previous.printed : "") + entry.message.replace(ANSI_ESCAPE, "");
    // A carriage return redraws its line, so only the text after the last one
    // stays. A CRLF line ending is not a redraw
    const text = printed
      .split("\n")
      .map((terminalLine) => {
        const withoutLineEnd = terminalLine.replace(/\r+$/, "");
        return withoutLineEnd.slice(withoutLineEnd.lastIndexOf("\r") + 1);
      })
      .join("\n");
    if (run && continuesPrevious) {
      run[run.length - 1] = { ...previous, printed, text };
      continue;
    }
    const line: EntryLine = {
      kind: "entry",
      entry,
      printed,
      text,
      tone: isStepError ? "error" : entry.output,
      isEvent,
    };
    if (isSameStep) {
      run.push(line);
    } else {
      runs.push([line]);
    }
  }

  const lines: BuildLogLine[] = [];
  runs.forEach((run, runIndex) => {
    const { stepId, step, time } = run[0].entry;
    lines.push({ kind: "step", step });
    const runKey = `${stepId}:${time}`;
    const folds =
      run.length > FOLD_ENTRIES_MIN &&
      runIndex < runs.length - 1 &&
      !expandedRunKeys.has(runKey) &&
      !run.some((line) => line.tone === "error");
    if (!folds) {
      lines.push(...run);
      return;
    }
    lines.push(
      ...run.slice(0, FOLD_EDGE_ENTRIES),
      { kind: "fold", runKey, entriesHidden: run.length - 2 * FOLD_EDGE_ENTRIES },
      ...run.slice(-FOLD_EDGE_ENTRIES),
    );
  });
  return lines;
}
