"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";

// TODO(deploy): REMOVE BEFORE MERGE — temporary chooser for the projects/onboarding artwork.
// Unlike the dev-only ASCII tune panel (?asciitune), this is intentionally NOT gated by env so
// we can flip between the animated ASCII look and the classic bordered-box look on a deploy and
// pick one. Choice is persisted in localStorage and synced across components on the page.
export type ArtStyle = "ascii" | "classic";

const STORAGE_KEY = "unkey-deploy-art-style";
const CHANGE_EVENT = "unkey-deploy-art-style-change";

export function useArtStyle(): [ArtStyle, (next: ArtStyle) => void] {
  // Default to "ascii" on the server and first paint; the stored choice is applied after mount
  // to avoid a hydration mismatch.
  const [style, setStyleState] = useState<ArtStyle>("ascii");

  useEffect(() => {
    const read = () => {
      const v = localStorage.getItem(STORAGE_KEY);
      if (v === "ascii" || v === "classic") {
        setStyleState(v);
      }
    };
    read();
    window.addEventListener(CHANGE_EVENT, read);
    return () => window.removeEventListener(CHANGE_EVENT, read);
  }, []);

  const setStyle = (next: ArtStyle) => {
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // ignore unavailable/blocked storage
    }
    setStyleState(next);
    window.dispatchEvent(new Event(CHANGE_EVENT));
  };

  return [style, setStyle];
}

const OPTIONS: { value: ArtStyle; label: string }[] = [
  { value: "ascii", label: "ASCII" },
  { value: "classic", label: "Classic" },
];

export function ArtStyleSwitcher({ className }: { className?: string }) {
  const [style, setStyle] = useArtStyle();

  return (
    <div
      className={cn(
        "fixed bottom-4 right-4 z-50 flex items-center gap-1 rounded-lg border border-grayA-4 bg-gray-2 p-1 shadow-sm",
        className,
      )}
    >
      <span className="px-2 text-[11px] uppercase tracking-wide text-gray-9">Art</span>
      {OPTIONS.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => setStyle(o.value)}
          className={cn(
            "rounded-md px-3 py-1 text-[13px] transition-colors",
            style === o.value
              ? "bg-gray-4 text-gray-12"
              : "text-gray-10 hover:bg-gray-3 hover:text-gray-12",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
