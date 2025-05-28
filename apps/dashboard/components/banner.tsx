"use client";

import type React from "react";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";
import { useLocalStorage, useResizeObserver } from "usehooks-ts";

import { cn } from "@/lib/utils";
import { X } from "lucide-react";

type Props = {
  variant?: "alert";
  /**
   * Annoying banners will come back after refreshing the page
   *
   * set the key in localstorage to use
   *
   * @example:
   * ```ts
   * <Banner persistChoice="billing-notification">...</Banner>
   */
  persistChoice?: string;
};

export const Banner: React.FC<PropsWithChildren<Props>> = ({
  children,
  variant,
  persistChoice,
}) => {
  const bannerHeightRef = useRef<number>(0);
  const bannerRef = useRef<HTMLDivElement>(null);

  const [visible, setVisible] = persistChoice
    ? useLocalStorage(`unkey_banner_${persistChoice}`, true, { initializeWithValue: false })
    : useState(true);

  useResizeObserver({
    ref: bannerRef,
    onResize: ({ height }) => {
      if (height !== bannerHeightRef.current) {
        // Update Banners.
        updateBannersBottomPosition();

        // Set current banner height.
        bannerHeightRef.current = height ?? 0;
      }
    },
  });

  useEffect(() => {
    if (visible && bannerRef.current) {
      updateBannersBottomPosition();

      bannerHeightRef.current = bannerRef.current.getBoundingClientRect().height;
    }
  }, [visible]);

  if (!visible) {
    return null;
  }

  return (
    <div
      ref={bannerRef}
      style={{ opacity: 0 }}
      className="Unkey__Banner fixed inset-x-0 bottom-0 z-50 pointer-events-none sm:flex sm:justify-center sm:px-6 sm:mb-5 lg:px-8"
    >
      <div
        className={cn(
          "pointer-events-auto flex items-center justify-between gap-x-6 px-6 py-2.5 sm:rounded-lg sm:py-3 sm:pl-4 sm:pr-3.5",
          {
            "bg-primary text-primary-foreground": variant === undefined,
            "bg-alert text-alert-foreground": variant === "alert",
          },
        )}
      >
        {children}
        <button type="button" className="-m-1.5 flex-none p-1.5" onClick={() => setVisible(false)}>
          <span className="sr-only">Close</span>
          <X className="w-5 h-5 " aria-hidden="true" />
        </button>
      </div>
    </div>
  );
};

/**
 * Retrieves all the <Banner /> elements and re-orders their bottom fixed positions.
 */
function updateBannersBottomPosition() {
  const banners = document.getElementsByClassName("Unkey__Banner");

  let yOffset = 0;

  // Must iterate backwards.
  for (let i = banners.length - 1; i >= 0; i--) {
    banners[i].setAttribute("style", `bottom:${yOffset}px;`);
    yOffset += banners[i].getBoundingClientRect().height + 6;
  }
}
