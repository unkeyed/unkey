import { TriangleWarning } from "@unkey/icons";
import type React from "react";
import type { PropsWithChildren } from "react";

export const ErrorBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="border border-error-6 text-error-11 p-4 rounded-lg bg-error-3">
    <p className="text-sm">{children}</p>
  </div>
);

export const WarnBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="border border-warning-6 text-warning-11 p-4 rounded-lg bg-warning-3 flex items-center gap-4 text-sm">
    <TriangleWarning className="w-4 h-4" />

    {children}
  </div>
);
