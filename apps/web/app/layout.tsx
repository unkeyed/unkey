/// <reference lib="dom" />
/// <reference lib="dom.iterable" />
// ^ https://bun.sh/docs/typescript#dom-types

import { PHProvider, PostHogPageview } from "@/providers/PostHogProvider";
import "@/styles/tailwind/tailwind.css";
import type { Metadata } from "next";
import { Inter } from "next/font/google";
import localFont from "next/font/local";
import type React from "react";
import { Suspense } from "react";
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
  description: "Accelerate your API development",
  openGraph: {
    title: "Open Source API Authentication",
    description: "Accelerate your API development ",
    url: "https://unkey.dev",
    siteName: "unkey.dev",
    images: ["https://unkey.dev/og.png"],
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
    images: ["https://unkey.dev/og.png"],
  },
  robots: {
    index: true,
    follow: true,
    nocache: true,
    googleBot: {
      index: true,
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
          <head>
            {(process.env.NODE_ENV === "development" || process.env.VERCEL_ENV === "preview") && (
              // eslint-disable-next-line @next/next/no-sync-scripts
              <script
                data-project-id="LgLkKSvYgWMJ231LDMvubknwDBTikeK7C7y9FEtb"
                data-is-production-environment="false"
                src="https://snippet.meticulous.ai/v1/meticulous.js"
              />
            )}
          </head>
          {children}
        </body>
      </PHProvider>
    </html>
  );
}
