"use client";

import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/lib/hooks/use-sidebar";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import type { ThemeProviderProps } from "next-themes/dist/types";
import * as React from "react";

export function Providers({ children, ...props }: ThemeProviderProps) {
  return (
    <NextThemesProvider {...props}>
      <SidebarProvider>
        <TooltipProvider>{children}</TooltipProvider>
      </SidebarProvider>
    </NextThemesProvider>
  );
}
