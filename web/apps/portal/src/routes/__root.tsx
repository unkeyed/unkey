import { HeadContent, Outlet, Scripts, createRootRoute } from "@tanstack/react-router";
/// <reference types="vite/client" />
import type { ReactNode } from "react";
import { ReactQueryProvider } from "~/providers/react-query-provider";
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
      {
        rel: "stylesheet",
        href: "https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700&family=Geist+Mono&display=swap",
      },
    ],
  }),
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <ReactQueryProvider>
        <Outlet />
      </ReactQueryProvider>
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
