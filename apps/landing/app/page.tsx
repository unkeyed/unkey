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
  HeroMainboardStuff,
  HeroMainboardStuffMobile,
  SubHeroMainboardStuff,
  TopLeftShiningLight,
  TopRightShiningLight,
} from "@/components/svg/hero";
import { LeveledUpApiAuthChip } from "@/components/svg/leveled-up-api-auth-chip";
import { OssLight } from "@/components/svg/oss-light";
import { UsageBento } from "@/components/usage-bento";
import { ChevronRight, LogIn } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";
import { CodeExamples } from "./code-examples";

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

export const revalidate = 300;

export default async function Landing() {
  return (
    <div className="container mx-auto relative">
      <TopLeftShiningLight />
      <TopRightShiningLight />
      <HeroMainboardStuff className="absolute right-0 -top-[4%]" />
      <HeroMainboardStuffMobile className="absolute right-0 -top-[5%]" />

      <Hero />

      <SubHeroMainboardStuff className="w-full absolute bottom-[-50px] left-[250px] pointer-events-none" />
      <div className="mt-[200px]" />
      <Suspense fallback={null}>
        <Stats />
      </Suspense>

      <CodeExamples className="mt-20" />
      <div className="mt-[220px]" />
      <OpenSource />
      <SectionTitle
        className="mt-[300px]"
        title="Efficient integration and process, always"
        titleWidth={743}
        contentWidth={641}
        text="Elevate operations effortlessly with our platform - seamless processes, reliable analytics, and billing ensure unparalleled efficiency and accuracy for all your integrated tasks and workflows"
        align="center"
        label="Platform"
      />
      <AnalyticsBento />
      <div className="mt-6 grid xl:grid-cols-[1fr_2fr] gap-6 z-50">
        <LatencyBento />
        <UsageBento />
      </div>
      <div className="relative w-full -z-10">
        <OssLight className="absolute left-[-70px] sm:left-[70px] md:left-[150px] lg:left-[200px] xl:left-[400px] top-[-200px]" />
      </div>
      <SectionTitle
        className="mt-[300px]"
        titleWidth={743}
        contentWidth={581}
        title="Secure from day one"
        text="Don’t waste time building boring-but-necessary features like audit logs. Unkey provides everything you need out of the box to build secure, infinitely scalable APIs."
        align="center"
        label="Protection"
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
      <div className="grid xl:grid-cols-[3fr_2fr] gap-6 relative z-50">
        <IpWhitelistingBento />
        <RateLimitsBento />
      </div>
      <div className="relative">
        <LeveledUpApiAuthChip className="absolute top-[-450px] right-0" />
        <SectionTitle
          className="mt-[400px] md:ml-10"
          title="Leveled-up API Auth"
          titleWidth={719}
          contentWidth={557}
          text="Elevate your API authentication with our leveled-up system. Experience heightened security, efficiency, and control for seamless integration and data protection."
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
      <FeatureGrid className="mt-20 relative z-50" />
      <div className="relative -z-10">
        <FeatureGridChip className="absolute top-[-90px]" />
      </div>
      <SectionTitle
        align="center"
        className="mt-[200px]"
        title="Protect your API. Start today."
        titleWidth={507}
      >
        <div className="flex space-x-6 ">
          <Link
            key="get-started"
            href="/app"
            className="flex items-center h-10 gap-2 px-4 font-medium text-black duration-150 bg-white border border-white rounded-lg shadow-md hover:text-white hover:bg-black"
          >
            Start Now <ChevronRight className="w-4 h-4" />
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
  );
}
