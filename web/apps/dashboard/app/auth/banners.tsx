"use client";

import { TriangleWarning } from "@unkey/icons";
import type React from "react";
import { type PropsWithChildren, useEffect, useState } from "react";
import { createPortal } from "react-dom";

// Banner state lives deep inside the card content (pages/challenges), but the
// design wants banners displayed above the card. Portal them into the slot
// rendered by app/auth/layout.tsx so call sites stay next to their state.
const BANNER_SLOT_ID = "auth-banner-slot";

function BannerPortal({ children }: PropsWithChildren) {
  const [slot, setSlot] = useState<HTMLElement | null>(null);

  useEffect(() => {
    setSlot(document.getElementById(BANNER_SLOT_ID));
  }, []);

  if (!slot) {
    return null;
  }
  return createPortal(children, slot);
}

export const ErrorBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <BannerPortal>
    <div className="mb-4 border border-error-6 text-error-11 p-4 rounded-lg bg-error-3">
      <p className="text-sm">{children}</p>
    </div>
  </BannerPortal>
);

export const WarnBanner: React.FC<PropsWithChildren> = ({ children }) => (
  <BannerPortal>
    <div className="mb-4 border border-warning-6 text-warning-11 p-4 rounded-lg bg-warning-3 flex items-center gap-4 text-sm">
      <TriangleWarning className="w-4 h-4" />

      {children}
    </div>
  </BannerPortal>
);
