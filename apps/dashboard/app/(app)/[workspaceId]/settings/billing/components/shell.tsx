"use client";
import type { PropsWithChildren } from "react";

export const Shell: React.FC<PropsWithChildren<{ workspace: { id: string; name: string } }>> = ({
  children,
}) => {
  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          Billing Settings
        </div>
        {children}
      </div>
    </div>
  );
};
