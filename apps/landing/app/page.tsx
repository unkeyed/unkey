import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { CodeExamples } from "./code-examples";
import { SectionHeader } from "./section";

import { DesktopNav, Hero, MobileNav } from "@/components";
import {
  HeroMainboard,
  Logo,
  SubHeroMainboard,
  TopLeftShiningLight,
  TopRightShiningLight,
} from "@/components/svg";

export const metadata = {
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

export default async function Landing() {
  return (
    <div className="container mx-auto antialiased">
      <DesktopNav />
      <MobileNav />
      {/* <TopLeftShiningLight />
      <TopRightShiningLight />

      <Navigation />

      <HeroMainboard className="absolute top-0 right-0" />
      <Hero />
      <SubHeroMainboard />

      <SectionHeader
        tag="Code"
        title="Any Language, any Framework, always secure!"
        subtitle="Unkey ensures security across any language or framework. Effortlessly manage API Keys with an intuitive console, providing timely data and streamlined settings for a seamless coding experience."
        actions={[
          <Link
            key="get-started"
            href="/app"
            className="h-10 shadow-md font-medium bg-white flex items-center border border-white px-4  rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
          >
            Get Started <ChevronRight className="w-4 h-4" />
          </Link>,
          <Link
            key="docs"
            href="/docs"
            className="h-10 flex items-center px-4 gap-2 text-white/50 hover:text-white duration-500"
          >
            Visit the docs <ChevronRight className="w-4 h-4" />
          </Link>,
        ]}
        align="center"
      />

      <CodeExamples className="mt-20" /> */}
    </div>
  );
}
