import { match } from "@unkey/match";
import type { ReactNode } from "react";
import type { PathHint } from "./path-hint";

export function pathHintMessage(
  hint: PathHint,
  noun: "File" | "Directory",
  onAccept: (path: string) => void,
): ReactNode {
  return match(hint)
    .with({ type: "none" }, () => undefined)
    .with({ type: "case-match" }, ({ path }) => (
      <span>
        Did you mean{" "}
        <button
          type="button"
          className="underline font-medium hover:text-warning-12"
          onClick={() => onAccept(path)}
        >
          {path}
        </button>
        ?
      </span>
    ))
    .with({ type: "not-found" }, ({ branch }) =>
      branch ? (
        <span>
          {noun} not found on branch <span className="font-medium text-gray-12">{branch}</span>
        </span>
      ) : (
        `${noun} not found on this branch`
      ),
    )
    .exhaustive();
}
