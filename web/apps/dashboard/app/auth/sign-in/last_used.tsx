"use client";

import { useLocalStorage } from "@/hooks/use-local-storage";

type LastUsedProvider = "github" | "google" | "email";

const LAST_USED_LOGIN_KEY = "last_unkey_login";

export function useLastUsed() {
  return useLocalStorage<LastUsedProvider | undefined>(LAST_USED_LOGIN_KEY, undefined);
}

export const LastUsed = () => {
  return <span className="absolute right-4 text-xs text-content-subtle">Last used</span>;
};
