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
    <div
      className={cn("w-full h-full flex items-center justify-center gap-4 mt-2 px-1", className)}
    >
      {children}
    </div>
  );
};
export { DateTimeActions };
