import { CommandMenu } from "@/components/dashboard/command-menu";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { PHProvider, PostHogPageview } from "@/providers/PostHogProvider";
import "@/styles/tailwind/tailwind.css";
import { ClerkProvider } from "@clerk/nextjs";
import type { Metadata } from "next";
import { Inter } from "next/font/google";
import localFont from "next/font/local";
import type React from "react";
import { Suspense } from "react";
import { ReactQueryProvider } from "./react-query-provider";
import { ThemeProvider } from "./theme-provider";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
});

const pangea = localFont({
  src: "../public/fonts/PangeaAfrikanTrial-Medium.woff2",
  variable: "--font-pangea",
});

export const metadata = {
  metadataBase: new URL("https://unkey.dev"),
  title: "Open Source API Authentication",
  description: "Build better APIs faster",
  openGraph: {
    title: "Open Source API Authentication",
    description: "Build better APIs faster ",
    url: "https://unkey.dev",
    siteName: "unkey.dev",
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
} satisfies Metadata;

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={[inter.variable, pangea.variable].join(" ")}>
      <Suspense>
        <PostHogPageview />
      </Suspense>
      <PHProvider>
        <body>
          <Toaster />
          <ClerkProvider
            afterSignInUrl="/"
            afterSignUpUrl="/new"
            appearance={{
              variables: {
                colorPrimary: "#5C36A3",
                colorText: "#5C36A3",
              },
            }}
          >
            <ReactQueryProvider>
              <ThemeProvider attribute="class">
                <TooltipProvider>
                  {children}
                  <CommandMenu />
                </TooltipProvider>
              </ThemeProvider>
            </ReactQueryProvider>
          </ClerkProvider>
        </body>
      </PHProvider>
    </html>
  );
}
