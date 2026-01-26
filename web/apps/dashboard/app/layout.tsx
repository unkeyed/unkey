import { CommandMenu } from "@/components/dashboard/command-menu";
import { WorkspaceProvider } from "@/providers/workspace-provider";
import { Toaster } from "@unkey/ui";
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';
import "@/styles/tailwind/tailwind.css";
import "@unkey/ui/css";
import * as Sentry from "@sentry/nextjs";
import type { Metadata } from "next";
import dynamic from "next/dynamic";
import type React from "react";
import { Suspense } from "react";
import { ReactQueryProvider } from "./react-query-provider";
import { ThemeProvider } from "./theme-provider";

export function generateMetadata(): Metadata {
  return {
    metadataBase: new URL("https://unkey.dev"),
    title: "Open Source API Authentication",
    description: "Build better APIs faster",
    openGraph: {
      title: "Open Source API Authentication",
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
    other: {
      ...Sentry.getTraceData(),
    },
  };
}

const Feedback = dynamic(() =>
  import("@/components/dashboard/feedback-component").then((mod) => mod.Feedback),
);



export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {

 
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`} suppressHydrationWarning>
      <body className="min-h-full antialiased">
        <ReactQueryProvider>
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <WorkspaceProvider>
              <Toaster />
              {children}
              <CommandMenu />
              <Suspense fallback={null}>
                <Feedback />
              </Suspense>
            </WorkspaceProvider>
          </ThemeProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}
