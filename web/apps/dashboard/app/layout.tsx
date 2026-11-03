import { CommandMenu } from "@/components/dashboard/command-menu";
import { getAuth, toAuthenticatedUser } from "@/lib/auth/get-auth";
import { FlagProvider } from "@/lib/flags/provider";
import { loadFlags } from "@/lib/flags/server";
import { WorkspaceProvider } from "@/providers/workspace-provider";
import { Toaster } from "@unkey/ui";
import { GeistMono, GeistSans } from "./fonts/geist";
import "./fonts/geist.css";
import "@/styles/tailwind.css";
import * as Sentry from "@sentry/nextjs";
import { Analytics } from "@vercel/analytics/next";
import type { Metadata } from "next";
import dynamic from "next/dynamic";
import type React from "react";
import { Suspense } from "react";
import { AuthProvider } from "./auth-provider";
import { ReactQueryProvider } from "./react-query-provider";
import { ThemeProvider } from "./theme-provider";

export function generateMetadata(): Metadata {
  return {
    metadataBase: new URL("https://unkey.dev"),
    title: "Dashboard | Unkey",
    description: "Build better APIs faster",
    openGraph: {
      title: "Dashboard | Unkey",
      description: "Build better APIs faster ",
      url: "https://app.unkey.com",
      siteName: "app.unkey.com",
      images: ["https://www.unkey.com/og.png"],
    },
    twitter: {
      title: "Unkey",
      card: "summary_large_image",
      images: ["https://www.unkey.com/og.png"],
    },
    robots: {
      index: false,
      follow: false,
      nocache: false,
      googleBot: {
        index: false,
        follow: false,
        noimageindex: true,
        "max-video-preview": -1,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
    icons: {
      icon: [
        { url: "/favicon/favicon.ico" },
        {
          url: "/favicon/favicon-light.svg",
          media: "(prefers-color-scheme: light)",
          type: "image/svg+xml",
        },
        {
          url: "/favicon/favicon-dark.svg",
          media: "(prefers-color-scheme: dark)",
          type: "image/svg+xml",
        },
      ],
      shortcut: "/favicon/favicon.ico",
      apple: [{ url: "/apple-touch-icon.png", sizes: "180x180", type: "image/png" }],
    },
    other: {
      ...Sentry.getTraceData(),
    },
  };
}

const Feedback = dynamic(() =>
  import("@/components/dashboard/feedback-component").then((mod) => mod.Feedback),
);

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [flags, auth] = await Promise.all([loadFlags(), getAuth()]);
  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      <body className="min-h-full antialiased">
        <ReactQueryProvider key={`${auth.orgId}:${auth.userId}`}>
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <AuthProvider>
              <WorkspaceProvider initialUser={toAuthenticatedUser(auth)}>
                <FlagProvider initialFlags={flags}>
                  <Toaster />
                  {children}
                  <CommandMenu />
                  <Suspense fallback={null}>
                    <Feedback />
                  </Suspense>
                  <Analytics />
                </FlagProvider>
              </WorkspaceProvider>
            </AuthProvider>
          </ThemeProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}
