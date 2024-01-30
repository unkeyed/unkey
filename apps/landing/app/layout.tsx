import { Footer } from "@/components/footer/footer";
import { Navigation } from "@/components/navbar/navigation";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Unkey",
  description: "Accelerate your API Development",
  openGraph: {
    title: "Unkey",
    description: "Accelerate your API Development",
    url: "https://unkey.dev/",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/unkey.png",
  },
};
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className="min-h-screen overflow-x-hidden antialiased bg-black">
        <Navigation />
        {children}
        <Footer />
      </body>
    </html>
  );
}
