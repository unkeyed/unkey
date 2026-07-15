"use client";

import type { PropsWithChildren } from "react";

type Props = {
  onClick: () => Promise<unknown>;
};

export function OAuthButton({ onClick, children }: PropsWithChildren<Props>) {
  return (
    <button
      type="button"
      className="relative flex w-full items-center justify-center h-10 gap-2 px-4 text-sm font-medium rounded-lg border cursor-pointer text-gray-12 bg-gray-2 border-gray-6 transition-colors hover:bg-gray-3 hover:border-gray-7"
      onClick={onClick}
    >
      {children}
    </button>
  );
}
