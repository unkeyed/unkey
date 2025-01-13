"use client";

// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";

export type ActionProps = {
  className?: string;
  children?: React.ReactNode;
  onApply: () => void;
};

function DateTimeActions({ className, children, onApply }: ActionProps) {
  return <div>Actions</div>;
}
DateTimeActions.displayName = "DateTime.Actions";
export { DateTimeActions };
