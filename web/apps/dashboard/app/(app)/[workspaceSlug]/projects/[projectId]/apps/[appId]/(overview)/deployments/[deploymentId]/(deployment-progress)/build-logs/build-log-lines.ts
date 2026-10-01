import type { BuildLogEntry } from "@unkey/api/models/components";
import { match } from "@unkey/match";

type Tone = "stdout" | "stderr" | "error" | "event";

export type BuildLogLine =
  | { kind: "step"; step: string }
  | { kind: "entry"; entry: BuildLogEntry; text: string; tone: Tone }
  | { kind: "fold"; runKey: string; entriesHidden: number };

// ctrl writes "ERROR: <reason>" to stderr when a build step fails
const STEP_ERROR_PREFIX = "ERROR: ";

// The lines ctrl adds when a step ends, see processBuildStatus in build.go
const STEP_DONE_LINE = /^DONE \d+\.\ds$/;

// CSI (colors, cursor moves, erase), OSC (titles, links), and two-byte escapes
// biome-ignore lint/suspicious/noControlCharactersInRegex: matching escape sequences is the point
const ANSI_ESCAPE = /\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/g;

export const FOLD_ENTRIES_MIN = 40;
export const FOLD_EDGE_ENTRIES = 5;

// A run is the entries one step printed between two other steps. A long run
// folds its middle unless it holds an error, the viewer expanded it, or it is
// the last run, which can still grow
export function toBuildLogLines(
  entries: BuildLogEntry[],
  expandedRunKeys: ReadonlySet<string>,
): BuildLogLine[] {
  const runs: Run[] = [];
  for (const entry of entries) {
    const tone = match(entry)
      .returnType<Tone>()
      .with(
        { output: "stderr" },
        ({ message }) => message.startsWith(STEP_ERROR_PREFIX),
        () => "error",
      )
      .with(
        { output: "stdout" },
        ({ message }) => message === "CACHED" || STEP_DONE_LINE.test(message),
        () => "event",
      )
      .otherwise(({ output }) => output);

    const printed = entry.message.replace(ANSI_ESCAPE, "");
    const run = runs.at(-1);
    const last = run?.at(-1);

    if (!run || run[0].entry.stepId !== entry.stepId) {
      runs.push([{ entry, printed, tone }]);
    } else if (
      // BuildKit cuts output into chunks anywhere, so a chunk after an
      // unfinished line of the same stream continues that line
      last &&
      last.tone === tone &&
      (tone === "stdout" || tone === "stderr") &&
      !last.printed.endsWith("\n")
    ) {
      last.printed += printed;
    } else {
      run.push({ entry, printed, tone });
    }
  }

  const lines: BuildLogLine[] = [];
  runs.forEach((run, runIndex) => {
    const { stepId, step, time } = run[0].entry;
    const runKey = `${stepId}:${time}`;
    const entryLines = run.map(
      ({ entry, printed, tone }): BuildLogLine => ({
        kind: "entry",
        entry,
        tone,
        text: terminalText(printed),
      }),
    );
    const folds =
      run.length > FOLD_ENTRIES_MIN &&
      runIndex < runs.length - 1 &&
      !expandedRunKeys.has(runKey) &&
      !run.some((row) => row.tone === "error");
    lines.push({ kind: "step", step });
    if (folds) {
      lines.push(
        ...entryLines.slice(0, FOLD_EDGE_ENTRIES),
        { kind: "fold", runKey, entriesHidden: run.length - 2 * FOLD_EDGE_ENTRIES },
        ...entryLines.slice(-FOLD_EDGE_ENTRIES),
      );
    } else {
      lines.push(...entryLines);
    }
  });
  return lines;
}

// printed is the output without ANSI codes, before carriage returns redraw it
type Run = { entry: BuildLogEntry; printed: string; tone: Tone }[];

// A carriage return redraws its line, so only the text after the last one
// stays. A CRLF line ending is not a redraw
function terminalText(printed: string): string {
  return printed
    .split("\n")
    .map((line) => {
      const withoutLineEnd = line.replace(/\r+$/, "");
      return withoutLineEnd.slice(withoutLineEnd.lastIndexOf("\r") + 1);
    })
    .join("\n");
}
