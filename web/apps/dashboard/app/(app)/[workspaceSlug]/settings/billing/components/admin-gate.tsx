"use client";

import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

type AdminGateProps = {
  isAdmin: boolean | undefined;
  blocked?: boolean;
  blockedReason?: string;
  children: (disabled: boolean) => ReactNode;
};

export function AdminGate({ isAdmin, blocked = false, blockedReason, children }: AdminGateProps) {
  const disabled = isAdmin !== true || blocked;
  const reason = isAdmin === false ? ADMIN_ONLY_TOOLTIP : blockedReason;

  return (
    <InfoTooltip content={reason} disabled={!disabled || reason === undefined} asChild>
      <span>{children(disabled)}</span>
    </InfoTooltip>
  );
}
