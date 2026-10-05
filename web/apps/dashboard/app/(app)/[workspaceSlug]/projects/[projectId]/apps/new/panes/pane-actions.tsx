"use client";

import { Button } from "@unkey/ui";
import { type ReactNode, createContext, useContext } from "react";
import { createPortal } from "react-dom";

const PaneFooterContext = createContext<HTMLElement | null>(null);

export const PaneFooterProvider = PaneFooterContext.Provider;

type PaneSubmitProps = {
  loading: boolean;
  disabled?: boolean;
  children?: ReactNode;
} & ({ form: string } | { onClick: () => void });

// The footer sits outside the scrolling body, so a submit button rendered here
// is outside its form in the DOM and needs the form's id.
export function PaneSubmit({
  loading,
  disabled = loading,
  children = "Continue",
  ...action
}: PaneSubmitProps) {
  const footer = useContext(PaneFooterContext);
  if (!footer) {
    return null;
  }
  return createPortal(
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
    </Button>,
    footer,
  );
}
