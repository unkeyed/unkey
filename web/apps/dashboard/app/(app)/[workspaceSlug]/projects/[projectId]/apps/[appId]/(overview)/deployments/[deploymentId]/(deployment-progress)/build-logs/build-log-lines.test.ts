import type { BuildLogEntry } from "@unkey/api/models/components";
import { describe, expect, test } from "vitest";
import {
  type BuildLogLine,
  FOLD_EDGE_ENTRIES,
  FOLD_ENTRIES_MIN,
  toBuildLogLines,
} from "./build-log-lines";

const startedAt = Date.now();

const entry = (
  stepId: string,
  message: string,
  output: BuildLogEntry["output"] = "stdout",
  time = startedAt,
): BuildLogEntry => ({ time, stepId, step: `step ${stepId}`, output, message });

const burst = (stepId: string, count: number) =>
  Array.from({ length: count }, (_, i) => entry(stepId, `KEBAP ${i}\n`, "stdout", startedAt + i));

const describeLine = (line: BuildLogLine) => {
  switch (line.kind) {
    case "step":
      return `# ${line.step}`;
    case "fold":
      return `... ${line.entriesHidden}`;
    case "entry":
      return line.text.trimEnd();
  }
};

const noneExpanded = new Set<string>();

describe("toBuildLogLines", () => {
  test("no entries yields no lines", () => {
    expect(toBuildLogLines([], noneExpanded)).toEqual([]);
  });

  test("a step header precedes each run of entries from one step", () => {
    const install = crypto.randomUUID();
    const npmTest = crypto.randomUUID();
    const entries = [
      entry(install, "KEBAP 1\n"),
      entry(install, "KEBAP 2\n"),
      entry(npmTest, "KEBAP 3\n"),
      entry(install, "DONE 5.3s"),
    ];

    expect(toBuildLogLines(entries, noneExpanded).map(describeLine)).toEqual([
      `# step ${install}`,
      "KEBAP 1",
      "KEBAP 2",
      `# step ${npmTest}`,
      "KEBAP 3",
      `# step ${install}`,
      "DONE 5.3s",
    ]);
  });

  test("only a stderr entry with the step error prefix is an error", () => {
    const stepId = crypto.randomUUID();
    const entries = [
      entry(stepId, "KEBAP out\n"),
      entry(stepId, "ERROR: KEBAP printed to stdout\n"),
      entry(stepId, "KEBAP warning\n", "stderr"),
      entry(stepId, "ERROR: KEBAP exit code: 1", "stderr"),
    ];

    expect(
      toBuildLogLines(entries, noneExpanded).flatMap((line) =>
        line.kind === "entry" ? [line.tone] : [],
      ),
    ).toEqual(["stdout", "stdout", "stderr", "error"]);
  });

  test("a long finished run folds its middle", () => {
    const noisy = crypto.randomUUID();
    const next = crypto.randomUUID();
    const count = FOLD_ENTRIES_MIN + 1;
    const lines = toBuildLogLines([...burst(noisy, count), entry(next, "DONE 1.0s")], noneExpanded);

    expect(lines.map(describeLine)).toEqual([
      `# step ${noisy}`,
      ...Array.from({ length: FOLD_EDGE_ENTRIES }, (_, i) => `KEBAP ${i}`),
      `... ${count - 2 * FOLD_EDGE_ENTRIES}`,
      ...Array.from(
        { length: FOLD_EDGE_ENTRIES },
        (_, i) => `KEBAP ${count - FOLD_EDGE_ENTRIES + i}`,
      ),
      `# step ${next}`,
      "DONE 1.0s",
    ]);
  });

  test("a run at the folding threshold, the last run, an expanded run, and a run with an error stay open", () => {
    const short = crypto.randomUUID();
    const expanded = crypto.randomUUID();
    const failed = crypto.randomUUID();
    const last = crypto.randomUUID();
    const entries = [
      ...burst(short, FOLD_ENTRIES_MIN),
      ...burst(expanded, FOLD_ENTRIES_MIN + 1),
      ...burst(failed, FOLD_ENTRIES_MIN + 1),
      entry(failed, "ERROR: KEBAP exit code: 1", "stderr"),
      ...burst(last, FOLD_ENTRIES_MIN + 1),
    ];

    const lines = toBuildLogLines(entries, new Set([`${expanded}:${startedAt}`]));

    expect(lines.filter((line) => line.kind === "fold")).toEqual([]);
    expect(lines.filter((line) => line.kind === "entry")).toHaveLength(entries.length);
  });

  test.each([
    {
      name: "strips ANSI colors",
      message: "\u001b[32m✓\u001b[0m KEBAP built\n",
      text: "✓ KEBAP built\n",
    },
    {
      name: "strips cursor and erase codes",
      message: "\u001b[2K\u001b[1GKEBAP step\n",
      text: "KEBAP step\n",
    },
    {
      name: "keeps the last redraw of a carriage return progress line",
      message: "KEBAP 10%\rKEBAP 50%\rKEBAP 100%\n",
      text: "KEBAP 100%\n",
    },
    { name: "keeps a CRLF line", message: "KEBAP line\r\n", text: "KEBAP line\n" },
    {
      name: "redraws each line on its own",
      message: "a\rKEBAP 1\nb\rKEBAP 2\n",
      text: "KEBAP 1\nKEBAP 2\n",
    },
    { name: "leaves plain text alone", message: "KEBAP\tplain\n", text: "KEBAP\tplain\n" },
  ])("$name", ({ message, text }) => {
    const [, line] = toBuildLogLines([entry(crypto.randomUUID(), message)], noneExpanded);
    expect(line.kind === "entry" && line.text).toBe(text);
  });

  describe("an entry that continues an open line of the same step", () => {
    const texts = (entries: BuildLogEntry[]) =>
      toBuildLogLines(entries, noneExpanded).flatMap((line) =>
        line.kind === "entry" ? [line.text] : [],
      );

    test.each([
      {
        name: "a progress bar redrawn across entries keeps its last state",
        entries: (id: string) => [
          entry(id, "\rKEBAP 10%"),
          entry(id, "\rKEBAP 50%"),
          entry(id, "\rKEBAP 100%"),
          entry(id, "\n"),
        ],
        texts: ["KEBAP 100%\n"],
      },
      {
        name: "a line split across entries joins",
        entries: (id: string) => [entry(id, "KEBAP hal"), entry(id, "f line\n")],
        texts: ["KEBAP half line\n"],
      },
      {
        name: "a finished line does not join",
        entries: (id: string) => [entry(id, "KEBAP one\n"), entry(id, "KEBAP two\n")],
        texts: ["KEBAP one\n", "KEBAP two\n"],
      },
      {
        name: "a blank line after a finished line stays",
        entries: (id: string) => [entry(id, "KEBAP one\n"), entry(id, "\n")],
        texts: ["KEBAP one\n", "\n"],
      },
      {
        name: "a ctrl event line never joins",
        entries: (id: string) => [entry(id, "\rKEBAP 100%"), entry(id, "DONE 3.1s")],
        texts: ["KEBAP 100%", "DONE 3.1s"],
      },
      {
        name: "stdout and stderr do not join",
        entries: (id: string) => [entry(id, "KEBAP out"), entry(id, "KEBAP err\n", "stderr")],
        texts: ["KEBAP out", "KEBAP err\n"],
      },
      {
        name: "a step error never joins",
        entries: (id: string) => [
          entry(id, "KEBAP fails", "stderr"),
          entry(id, "ERROR: KEBAP exit code: 1", "stderr"),
        ],
        texts: ["KEBAP fails", "ERROR: KEBAP exit code: 1"],
      },
    ])("$name", ({ entries, texts: expected }) => {
      expect(texts(entries(crypto.randomUUID()))).toEqual(expected);
    });

    test("entries of different steps do not join", () => {
      const first = crypto.randomUUID();
      const second = crypto.randomUUID();
      expect(texts([entry(first, "KEBAP open"), entry(second, "KEBAP other\n")])).toEqual([
        "KEBAP open",
        "KEBAP other\n",
      ]);
    });
  });
});
