"use client";

/**
 * The omnibox itself: one large monospace input that composes
 * `resource_path#action` with segment-aware autocomplete. The URN prefix is a
 * fixed decoration; wildcards and actions are suggested in place, so valid
 * grammar is the path of least resistance.
 */

import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  cn,
  toast,
} from "@unkey/ui";
import { type ChangeEvent, type KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";
import { UrnText } from "../components/urn-display";
import { actionsForPath } from "../lib/catalog";
import { WORKSPACE_ID, coverage, labelForPath, perm, suggestNextSegments } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { isPattern, validateAction, validateResourcePath } from "../lib/urn";

export interface GrantRecord {
  id: string;
  principalID: string;
  principalName: string;
  permission: string;
  at: number;
}

type OmniSuggestion =
  | {
      type: "segment";
      value: string;
      label: string;
      kind: "collection" | "id" | "wildcard" | "descendants";
    }
  | { type: "hash"; path: string }
  | { type: "action"; action: string; description: string };

interface Analysis {
  mode: "path" | "action";
  pathPart: string;
  actionPart: string;
  /** fully typed segments before the partial one */
  completed: string[];
  /** the segment currently being typed ("" right after a "/") */
  partial: string;
  /** the path as it stands, trailing slash stripped */
  candidatePath: string;
  /** a valid, actionable path the user could put "#" after right now */
  grantablePath: string | null;
  error: string | null;
  complete: boolean;
  permission: string | null;
}

function analyze(input: string): Analysis {
  const hashIdx = input.indexOf("#");
  const mode: Analysis["mode"] = hashIdx === -1 ? "path" : "action";
  const pathPart = mode === "path" ? input : input.slice(0, hashIdx);
  const actionPart = mode === "action" ? input.slice(hashIdx + 1) : "";

  const rawSegments = pathPart === "" ? [] : pathPart.split("/");
  const partial =
    mode === "path" && rawSegments.length > 0 ? rawSegments[rawSegments.length - 1] : "";
  const completed = mode === "path" ? rawSegments.slice(0, -1) : rawSegments;
  const candidatePath = mode === "path" && partial === "" ? completed.join("/") : pathPart;

  let error: string | null = null;
  if (mode === "path") {
    error = candidatePath === "" ? null : validateResourcePath(candidatePath);
  } else {
    error =
      validateResourcePath(pathPart) ??
      (actionPart === "" ? null : validateAction(actionPart, pathPart));
  }

  const grantablePath =
    mode === "path" &&
    candidatePath !== "" &&
    validateResourcePath(candidatePath) === null &&
    actionsForPath(candidatePath).length > 0
      ? candidatePath
      : null;

  const complete = mode === "action" && actionPart !== "" && error === null;
  return {
    mode,
    pathPart,
    actionPart,
    completed,
    partial,
    candidatePath,
    grantablePath,
    error,
    complete,
    permission: complete ? perm(pathPart, actionPart) : null,
  };
}

function coverageText(path: string): string {
  if (!isPattern(path)) {
    return `covers exactly this resource (${labelForPath(path)})`;
  }
  const count = coverage(path).length;
  if (count === 0) {
    return "covers no resources today";
  }
  return `covers ${count} resource${count === 1 ? "" : "s"} today`;
}

function KindBadge({ kind }: { kind: "id" | "collection" | "wildcard" | "actions" }) {
  const styles = {
    id: "text-info-11 bg-infoA-3",
    collection: "text-gray-11 bg-grayA-3",
    wildcard: "text-warning-11 bg-warningA-3",
    actions: "text-gray-11 bg-grayA-3",
  }[kind];
  return (
    <span
      className={cn("rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide shrink-0", styles)}
    >
      {kind}
    </span>
  );
}

export function OmniboxComposer({
  principalID,
  onPrincipalChange,
  onGranted,
}: {
  principalID: string;
  onPrincipalChange: (id: string) => void;
  onGranted: (record: GrantRecord) => void;
}) {
  const lab = usePermissionsLab();
  const [input, setInput] = useState("");
  const [highlight, setHighlight] = useState(0);
  const [closed, setClosed] = useState(false);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const optionRefs = useRef<(HTMLLIElement | null)[]>([]);

  const principals = lab.state.principals;
  const principal = principals.find((p) => p.id === principalID);

  const analysis = useMemo(() => analyze(input), [input]);
  const {
    mode,
    pathPart,
    actionPart,
    completed,
    partial,
    grantablePath,
    error,
    complete,
    permission,
  } = analysis;

  const suggestions = useMemo<OmniSuggestion[]>(() => {
    if (mode === "action") {
      const q = actionPart.toLowerCase();
      return actionsForPath(pathPart)
        .filter(
          (a) => a.action.startsWith(q) || (q !== "" && a.description.toLowerCase().includes(q)),
        )
        .map((a) => ({ type: "action" as const, action: a.action, description: a.description }));
    }
    if (error !== null) {
      return [];
    }
    const q = partial.toLowerCase();
    const segments = suggestNextSegments(completed)
      .filter(
        (s) =>
          s.value.toLowerCase().startsWith(q) || (q !== "" && s.label.toLowerCase().includes(q)),
      )
      .map((s) => ({ type: "segment" as const, value: s.value, label: s.label, kind: s.kind }));
    const hash: OmniSuggestion | null = grantablePath
      ? { type: "hash", path: grantablePath }
      : null;
    const out: OmniSuggestion[] = [];
    if (hash && partial === "") {
      out.push(hash);
    }
    out.push(...segments);
    if (hash && partial !== "") {
      out.push(hash);
    }
    return out;
  }, [mode, actionPart, pathPart, error, partial, completed, grantablePath]);

  // The list shrank underneath the cursor: snap back to the top entry.
  useEffect(() => {
    if (highlight >= suggestions.length) {
      setHighlight(0);
    }
  }, [highlight, suggestions.length]);

  useEffect(() => {
    optionRefs.current[highlight]?.scrollIntoView({ block: "nearest" });
  }, [highlight]);

  const open = focused && !closed;
  const listVisible = open && suggestions.length > 0;
  const emptyVisible =
    open && suggestions.length === 0 && (mode === "action" || partial !== "") && error === null;

  const alreadyGranted =
    permission !== null && lab.effectivePermissions(principalID).includes(permission);

  const accept = (s: OmniSuggestion) => {
    if (s.type === "segment") {
      const next = [...completed, s.value].join("/");
      // "**" must stay the last segment, so jump straight to the action.
      setInput(s.value === "**" ? `${next}#` : `${next}/`);
      setClosed(false);
    } else if (s.type === "hash") {
      setInput(`${s.path}#`);
      setClosed(false);
    } else {
      setInput(`${pathPart}#${s.action}`);
      setClosed(true);
    }
    setHighlight(0);
    inputRef.current?.focus();
  };

  const grant = () => {
    if (!permission || !principal || alreadyGranted) {
      return;
    }
    lab.commit("Grant via omnibox", [{ op: "add", principalID, permission }]);
    onGranted({
      id: crypto.randomUUID(),
      principalID,
      principalName: principal.name,
      permission,
      at: Date.now(),
    });
    toast.success(`Granted to ${principal.name}`);
    setInput("");
    setClosed(false);
    setHighlight(0);
    inputRef.current?.focus();
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!listVisible) {
        setClosed(false);
        return;
      }
      const delta = e.key === "ArrowDown" ? 1 : -1;
      setHighlight((h) => (h + delta + suggestions.length) % suggestions.length);
      return;
    }
    if (e.key === "Escape") {
      if (open) {
        e.preventDefault();
        setClosed(true);
      }
      return;
    }
    if (e.key === "Tab") {
      const s = suggestions[highlight];
      if (listVisible && s) {
        e.preventDefault();
        accept(s);
      }
      return;
    }
    if (e.key === "Enter") {
      const s = suggestions[highlight];
      if (listVisible && s) {
        e.preventDefault();
        accept(s);
        return;
      }
      if (complete && !alreadyGranted) {
        e.preventDefault();
        grant();
      }
    }
  };

  const onChange = (e: ChangeEvent<HTMLInputElement>) => {
    // No whitespace in URNs; "/#" collapses to "#" so typing "#" right after
    // accepting a segment does the obvious thing.
    let next = e.target.value.replace(/\s+/g, "");
    if (next.endsWith("/#")) {
      next = `${next.slice(0, -2)}#`;
    }
    setInput(next);
    setClosed(false);
    setHighlight(0);
  };

  const activeID = listVisible ? `omnibox-option-${highlight}` : undefined;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col lg:flex-row gap-3 lg:items-start">
        <div className="relative flex-1 min-w-0">
          <div
            className={cn(
              "flex items-center h-12 px-4 rounded-lg border bg-gray-2 dark:bg-black font-mono text-sm transition-colors",
              "focus-within:ring-3 focus-within:ring-gray-5",
              error !== null
                ? "border-error-9 focus-within:border-error-9 focus-within:ring-error-4"
                : "border-gray-5 focus-within:border-accent-12",
            )}
          >
            <span aria-hidden="true" className="select-none whitespace-nowrap text-gray-8">
              unkey:v1:
            </span>
            <span aria-hidden="true" className="select-none whitespace-nowrap text-gray-9">
              {WORKSPACE_ID}:
            </span>
            <input
              ref={inputRef}
              value={input}
              onChange={onChange}
              onKeyDown={onKeyDown}
              onFocus={() => {
                setFocused(true);
                setClosed(false);
              }}
              onBlur={() => setFocused(false)}
              role="combobox"
              aria-label="Permission to grant"
              aria-expanded={listVisible}
              aria-controls="omnibox-listbox"
              aria-activedescendant={activeID}
              aria-autocomplete="list"
              autoComplete="off"
              spellCheck={false}
              placeholder="resource/path#action"
              className="flex-1 min-w-0 bg-transparent outline-none font-mono text-sm text-gray-12 placeholder:text-gray-7"
            />
          </div>

          {(listVisible || emptyVisible) && (
            <div className="absolute left-0 right-0 top-full mt-2 z-50 rounded-lg border border-grayA-4 bg-gray-1 shadow-lg overflow-hidden">
              {listVisible ? (
                // biome-ignore lint/a11y/useFocusableInteractive: focus stays on the input; options are referenced via aria-activedescendant
                <ul
                  id="omnibox-listbox"
                  // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: WAI-ARIA combobox popup; the suggestion list is a listbox by spec
                  // biome-ignore lint/a11y/useSemanticElements: a native select cannot back a free-text combobox
                  role="listbox"
                  aria-label={mode === "action" ? "Actions" : "Path segments"}
                  className="max-h-72 overflow-y-auto p-1"
                >
                  {suggestions.map((s, i) => {
                    const key =
                      s.type === "segment"
                        ? `seg-${s.value}`
                        : s.type === "hash"
                          ? "hash"
                          : `act-${s.action}`;
                    return (
                      // biome-ignore lint/a11y/useFocusableInteractive: not focusable by design; highlighted via aria-activedescendant
                      // biome-ignore lint/a11y/useKeyWithClickEvents: keyboard selection is handled on the combobox input
                      <li
                        key={key}
                        id={`omnibox-option-${i}`}
                        ref={(el) => {
                          optionRefs.current[i] = el;
                        }}
                        // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: WAI-ARIA combobox option by spec
                        // biome-ignore lint/a11y/useSemanticElements: options live in a custom listbox, not a native select
                        role="option"
                        aria-selected={i === highlight}
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => accept(s)}
                        onMouseEnter={() => setHighlight(i)}
                        className={cn(
                          "flex items-center gap-3 px-3 py-2 rounded-md cursor-default",
                          i === highlight && "bg-grayA-3",
                        )}
                      >
                        {s.type === "segment" && (
                          <>
                            <span
                              className={cn(
                                "font-mono text-xs",
                                s.kind === "wildcard" || s.kind === "descendants"
                                  ? "text-warning-11 font-semibold"
                                  : "text-gray-12",
                              )}
                            >
                              {s.value}
                            </span>
                            {s.label !== s.value && (
                              <span className="text-xs text-gray-11 truncate">{s.label}</span>
                            )}
                            <span className="flex-1" />
                            <KindBadge
                              kind={
                                s.kind === "descendants"
                                  ? "wildcard"
                                  : s.kind === "wildcard"
                                    ? "wildcard"
                                    : s.kind
                              }
                            />
                          </>
                        )}
                        {s.type === "hash" && (
                          <>
                            <span className="font-mono text-xs text-gray-12 font-semibold">#</span>
                            <span className="text-xs text-gray-11 truncate">
                              grant an action on this path
                            </span>
                            <span className="flex-1" />
                            <KindBadge kind="actions" />
                          </>
                        )}
                        {s.type === "action" && (
                          <>
                            <span
                              className={cn(
                                "font-mono text-xs",
                                s.action === "*" ? "text-error-11 font-semibold" : "text-info-11",
                              )}
                            >
                              {s.action}
                            </span>
                            <span className="text-xs text-gray-11 truncate">{s.description}</span>
                            <span className="flex-1" />
                            {s.action === "*" && <KindBadge kind="wildcard" />}
                          </>
                        )}
                      </li>
                    );
                  })}
                </ul>
              ) : (
                <div className="px-3 py-2.5 text-xs text-gray-10">
                  {mode === "action"
                    ? "No matching actions for this resource."
                    : "No completions. Keep typing to use a custom id."}
                </div>
              )}
              <div className="flex items-center gap-3 border-t border-grayA-4 px-3 py-1.5 text-[10px] text-gray-9">
                <span>
                  <kbd className="font-sans">&uarr;&darr;</kbd> navigate
                </span>
                <span>
                  <kbd className="font-sans">tab</kbd> or <kbd className="font-sans">enter</kbd>{" "}
                  accept
                </span>
                <span>
                  <kbd className="font-sans">esc</kbd> close
                </span>
                {complete && (
                  <span>
                    <kbd className="font-sans">enter</kbd> grant
                  </span>
                )}
              </div>
            </div>
          )}
        </div>

        <Select
          value={principalID}
          items={principals.map((p) => ({ value: p.id, label: p.name }))}
          onValueChange={(next) => {
            if (typeof next === "string") {
              onPrincipalChange(next);
            }
          }}
        >
          <SelectTrigger className="h-12 w-full lg:w-56" aria-label="Principal">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {principals.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                <span className="flex flex-col items-start gap-0.5">
                  <span>{p.name}</span>
                  <span className="font-mono text-[11px] text-gray-9">{p.id}</span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Button
          variant="primary"
          className="h-12 px-6 w-full lg:w-auto"
          disabled={!complete || alreadyGranted || !principal}
          onClick={grant}
        >
          Grant
        </Button>
      </div>

      <div className="min-h-10 flex flex-col gap-1 px-1 text-xs">
        {input === "" ? (
          <span className="text-gray-9">
            Type a resource path, then <span className="font-mono">#</span> and an action. Focus the
            input to browse from the top.
          </span>
        ) : error !== null ? (
          <span className="text-error-11">{error}</span>
        ) : complete && permission !== null ? (
          <>
            <UrnText value={permission} />
            <span className="text-gray-10">
              {coverageText(pathPart)}
              {alreadyGranted && principal && (
                <span className="text-info-11">
                  {" "}
                  &middot; {principal.name} already has this exact grant, granting again would
                  change nothing
                </span>
              )}
            </span>
          </>
        ) : mode === "action" ? (
          <span className="text-gray-10">
            {coverageText(pathPart)} &middot; pick an action to finish
          </span>
        ) : grantablePath !== null ? (
          <span className="text-gray-10">
            {coverageText(grantablePath)} &middot; type <span className="font-mono">#</span> to pick
            an action
          </span>
        ) : (
          <span className="text-gray-9">Keep typing, this path needs more segments.</span>
        )}
      </div>
    </div>
  );
}
