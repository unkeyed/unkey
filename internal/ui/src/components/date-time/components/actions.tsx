"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../../lib/utils";

type ActionProps = {
  className?: string;
  children?: React.ReactNode;
};

const DateTimeActions: React.FC<ActionProps> = ({ className, children }) => {
  return (
    <div className={cn("w-full flex items-center justify-center pb-2", className)}>{children}</div>
  );
};
export { DateTimeActions };
