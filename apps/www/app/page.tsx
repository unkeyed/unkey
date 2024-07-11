import { Hero } from "@/app/hero";
import { AnalyticsBento } from "@/components/analytics/analytics-bento";
import { AuditLogsBento } from "@/components/audit-logs-bento";
import { PrimaryButton, SecondaryButton } from "@/components/button";
import { CTA } from "@/components/cta";
import { FeatureGrid } from "@/components/feature/feature-grid";
import { HashedKeysBento } from "@/components/hashed-keys-bento";
import { ImageWithBlur } from "@/components/image-with-blur";
import { IpWhitelistingBento } from "@/components/ip-whitelisting-bento";
import { LatencyBento } from "@/components/latency-bento";
import { OpenSource } from "@/components/open-source";
import { RateLimitsBento } from "@/components/rate-limits-bento";
import { Section, SectionTitle } from "@/components/section";
import { Stats } from "@/components/stats";
import { FeatureGridChip } from "@/components/svg/feature-grid-chip";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/hero";
import { OssLight } from "@/components/svg/oss-light";
import { UsageBento } from "@/components/usage-bento";
import { ChevronRight, LogIn } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Suspense } from "react";
import mainboard from "../images/mainboard.svg";
import { DesktopLogoCloud, MobileLogoCloud } from "./(components)/logo-cloud-content";
import { CodeExamples } from "./code-examples";

export const metadata = {
  title: "Unkey",
  description: "Build better APIs faster",
  openGraph: {
    title: "Unkey",
    description: "Build better APIs faster",
    url: "https://unkey.com/",
    siteName: "unkey.com",
    images: [
      {
        url: "https://unkey.com/og.png",
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
      <div className="relative w-full pt-6 overflow-hidden">
        <div className="container relative mx-auto">
          <Image
            src={mainboard}
            alt="Animated SVG showing computer circuits lighting up"
            className="absolute inset-x-0 flex  xl:hidden -z-10 scale-[2]"
            priority
          />
        </div>
        <div className="container relative flex flex-col mx-auto space-y-16 md:space-y-32">
          <Section>
            <Hero />
          </Section>
          {/* <Section className="mt-16 md:mt-32">
            <Suspense fallback={null}>
              <Stats />
            </Suspense>
          </Section> */}
          <Section className="mt-16 md:mt-32">
            <DesktopLogoCloud />
            <MobileLogoCloud />
          </Section>
          <Section className="mt-16 md:mt-18">
            <CodeExamples />
          </Section>
          <Section className="mt-16 md:mt-18">
            <OpenSource />
          </Section>

          <Section className="mt-16 md:mt-20">
            <SectionTitle
              className="mt-8 md:mt-16 lg:mt-32"
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
          </Section>
          <div className="relative w-full -z-10 ">
            <OssLight className="absolute scale-[2] left-[-70px] sm:left-[70px] md:left-[150px] lg:left-[200px] xl:left-[420px] top-[-250px]" />
          </div>

          <Section className="mt-16 md:mt-32">
            <SectionTitle
              title="Secure and scalable from day one"
              text="We give you crucial security features out of the box, so that you can focus on rapidly iterating on your API."
              align="center"
              label="Security"
            >
              <div className="flex mt-10 mb-10 space-x-6">
                <Link href="https://app.unkey.com" className="group">
                  <PrimaryButton shiny IconLeft={LogIn} label="Get Started" className="h-10" />
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

            <div className="relative grid md:grid-cols-[1fr_1fr] xl:grid-cols-[3fr_2fr] gap-6 z-50">
              {/* TODO: optimize to avoid fetching svg on mobile */}
              <div
                aria-hidden
                className="hidden lg:block pointer-events-none absolute top-[calc(100%-51px)] right-[226px] lg:right-[500px] aspect-[1400/541] w-[1400px]"
              >
                <ImageWithBlur
                  src="/images/landing/leveled-up-api-auth-chip-min.svg"
                  alt="Visual decoration auth chip"
                  fill
                />
              </div>

              <IpWhitelistingBento />
              <RateLimitsBento />
            </div>
          </Section>
          <Section className="mt-16 md:mt-32">
            <div className="relative">
              {/* TODO: horizontal scroll */}
              <SectionTitle
                className="mt-8 md:mt-16 lg:mt-32 xl:mt-48"
                title="Leveled-up API management"
                text="With enhanced security, low latency, and better control, you can seamlessly integrate into your APIs and protect your data like never before."
                label="Features"
              >
                <div className="flex mt-10 mb-10 space-x-6">
                  <Link href="https://app.unkey.com" className="group">
                    <PrimaryButton shiny IconLeft={LogIn} label="Get Started" className="h-10" />
                  </Link>

                  <Link href="/docs">
                    <SecondaryButton label="Visit the Docs" IconRight={ChevronRight} />
                  </Link>
                </div>
              </SectionTitle>
            </div>
            <FeatureGrid className="relative z-50 mt-20" />
            <div className="relative -z-10">
              <FeatureGridChip className="absolute top-[50px] left-[400px]" />
            </div>
          </Section>
          <Section className="mt-16 md:mt-32">
            <CTA />
          </Section>
        </div>
      </div>
    </>
  );
}
