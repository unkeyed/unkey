import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";

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
    <html lang="en" className={GeistSans.className}>
      <body>
        <div className="bg-gradient-radial mx-auto min-w-[800px] mini-h-[800px]" />
        {children}
      </body>
    </html>
  );
}
