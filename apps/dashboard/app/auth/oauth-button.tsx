"use client";

import type { ComponentProps } from "react";

type ButtonElementProps = ComponentProps<"button">;

export const OAuthButton: React.FC<ButtonElementProps> = ({ onClick, disabled, children }) => {
  return (
    <button
      type="button"
      disabled={disabled}
      className="relative flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-white duration-500 border rounded-lg bg-white/10 enabled:hover:bg-white enabled:hover:text-black border-white/10 disabled:opacity-50 disabled:cursor-not-allowed"
      onClick={onClick}
    >
      {children}
    </button>
  );
};
