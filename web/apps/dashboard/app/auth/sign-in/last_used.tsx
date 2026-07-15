"use client";

import { useLocalStorage } from "@/hooks/use-local-storage";

type LastUsedProvider = "github" | "google" | "email";

const LAST_USED_LOGIN_KEY = "last_unkey_login";

export function useLastUsed() {
  return useLocalStorage<LastUsedProvider | undefined>(LAST_USED_LOGIN_KEY, undefined);
}

export function LastUsed() {
  // Straddles the top edge of the (relative) button it renders inside,
  // Vercel-style. bg-gray-3 is opaque so the button border doesn't show
  // through where the pill crosses it.
  return (
    <span className="absolute -top-2.5 right-3 rounded-full border px-2 py-0.5 text-[11px] font-medium leading-4 border-gray-6 bg-gray-3 text-gray-11">
      Last used
    </span>
  );
}
