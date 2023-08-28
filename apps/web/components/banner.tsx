"use client";

import React, { PropsWithChildren, useState } from "react";

import { cn } from "@/lib/utils";
import { X } from "lucide-react";

type Props = {
  variant?: "alert";
};

export const Banner: React.FC<PropsWithChildren<Props>> = ({ children, variant }) => {
  const [visible, setVisible] = useState(true);
  if (!visible) {
    return null;
  }
  return (
    <div className="fixed inset-x-0 bottom-0 z-50 pointer-events-none sm:flex sm:justify-center sm:px-6 sm:pb-5 lg:px-8">
      <div
        className={cn(
          "pointer-events-auto flex items-center justify-between gap-x-6  px-6 py-2.5 sm:rounded-lg sm:py-3 sm:pl-4 sm:pr-3.5",
          {
            "bg-primary text-primary-foreground": variant === undefined,
            "bg-destructive text-destructive-foreground": variant === "alert",
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
