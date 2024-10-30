import { AlertTriangle } from "lucide-react";
import type React from "react";
import type { PropsWithChildren } from "react";

export const ErrorBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="border border-[#FB1048]/15 text-[#FB1048] p-4 rounded-lg bg-[#FB1048]/15 flex items-center gap-4">
    <AlertTriangle className="w-4 h-4" />

    <p className="text-sm">{children}</p>
  </div>
);

export const WarnBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="border border-[#FFD55D]/15 text-[#FFD55D] p-4 rounded-lg bg-[#FFD55D]/15 flex items-center gap-4 text-sm">
    <AlertTriangle className="w-4 h-4" />

    {children}
  </div>
);
