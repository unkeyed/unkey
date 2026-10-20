"use client";

import { Button } from "@unkey/ui";
import { type ReactNode, createContext, use } from "react";
import { createPortal } from "react-dom";

const PaneFooterContext = createContext<HTMLElement | null>(null);

export const PaneFooterProvider = PaneFooterContext.Provider;

type PaneSubmitProps = {
  loading: boolean;
  disabled?: boolean;
  children?: ReactNode;
  notice?: ReactNode;
} & ({ form: string } | { onClick: () => void });

export function PaneSubmit({
  loading,
  disabled = loading,
  children = "Continue",
  notice,
  ...action
}: PaneSubmitProps) {
  const footer = use(PaneFooterContext);
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
