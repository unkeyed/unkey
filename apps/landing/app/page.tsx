import { Hero } from "@/app/hero";
import { SectionTitle } from "@/app/section-title";
import { AnalyticsBento } from "@/components/analytics/analytics-bento";
import { AuditLogsBento } from "@/components/audit-logs-bento";
import { PrimaryButton, SecondaryButton } from "@/components/button";
import { FeatureGrid } from "@/components/feature/feature-grid";
import { HashedKeysBento } from "@/components/hashed-keys-bento";
import { IpWhitelistingBento } from "@/components/ip-whitelisting-bento";
import { LatencyBento } from "@/components/latency-bento";
import { OpenSource } from "@/components/open-source";
import { RateLimitsBento } from "@/components/rate-limits-bento";
import { Stats } from "@/components/stats";
import { FeatureGridChip } from "@/components/svg/feature-grid-chip";
import {
  SubHeroMainboardStuff,
  TopLeftShiningLight,
  TopRightShiningLight,
} from "@/components/svg/hero";
import { LeveledUpApiAuthChip } from "@/components/svg/leveled-up-api-auth-chip";
import { OssLight } from "@/components/svg/oss-light";
import { UsageBento } from "@/components/usage-bento";
import { ChevronRight, LogIn } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Suspense } from "react";
import mainboardMobile from "../images/mainboard-mobile.svg";
import mainboard from "../images/mainboard.svg";
import { CodeExamples } from "./code-examples";

export const metadata = {
  title: "Unkey",
  description: "Build better APIs faster",
  openGraph: {
    title: "Unkey",
    description: "Build better APIs faster",
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

export const dynamic = "error";
export const revalidate = 300;

export default async function Landing() {
  return (
    <>
      <TopRightShiningLight />
      <TopLeftShiningLight />
      <Image
        src={mainboard}
        alt="Animated SVG showing computer circuits lighting up"
        className="hidden md:flex w-full absolute right-0 top-[-140px] -z-10"
        priority={true}
      />

      <Image
        src={mainboardMobile}
        alt="Animated SVG showing computer circuits lighting up"
        className="flex md:hidden w-full absolute h-[300px] -z-10 "
        priority={true}
      />
      <div className="container relative mx-auto">
        <Hero />
        <SubHeroMainboardStuff className="w-full absolute bottom-[-50px] left-[250px] pointer-events-none" />
        <div className="mt-[200px]" />
        <Suspense fallback={null}>
          <Stats />
        </Suspense>

        <CodeExamples className="mt-[144px] md:mt-[120px]" />
        <div className="mt-[220px]" />
        <OpenSource />
        <SectionTitle
          className="mt-[300px]"
          title="Everything you need for your API"
          text="Our platform simplifies the API-building process, allowing you to monetize, analyze, and protect endpoints."
          align="center"
          label="Platform"
        />
        <AnalyticsBento />
        <div className="mt-6 grid md:grid-cols-[1fr_1fr] lg:grid-cols-[1fr_2fr] gap-6 z-50">
          <LatencyBento />
          <UsageBento />
        </div>
        <div className="relative w-full -z-10">
          <OssLight className="absolute left-[-70px] sm:left-[70px] md:left-[150px] lg:left-[200px] xl:left-[400px] top-[-200px]" />
        </div>
        <SectionTitle
          className="mt-[300px]"
          title="Secure and scalable from day one"
          text="We give you crucial security features out of the box, so that you can focus on rapidly iterating on your API."
          align="center"
          label="Security"
        >
          <div className="flex mt-10 mb-10 space-x-6">
            <Link href="/app" className="group">
              <PrimaryButton IconLeft={LogIn} label="Get Started" className="h-10" />
            </Link>

            <Link href="/docs">
              <SecondaryButton label="Visit the Docs" IconRight={ChevronRight} />
            </Link>
          </div>
        </SectionTitle>
        <div className="grid xl:grid-cols-[2fr_3fr] gap-6">
          <HashedKeysBento />
          <AuditLogsBento />
        </div>
        <div className="grid md:grid-cols-[1fr_1fr] xl:grid-cols-[3fr_2fr] gap-6 relative z-50">
          <IpWhitelistingBento />
          <RateLimitsBento />
        </div>
        <div className="relative">
          {/* TODO: horizontal scroll */}
          <LeveledUpApiAuthChip className="absolute top-[-450px] right-[-100px]" />
          <SectionTitle
            className="mt-[400px] md:ml-10"
            title="Leveled-up API management"
            text="With enhanced security, low latency, and better control, you can seamlessly integrate into your APIs and protect your data like never before."
            label="More"
          >
            <div className="flex mt-10 mb-10 space-x-6">
              <Link href="/app" className="group">
                <PrimaryButton IconLeft={LogIn} label="Get Started" className="h-10" />
              </Link>

              <Link href="/docs">
                <SecondaryButton label="Visit the Docs" IconRight={ChevronRight} />
              </Link>
            </div>
          </SectionTitle>
        </div>
        <FeatureGrid className="relative z-50 mt-20" />
        <div className="relative -z-10">
          <FeatureGridChip className="absolute top-[-90px]" />
        </div>
        <SectionTitle align="center" className="mt-[200px]" title="Protect your API. Start today.">
          <div className="flex space-x-6 ">
            <Link key="get-started" href="/app">
              <PrimaryButton label="Start Now" IconRight={ChevronRight} />
            </Link>
          </div>
        </SectionTitle>
        <div className="mt-10 mb-[200px]">
          <p className="w-full mx-auto text-sm leading-6 text-center text-white/60">
            2500 verifications FREE per month.
          </p>
          <p className="w-full mx-auto text-sm leading-6 text-center text-white/60">
            No CC required.
          </p>
        </div>
      </div>
    </>
  );
}
