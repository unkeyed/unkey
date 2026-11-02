"use client";

import { Button } from "@unkey/ui";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { usePaneFooter } from "./card";

type PaneSubmitProps = {
  loading?: boolean;
  disabled?: boolean;
  children?: ReactNode;
  notice?: ReactNode;
} & ({ form: string } | { onClick: () => void });

export function PaneSubmit({
  loading = false,
  disabled = loading,
  children = "Continue",
  notice,
  ...action
}: PaneSubmitProps) {
  const footer = usePaneFooter();
  if (!footer) {
    return null;
  }
  return createPortal(
    <>
      {notice}
      <Button
        type={"form" in action ? "submit" : "button"}
        {...action}
        variant="primary"
        size="sm"
        className="px-3"
        loading={loading}
        disabled={disabled}
      >
        {children}
      </Button>
    </>,
    footer,
  );
}
