"use client";
import type { PropsWithChildren } from "react";

interface ShellProps extends PropsWithChildren {
  title?: string;
}

export const Shell = ({ children, title = "Billing Settings" }: ShellProps) => {
  return (
    <main className="w-full py-3 flex justify-center">
      <div className="w-full max-w-[900px] mx-6 flex flex-col items-center gap-5">
        <h1 className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          {title}
        </h1>
        {children}
      </div>
    </main>
  );
};
