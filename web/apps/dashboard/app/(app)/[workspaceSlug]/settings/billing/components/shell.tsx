"use client";
import type { PropsWithChildren } from "react";

export const Shell = ({ children }: PropsWithChildren) => {
  return (
    <main className="w-full py-3 flex justify-center">
      <div className="w-full max-w-[900px] mx-6 flex flex-col items-center gap-5 mt-4">
        {children}
      </div>
    </main>
  );
};
