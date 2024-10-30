import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";
import { CSPostHogProvider } from "./providers";

import { Toaster } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";
import "./globals.css";

export const metadata: Metadata = {
  title: "Unkey Playground",
  description: "Playground for Unkey API",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={cn("dark", GeistSans.className)}>
      <CSPostHogProvider>
        <body className="w-full bg-black text-[#E2E2E2]">
          {children}

          <Toaster duration={7_000} />
        </body>
      </CSPostHogProvider>
    </html>
  );
}
