import { RootProvider } from "fumadocs-ui/provider";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";

import type { ReactNode } from "react";

import { TooltipProvider } from "@unkey/ui/src/components/tooltip";
import "./global.css";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      <body>
        <RootProvider>
          <TooltipProvider>{children}</TooltipProvider>
        </RootProvider>
      </body>
    </html>
  );
}
