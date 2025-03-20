import { Footer } from "@/components/footer/footer";
import { Navigation } from "@/components/navbar/navigation";
import { env } from "@/lib/env";
import { PHProvider } from "@/providers/posthog/PostHogProvider";
import "@unkey/ui/css";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";
import dynamic from "next/dynamic";
import "./globals.css";

const PostHogPageView = dynamic(() => import("@/providers/posthog/PostHogPageView"), {
  ssr: false,
});

const parsedEnv = env();

export const metadata: Metadata = {
  metadataBase: new URL(parsedEnv.NEXT_PUBLIC_BASE_URL),
  title: "Unkey",
  description: "Build better APIs faster",
  openGraph: {
    title: "Unkey",
    description: "Build better APIs faster",
    url: parsedEnv.NEXT_PUBLIC_BASE_URL,
    siteName: "unkey.com",
    images: [
      {
        url: `${parsedEnv.NEXT_PUBLIC_BASE_URL}/og.png`,
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
    <html
      lang="en"
      className={`[color-scheme:dark] scroll-smooth ${GeistSans.variable} ${GeistMono.variable}`}
    >
      <PHProvider>
        <body className="min-h-screen overflow-x-hidden antialiased bg-black text-pretty">
          <PostHogPageView />
          <div className="relative overflow-x-clip">
            <Navigation />
            {children}
            {process.env.NODE_ENV !== "production" ? (
              <div className="fixed bottom-0 right-0 flex items-center justify-center w-6 h-6 p-3 m-8 font-mono text-xs text-black bg-white rounded-lg pointer-events-none ">
                <div className="block sm:hidden md:hidden lg:hidden xl:hidden 2xl:hidden">al</div>
                <div className="hidden sm:block md:hidden lg:hidden xl:hidden 2xl:hidden">sm</div>
                <div className="hidden sm:hidden md:block lg:hidden xl:hidden 2xl:hidden">md</div>
                <div className="hidden sm:hidden md:hidden lg:block xl:hidden 2xl:hidden">lg</div>
                <div className="hidden sm:hidden md:hidden lg:hidden xl:block 2xl:hidden">xl</div>
                <div className="hidden sm:hidden md:hidden lg:hidden xl:hidden 2xl:block">2xl</div>
              </div>
            ) : null}
          </div>
          <Footer />
        </body>
      </PHProvider>
    </html>
  );
}
