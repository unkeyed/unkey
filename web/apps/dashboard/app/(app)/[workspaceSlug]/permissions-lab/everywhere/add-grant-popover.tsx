"use client";

/**
 * Inline "+ Add" chip that opens a compact grant composer in a popover:
 * pattern input with segment suggestions, action picker, live URN preview.
 * Fast like adding a label in Linear; Escape dismisses.
 */

import { Button, Input, Popover, PopoverContent, PopoverTrigger, cn, toast } from "@unkey/ui";
import { useMemo, useState } from "react";
import { UrnText } from "../components/urn-display";
import { actionsForPath } from "../lib/catalog";
import { perm, suggestNextSegments } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { validateResourcePath } from "../lib/urn";

const SUGGESTION_LIMIT = 6;

export function AddGrantPopover({
  principalID,
  principalName,
}: {
  principalID: string;
  principalName: string;
}) {
  const lab = usePermissionsLab();
  const [open, setOpen] = useState(false);
  const [path, setPath] = useState("");
  const [pickedAction, setPickedAction] = useState<string | null>(null);

  const pathError = path === "" ? null : validateResourcePath(path);
  const pathValid = path !== "" && pathError === null;

  const actions = useMemo(() => (pathValid ? actionsForPath(path) : []), [path, pathValid]);
  const action =
    pickedAction && actions.some((a) => a.action === pickedAction) ? pickedAction : null;

  const suggestions = useMemo(() => {
    const segments = path === "" ? [""] : path.split("/");
    const prefix = segments.slice(0, -1);
    const partial = segments[segments.length - 1] ?? "";
    if (prefix.some((s) => s === "" || s === "**")) {
      return [];
    }
    return suggestNextSegments(prefix)
      .filter((s) => s.value.startsWith(partial) && s.value !== partial)
      .slice(0, SUGGESTION_LIMIT);
  }, [path]);

  const applySuggestion = (value: string, kind: string) => {
    const segments = path === "" ? [""] : path.split("/");
    const prefix = segments.slice(0, -1);
    const next = [...prefix, value].join("/");
    setPath(kind === "collection" ? `${next}/` : next);
  };

  const reset = () => {
    setPath("");
    setPickedAction(null);
  };

  const ready = pathValid && action !== null;

  const submit = () => {
    if (!pathValid || action === null) {
      return;
    }
    const permission = perm(path, action);
    lab.commit(`Grant ${action} on ${path} to ${principalName}`, [
      { op: "add", principalID, permission },
    ]);
    toast.success("Permission granted", { description: permission });
    reset();
    setOpen(false);
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) {
          reset();
        }
      }}
    >
      <PopoverTrigger
        render={
          <button
            type="button"
            className="inline-flex items-center gap-1 rounded-md border border-dashed border-grayA-6 bg-transparent px-2 py-1 text-xs text-gray-10 transition-colors hover:border-grayA-8 hover:bg-grayA-2 hover:text-gray-12 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-6"
          >
            + Add
          </button>
        }
      />
      <PopoverContent align="start" className="w-96 p-3">
        <div className="flex flex-col gap-2.5">
          <div className="flex flex-col gap-1.5">
            <label htmlFor="grant-pattern" className="text-xs font-medium text-gray-11">
              Resource pattern
            </label>
            <Input
              id="grant-pattern"
              value={path}
              variant={pathError ? "error" : "default"}
              placeholder="keyspaces/ks_payments_prod/keys/*"
              className="font-mono text-xs"
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setPath(e.currentTarget.value.trim())}
              onKeyDown={(e) => {
                if (e.key === "Enter" && ready) {
                  e.preventDefault();
                  submit();
                }
              }}
            />
            {pathError && <p className="text-xs text-error-11">{pathError}</p>}
            {suggestions.length > 0 && (
              <div className="flex flex-col rounded-md border border-grayA-4 overflow-hidden">
                {suggestions.map((s) => (
                  <button
                    key={s.value}
                    type="button"
                    onClick={() => applySuggestion(s.value, s.kind)}
                    className="flex items-center justify-between gap-3 px-2 py-1.5 text-left transition-colors hover:bg-grayA-3 focus-visible:outline-hidden focus-visible:bg-grayA-3"
                  >
                    <span
                      className={cn(
                        "font-mono text-xs",
                        s.kind === "wildcard" || s.kind === "descendants"
                          ? "text-warning-11"
                          : "text-gray-12",
                      )}
                    >
                      {s.value}
                    </span>
                    <span className="text-[11px] text-gray-9 truncate">{s.label}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-gray-11">Action</span>
            {pathValid ? (
              actions.length === 0 ? (
                <p className="text-xs text-warning-11">
                  No known actions apply to this pattern. Check the path shape.
                </p>
              ) : (
                <div className="flex flex-col rounded-md border border-grayA-4 max-h-36 overflow-y-auto">
                  {actions.map((a) => (
                    <button
                      key={a.action}
                      type="button"
                      aria-pressed={action === a.action}
                      onClick={() => setPickedAction(a.action)}
                      className={cn(
                        "flex items-center justify-between gap-3 px-2 py-1.5 text-left transition-colors hover:bg-grayA-3 focus-visible:outline-hidden focus-visible:bg-grayA-3",
                        action === a.action && "bg-grayA-3",
                      )}
                    >
                      <span
                        className={cn(
                          "font-mono text-xs",
                          a.action === "*" ? "text-error-11" : "text-info-11",
                        )}
                      >
                        {a.action}
                      </span>
                      <span className="text-[11px] text-gray-9 truncate">{a.description}</span>
                    </button>
                  ))}
                </div>
              )
            ) : (
              <p className="text-xs text-gray-9">Enter a valid resource pattern first.</p>
            )}
          </div>

          {ready && action !== null && (
            <div className="rounded-md bg-grayA-2 border border-grayA-4 px-2 py-1.5 overflow-x-auto">
              <UrnText value={perm(path, action)} />
            </div>
          )}

          <div className="flex items-center justify-between pt-0.5">
            <span className="text-[11px] text-gray-9">Esc to dismiss</span>
            <Button variant="primary" size="sm" disabled={!ready} onClick={submit}>
              Add permission
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
