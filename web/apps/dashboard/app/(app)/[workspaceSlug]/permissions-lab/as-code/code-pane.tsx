"use client";

/**
 * The text half of the two-way editor: one canonical permission string per
 * line. Edits are parsed live; a valid draft becomes an add/remove diff
 * against the current direct grants that the caller can apply as one commit.
 */

import { Button, Textarea } from "@unkey/ui";
import { WORKSPACE_ID } from "../lib/mock-data";
import { formatPermission, parsePermission } from "../lib/urn";

export interface LineError {
  line: number;
  message: string;
}

export interface DraftAnalysis {
  errors: LineError[];
  /** valid lines not present in the current direct grants */
  added: string[];
  /** current direct grants missing from the draft */
  removed: string[];
}

/**
 * Parses every non-empty line and diffs the valid set against the current
 * direct grants. Lines with errors contribute nothing to the parsed set, so
 * the diff is only trustworthy when `errors` is empty; callers gate Apply on
 * that.
 */
export function analyzeDraft(text: string, current: string[]): DraftAnalysis {
  const errors: LineError[] = [];
  const parsed = new Set<string>();

  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "") {
      continue;
    }
    const result = parsePermission(line);
    if (!result.ok) {
      errors.push({ line: i + 1, message: result.error });
      continue;
    }
    if (result.value.urn.workspaceID !== WORKSPACE_ID) {
      errors.push({
        line: i + 1,
        message: `workspace must be "${WORKSPACE_ID}" in this workspace`,
      });
      continue;
    }
    parsed.add(formatPermission(result.value));
  }

  const currentSet = new Set(current);
  const added = [...parsed].filter((p) => !currentSet.has(p)).sort();
  const removed = current.filter((p) => !parsed.has(p)).sort();
  return { errors, added, removed };
}

export function CodePane({
  text,
  dirty,
  conflict,
  analysis,
  onChange,
  onApply,
  onRevert,
}: {
  text: string;
  /** the textarea holds an unapplied draft */
  dirty: boolean;
  /** the grants changed underneath an unapplied draft (visual edit while typing) */
  conflict: boolean;
  analysis: DraftAnalysis;
  onChange: (text: string) => void;
  onApply: () => void;
  onRevert: () => void;
}) {
  const hasErrors = analysis.errors.length > 0;
  const changeCount = analysis.added.length + analysis.removed.length;
  const rows = Math.max(text.split("\n").length + 1, 12);

  return (
    <section className="rounded-lg border border-grayA-4 p-4 flex flex-col gap-3 min-w-0">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-medium text-gray-12">Code</h2>
        <p className="text-xs text-gray-10">
          One permission per line, direct grants only. Edit freely, then apply the diff.
        </p>
      </header>

      {conflict && (
        <p className="text-xs text-warning-11 rounded-md bg-warningA-3 px-2 py-1.5">
          The grants changed while you were editing. Your draft is kept; applying makes this text
          the new source of truth, reverting discards it.
        </p>
      )}

      <Textarea
        value={text}
        onChange={(e) => onChange(e.target.value)}
        variant={hasErrors ? "error" : "default"}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        rows={rows}
        aria-label="Permissions as text, one per line"
        placeholder={`unkey:v1:${WORKSPACE_ID}:keyspaces/ks_payments_prod/keys/*#read_key`}
        className="font-mono text-xs leading-5 resize-y"
      />

      {hasErrors && (
        <ul className="flex flex-col gap-0.5" aria-live="polite">
          {analysis.errors.map((err) => (
            <li key={`${err.line}-${err.message}`} className="font-mono text-xs text-error-11">
              line {err.line}: {err.message}
            </li>
          ))}
        </ul>
      )}

      {dirty ? (
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div className="text-xs">
            {hasErrors ? (
              <span className="text-error-11">Fix the errors above to apply.</span>
            ) : changeCount > 0 ? (
              <span className="font-mono">
                <span className="text-success-11">+{analysis.added.length} added</span>
                <span className="text-gray-8">, </span>
                <span className="text-error-11">-{analysis.removed.length} removed</span>
              </span>
            ) : (
              <span className="text-gray-10">No changes yet. The draft matches the grants.</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={onRevert}>
              Revert
            </Button>
            <Button
              variant="primary"
              size="sm"
              disabled={hasErrors || changeCount === 0}
              onClick={onApply}
            >
              Apply changes
            </Button>
          </div>
        </div>
      ) : (
        <p className="text-xs text-gray-9">In sync with the visual editor.</p>
      )}
    </section>
  );
}
