"use client";
import { useLocalStorage } from "usehooks-ts";

export function useLastUsed() {
  return useLocalStorage<"github" | "google" | "email" | undefined>("last_unkey_login", undefined);
}

export const LastUsed: React.FC = () => {
  return <span className="absolute right-4 text-xs text-content-subtle">Last used</span>;
};
