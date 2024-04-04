import Link from "next/link";

import { PrimaryButton, RainbowDarkButton, SecondaryButton } from "@/components/button";
import { ArrowRight, BookOpen, ChevronRight, LogIn } from "lucide-react";

export function HeroMainSection() {
  return (
    <div className="relative flex flex-col items-center text-center xl:text-left xl:items-start">
      <Link href="https://unkey.dev/blog/introducing-ratelimiting" target="">
        <RainbowDarkButton
          className="mb-4"
          label="New: global rate limiting"
          IconRight={ArrowRight}
        />
      </Link>
      <h1 className="bg-gradient-to-br text-pretty text-transparent bg-gradient-stop bg-clip-text from-white via-white max-w-sm sm:max-w-md via-30% to-white/30 font-medium text-[32px] leading-[48px]  sm:text-[56px] sm:leading-[72px] md:text-[64px] md:leading-[80px] xl:text-[64px] xl:leading-[80px]  ">
        Build better APIs faster
      </h1>

      <p className="mt-8 bg-gradient-to-br text-transparent text-pretty bg-gradient-stop bg-clip-text max-w-sm sm:max-w-md  from-white via-white via-40% to-white/30 md:max-w-lg text-sm sm:text-base">
        Redefined API management for developers. Quickly add API keys, rate limiting, and usage
        analytics to your API at any scale.
      </p>

      <div className="flex items-center gap-6 mt-16">
        <Link href="/app" className="group">
          <PrimaryButton IconLeft={LogIn} label="Get started" className="h-10" />
        </Link>

        <Link href="/docs" className="hidden sm:flex">
          <SecondaryButton IconLeft={BookOpen} label="Documentation" IconRight={ChevronRight} />
        </Link>
      </div>
    </div>
  );
}
