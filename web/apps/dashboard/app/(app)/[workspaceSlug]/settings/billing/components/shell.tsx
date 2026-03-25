"use client";
import type { PropsWithChildren } from "react";

export const Shell = ({ children }: PropsWithChildren) => {
  return (
    <main className="w-225 flex flex-col justify-center items-center gap-6 mx-auto my-14">
      {children}
    </main>
  );
};
