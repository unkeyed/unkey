"use client";

import { type ReactNode, createContext, useContext } from "react";
import { createPortal } from "react-dom";

const PaneFooterContext = createContext<HTMLElement | null>(null);

export const PaneFooterProvider = PaneFooterContext.Provider;

// The footer sits outside the scrolling body, so a submit button rendered here
// is outside its form in the DOM and needs the form's id.
export function PaneActions({ children }: { children: ReactNode }) {
  const footer = useContext(PaneFooterContext);
  return footer ? createPortal(children, footer) : null;
}
