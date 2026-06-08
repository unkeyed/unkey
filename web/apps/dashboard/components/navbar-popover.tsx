"use client";
import React, { type PropsWithChildren } from "react";

export const QuickNavPopover = ({ children }: PropsWithChildren) => {
  return <QuickNavLabel>{children}</QuickNavLabel>;
};

const QuickNavLabel = ({ children }: PropsWithChildren) => {
  if (React.isValidElement<{ children?: React.ReactNode }>(children)) {
    const inner = React.Children.toArray(children.props.children);
    return <>{inner[0] ?? null}</>;
  }
  return <>{children}</>;
};
