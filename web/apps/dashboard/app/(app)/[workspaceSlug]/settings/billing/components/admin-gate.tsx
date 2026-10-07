"use client";

import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

type AdminGateProps = {
  isAdmin: boolean | undefined;
  children: (disabled: boolean) => ReactNode;
};

export function AdminGate({ isAdmin, children }: AdminGateProps) {
  const disabled = isAdmin !== true;
  const reason = isAdmin === false ? ADMIN_ONLY_TOOLTIP : undefined;

  return (
    <InfoTooltip content={reason} disabled={!disabled || reason === undefined} asChild>
      <span>{children(disabled)}</span>
    </InfoTooltip>
  );
}
