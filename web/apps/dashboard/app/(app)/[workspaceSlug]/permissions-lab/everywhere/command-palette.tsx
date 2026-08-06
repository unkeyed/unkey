"use client";

/**
 * Global command palette (cmdk inside the UI-kit dialog for focus trapping).
 * Multi-step flows: Grant, Revoke, and Check access. A breadcrumb of chosen
 * steps sits above the input; Backspace on an empty input steps back.
 */

import { Dialog, DialogContent, DialogTitle, cn, toast } from "@unkey/ui";
import { Command } from "cmdk";
import { useEffect, useState } from "react";
import { UrnText } from "../components/urn-display";
import { actionsForPath } from "../lib/catalog";
import { ALL_RESOURCES, WORKSPACE_ID, labelForPath, perm } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";

type Flow = "grant" | "revoke" | "check";

type Step = "root" | "principal" | "resource" | "action" | "revoke-grant" | "result";

interface CheckResult {
  action: string;
  allowed: boolean;
}

const FLOW_LABELS: Record<Flow, string> = {
  grant: "Grant",
  revoke: "Revoke",
  check: "Check access",
};

const ITEM_CLASS = cn(
  "flex items-center gap-2 rounded-md px-2 py-2 text-[13px] text-gray-12",
  "cursor-pointer select-none outline-none min-w-0",
  "data-[selected=true]:bg-grayA-3",
);

const GROUP_CLASS = cn(
  "**:[[cmdk-group-heading]]:px-2 **:[[cmdk-group-heading]]:py-1.5",
  "**:[[cmdk-group-heading]]:text-[11px] **:[[cmdk-group-heading]]:uppercase",
  "**:[[cmdk-group-heading]]:tracking-wide **:[[cmdk-group-heading]]:text-gray-9",
);

const PLACEHOLDERS: Record<Step, string> = {
  root: "Type a command...",
  principal: "Search root keys...",
  resource: "Search resources...",
  action: "Search actions...",
  "revoke-grant": "Search grants to revoke...",
  result: "What next?",
};

export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const lab = usePermissionsLab();

  const [search, setSearch] = useState("");
  const [flow, setFlow] = useState<Flow | null>(null);
  const [principalID, setPrincipalID] = useState<string | null>(null);
  const [resourcePath, setResourcePath] = useState<string | null>(null);
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        onOpenChange(!open);
      }
    };
    window.addEventListener("keydown", down);
    return () => window.removeEventListener("keydown", down);
  }, [open, onOpenChange]);

  const resetAll = () => {
    setSearch("");
    setFlow(null);
    setPrincipalID(null);
    setResourcePath(null);
    setCheckResult(null);
  };

  const close = () => {
    onOpenChange(false);
    resetAll();
  };

  const principal = principalID
    ? (lab.state.principals.find((p) => p.id === principalID) ?? null)
    : null;

  const step: Step =
    flow === null
      ? "root"
      : principal === null
        ? "principal"
        : flow === "revoke"
          ? "revoke-grant"
          : resourcePath === null
            ? "resource"
            : flow === "check" && checkResult !== null
              ? "result"
              : "action";

  const stepBack = () => {
    setSearch("");
    switch (step) {
      case "result":
        setCheckResult(null);
        break;
      case "action":
        setResourcePath(null);
        break;
      case "resource":
      case "revoke-grant":
        setPrincipalID(null);
        break;
      case "principal":
        setFlow(null);
        break;
      case "root":
        break;
    }
  };

  const crumbs: string[] = [];
  if (flow) {
    crumbs.push(FLOW_LABELS[flow]);
  }
  if (principal) {
    crumbs.push(principal.name);
  }
  if (resourcePath) {
    crumbs.push(labelForPath(resourcePath));
  }
  if (checkResult) {
    crumbs.push(checkResult.action);
  }

  const pickAction = (action: string) => {
    if (!principal || !resourcePath) {
      return;
    }
    const permission = perm(resourcePath, action);
    if (flow === "grant") {
      lab.commit(`Grant ${action} on ${resourcePath} to ${principal.name}`, [
        { op: "add", principalID: principal.id, permission },
      ]);
      toast.success(`Granted to ${principal.name}`, { description: permission });
      close();
      return;
    }
    const allowed = lab.can(principal.id, {
      urn: { workspaceID: WORKSPACE_ID, resource: resourcePath },
      action,
    });
    setCheckResult({ action, allowed });
    setSearch("");
  };

  const revokeGrant = (permission: string) => {
    if (!principal) {
      return;
    }
    lab.commit(`Revoke ${permission} from ${principal.name}`, [
      { op: "remove", principalID: principal.id, permission },
    ]);
    toast.success(`Revoked from ${principal.name}`, { description: permission });
    close();
  };

  const grantCheckedPermission = () => {
    if (!principal || !resourcePath || !checkResult) {
      return;
    }
    const permission = perm(resourcePath, checkResult.action);
    lab.commit(`Grant ${checkResult.action} on ${resourcePath} to ${principal.name}`, [
      { op: "add", principalID: principal.id, permission },
    ]);
    toast.success(`Granted to ${principal.name}`, { description: permission });
    close();
  };

  const directGrants = principal?.permissions ?? [];

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) {
          resetAll();
        }
      }}
    >
      <DialogContent
        className={cn(
          "top-[18%] translate-y-0 max-w-xl p-0 gap-0 overflow-hidden",
          "rounded-lg border border-grayA-4 bg-gray-1 shadow-xl",
          "[&_button[aria-label='Close_dialog']]:hidden",
        )}
      >
        <DialogTitle className="sr-only">Permissions command palette</DialogTitle>
        <Command loop label="Permissions command palette" className="flex flex-col">
          {crumbs.length > 0 && (
            <div className="flex flex-wrap items-center gap-1.5 border-b border-grayA-4 px-3 py-2">
              {crumbs.map((crumb, i) => (
                <span key={`${i}-${crumb}`} className="flex items-center gap-1.5">
                  {i > 0 && <span className="text-gray-8 text-xs">/</span>}
                  <span className="rounded bg-grayA-3 px-1.5 py-0.5 text-[11px] text-gray-11">
                    {crumb}
                  </span>
                </span>
              ))}
            </div>
          )}

          {step === "result" && principal && resourcePath && checkResult && (
            <div className="flex flex-col gap-2 border-b border-grayA-4 px-3 py-3">
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    "rounded px-1.5 py-0.5 text-[11px] font-semibold tracking-wide",
                    checkResult.allowed
                      ? "bg-successA-3 text-success-11"
                      : "bg-errorA-3 text-error-11",
                  )}
                >
                  {checkResult.allowed ? "ALLOWED" : "DENIED"}
                </span>
                <span className="text-xs text-gray-11">
                  {principal.name}{" "}
                  {checkResult.allowed
                    ? "has a grant covering this permission."
                    : "has no grant covering this permission."}
                </span>
              </div>
              <div className="overflow-x-auto">
                <UrnText value={perm(resourcePath, checkResult.action)} />
              </div>
            </div>
          )}

          <Command.Input
            autoFocus
            value={search}
            onValueChange={setSearch}
            placeholder={PLACEHOLDERS[step]}
            onKeyDown={(e) => {
              if (e.key === "Backspace" && search === "" && step !== "root") {
                e.preventDefault();
                stepBack();
              }
            }}
            className={cn(
              "h-11 w-full border-b border-grayA-4 bg-transparent px-3 text-[13px] text-gray-12",
              "placeholder:text-grayA-8 outline-none",
            )}
          />

          <Command.List className="max-h-80 overflow-y-auto overflow-x-hidden p-1">
            <Command.Empty className="py-6 text-center text-[13px] text-gray-9">
              No results.
            </Command.Empty>

            {step === "root" && (
              <Command.Group heading="Permissions" className={GROUP_CLASS}>
                <Command.Item
                  value="grant permission"
                  onSelect={() => {
                    setFlow("grant");
                    setSearch("");
                  }}
                  className={ITEM_CLASS}
                >
                  <span>Grant permission...</span>
                </Command.Item>
                <Command.Item
                  value="revoke permission"
                  onSelect={() => {
                    setFlow("revoke");
                    setSearch("");
                  }}
                  className={ITEM_CLASS}
                >
                  <span>Revoke permission...</span>
                </Command.Item>
                <Command.Item
                  value="check access"
                  onSelect={() => {
                    setFlow("check");
                    setSearch("");
                  }}
                  className={ITEM_CLASS}
                >
                  <span>Check access...</span>
                </Command.Item>
              </Command.Group>
            )}

            {step === "principal" && (
              <Command.Group heading="Root keys" className={GROUP_CLASS}>
                {lab.state.principals.map((p) => (
                  <Command.Item
                    key={p.id}
                    value={`${p.name} ${p.id}`}
                    onSelect={() => {
                      setPrincipalID(p.id);
                      setSearch("");
                    }}
                    className={ITEM_CLASS}
                  >
                    <span className="truncate">{p.name}</span>
                    <span className="ml-auto font-mono text-[11px] text-gray-9">{p.id}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {step === "resource" && (
              <Command.Group heading="Resources" className={GROUP_CLASS}>
                {ALL_RESOURCES.map((r) => (
                  <Command.Item
                    key={r.path}
                    value={`${r.label} ${r.path}`}
                    onSelect={() => {
                      setResourcePath(r.path);
                      setSearch("");
                    }}
                    className={ITEM_CLASS}
                  >
                    <span className="truncate">{r.label}</span>
                    <span className="ml-auto font-mono text-[11px] text-gray-9 truncate max-w-[55%]">
                      {r.path}
                    </span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {step === "action" && resourcePath && (
              <Command.Group heading="Actions" className={GROUP_CLASS}>
                {actionsForPath(resourcePath).length === 0 ? (
                  <div className="px-2 py-4 text-center text-[13px] text-gray-9">
                    No actions apply to this resource.
                  </div>
                ) : (
                  actionsForPath(resourcePath).map((a) => (
                    <Command.Item
                      key={a.action}
                      value={`${a.action} ${a.description}`}
                      onSelect={() => pickAction(a.action)}
                      className={ITEM_CLASS}
                    >
                      <span className="font-mono text-xs text-info-11">{a.action}</span>
                      <span className="ml-auto text-[11px] text-gray-9 truncate max-w-[60%]">
                        {a.description}
                      </span>
                    </Command.Item>
                  ))
                )}
              </Command.Group>
            )}

            {step === "revoke-grant" &&
              (directGrants.length === 0 ? (
                <div className="px-2 py-6 text-center text-[13px] text-gray-9">
                  {principal?.name ?? "This key"} has no direct grants to revoke.
                </div>
              ) : (
                <Command.Group heading="Direct grants" className={GROUP_CLASS}>
                  {directGrants.map((grant) => (
                    <Command.Item
                      key={grant}
                      value={grant}
                      onSelect={() => revokeGrant(grant)}
                      className={ITEM_CLASS}
                    >
                      <span className="overflow-x-hidden">
                        <UrnText value={grant} />
                      </span>
                    </Command.Item>
                  ))}
                </Command.Group>
              ))}

            {step === "result" && checkResult && (
              <Command.Group heading="Next" className={GROUP_CLASS}>
                {!checkResult.allowed && (
                  <Command.Item
                    value="grant this permission"
                    onSelect={grantCheckedPermission}
                    className={ITEM_CLASS}
                  >
                    <span>Grant this permission now</span>
                  </Command.Item>
                )}
                <Command.Item
                  value="check another action"
                  onSelect={() => {
                    setCheckResult(null);
                    setSearch("");
                  }}
                  className={ITEM_CLASS}
                >
                  <span>Check another action</span>
                </Command.Item>
                <Command.Item
                  value="check another resource"
                  onSelect={() => {
                    setCheckResult(null);
                    setResourcePath(null);
                    setSearch("");
                  }}
                  className={ITEM_CLASS}
                >
                  <span>Check another resource</span>
                </Command.Item>
                <Command.Item value="close palette" onSelect={close} className={ITEM_CLASS}>
                  <span>Close</span>
                </Command.Item>
              </Command.Group>
            )}
          </Command.List>

          <div className="flex items-center gap-3 border-t border-grayA-4 px-3 py-2 text-[11px] text-gray-9">
            <span>
              <span className="font-mono">↑↓</span> navigate
            </span>
            <span>
              <span className="font-mono">↵</span> select
            </span>
            <span>
              <span className="font-mono">⌫</span> back
            </span>
            <span className="ml-auto">
              <span className="font-mono">esc</span> close
            </span>
          </div>
        </Command>
      </DialogContent>
    </Dialog>
  );
}
