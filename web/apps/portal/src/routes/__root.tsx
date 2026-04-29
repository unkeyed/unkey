import { HeadContent, Outlet, Scripts, createRootRoute } from "@tanstack/react-router";
/// <reference types="vite/client" />
import type { ReactNode } from "react";
import { TooltipProvider } from "~/components/ui/tooltip";
import "~/styles/tailwind.css";

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { title: "Customer Portal" },
      { name: "robots", content: "noindex, nofollow" },
    ],
    links: [
      // TODO: wire up customer favicon here
      { rel: "icon", href: "/favicon.svg" },
      {
        rel: "stylesheet",
        href: "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap",
      },
    ],
  }),
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <TooltipProvider delayDuration={300}>
        <Outlet />
      </TooltipProvider>
    </RootDocument>
  );
}

function RootDocument({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="en">
      <head>
        <HeadContent />
      </head>
      <body
        className="min-h-screen text-content antialiased"
        style={{ backgroundColor: "var(--portal-secondary, #f8fafc)" }}
      >
        {children}
        <Scripts />
      </body>
    </html>
  );
}
