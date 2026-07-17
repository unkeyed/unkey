"use client";

/**
 * Human-readable diff line for a single changeset operation. Shared between
 * the draft composer and the timeline so a change reads identically before
 * staging and in history.
 */

import { cn } from "@unkey/ui";
import type { ReactNode } from "react";
import { UrnText } from "../components/urn-display";
import { labelForPath } from "../lib/mock-data";
import { type ChangeOp, usePermissionsLab } from "../lib/store";
import { isPattern, parsePermission } from "../lib/urn";

function isAddOp(op: ChangeOp): boolean {
  return op.op === "add" || op.op === "add_role";
}

function Mono({ children }: { children: ReactNode }) {
  return <span className="font-mono text-xs text-gray-12">{children}</span>;
}

export function DiffLine({ op, onRemove }: { op: ChangeOp; onRemove?: () => void }) {
  const { state } = usePermissionsLab();
  const add = isAddOp(op);
  const principal = state.principals.find((p) => p.id === op.principalID);
  const principalName = principal?.name ?? op.principalID;

  let sentence: ReactNode;
  let detail: ReactNode = null;

  if ("permission" in op) {
    const parsed = parsePermission(op.permission);
    if (parsed.ok) {
      const path = parsed.value.urn.resource;
      const action = parsed.value.action;
      const verb = add ? "can" : "can no longer";
      if (action === "*") {
        sentence = (
          <>
            {principalName} {verb} perform <Mono>every action</Mono> in the workspace
          </>
        );
      } else if (isPattern(path)) {
        sentence = (
          <>
            {principalName} {verb} <Mono>{action}</Mono> in resources matching <Mono>{path}</Mono>
          </>
        );
      } else {
        sentence = (
          <>
            {principalName} {verb} <Mono>{action}</Mono> in {labelForPath(path)}{" "}
            <span className="font-mono text-xs text-gray-10">({path})</span>
          </>
        );
      }
    } else {
      // A permission that fails to parse should never be staged, but render it
      // rather than hiding the op.
      sentence = (
        <>
          {principalName} {add ? "gains" : "loses"} a permission
        </>
      );
    }
    detail = <UrnText value={op.permission} />;
  } else {
    const role = state.roles.find((r) => r.id === op.roleID);
    const roleName = role?.name ?? op.roleID;
    sentence = (
      <>
        {principalName} {add ? "gets" : "loses"} role <Mono>{roleName}</Mono>
      </>
    );
    if (role) {
      detail = <span className="text-xs text-gray-10">{role.description}</span>;
    }
  }

  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border-l-2 bg-grayA-2 px-3 py-2",
        add ? "border-success-9" : "border-error-9",
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          "select-none font-mono text-sm font-semibold leading-5",
          add ? "text-success-11" : "text-error-11",
        )}
      >
        {add ? "+" : "-"}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="text-sm leading-5 text-gray-12">{sentence}</span>
        {detail && <div className="overflow-x-auto py-0.5">{detail}</div>}
      </div>
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label="Remove change from draft"
          className="leading-5 text-gray-9 transition-colors hover:text-error-11"
        >
          &times;
        </button>
      )}
    </div>
  );
}
