"use client";

import type { PropsWithChildren } from "react";

type Props = {
  onClick: () => Promise<unknown>;
};
export const OAuthButton: React.FC<PropsWithChildren<Props>> = ({ onClick, children }) => {
  return (
    <button
      type="button"
      className="relative flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-white duration-500 border rounded-lg bg-white/10 hover:bg-white hover:text-black border-white/10"
      onClick={onClick}
    >
      {children}
    </button>
  );
};
