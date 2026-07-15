"use client";

import { useLocalStorage } from "@/hooks/use-local-storage";

type LastUsedProvider = "github" | "google" | "email";

const LAST_USED_LOGIN_KEY = "last_unkey_login";

export function useLastUsed() {
  return useLocalStorage<LastUsedProvider | undefined>(LAST_USED_LOGIN_KEY, undefined);
}

export function LastUsed() {
  // Sits inside the right edge of the (relative) button it renders in,
  // vertically centered.
  return (
    <span className="absolute right-3 top-1/2 -translate-y-1/2 rounded-full border px-2 py-0.5 text-[11px] font-medium leading-4 border-gray-6 bg-gray-3 text-gray-11">
      Last used
    </span>
  );
}
