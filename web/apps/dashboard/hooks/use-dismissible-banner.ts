"use client";

import { useEffect, useState } from "react";

export function useDismissibleBanner(key: string) {
  const storageKey = `unkey-banner-dismissed-${key}`;
  const [dismissed, setDismissed] = useState(true);

  useEffect(() => {
    setDismissed(localStorage.getItem(storageKey) === "true");
  }, [storageKey]);

  const dismiss = () => {
    localStorage.setItem(storageKey, "true");
    setDismissed(true);
  };

  return { dismissed, dismiss };
}
