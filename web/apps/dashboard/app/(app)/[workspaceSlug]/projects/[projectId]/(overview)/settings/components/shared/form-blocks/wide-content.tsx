import type React from "react";

export const WideContent = ({ children }: { children: React.ReactNode }) => (
  <div data-form-wide className="flex flex-col gap-1.5">
    {children}
  </div>
);
