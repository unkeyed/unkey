import Link from "next/link";

import { PrimaryButton, RainbowDarkButton, SecondaryButton } from "@/components/button";
import { ArrowRight, BookOpen, ChevronRight, LogIn } from "lucide-react";

export function HeroMainSection() {
  return (
    <div className="relative flex flex-col items-center text-center xl:text-left xl:items-start">
      <Link href="/accelerate" target="">
        <RainbowDarkButton
          className="mb-6"
          label="Launch Week: June 24-29"
          IconRight={ArrowRight}
        />
      </Link>
      <h1 className="bg-gradient-to-br text-pretty text-transparent bg-gradient-stop bg-clip-text from-white via-white via-30% to-white/30 max-w-sm sm:max-w-none xl:max-w-lg font-medium text-[32px] leading-none sm:text-[56px] md:text-[64px] xl:text-[64px] tracking-tighter">
        Build better APIs faster
      </h1>

      <p className="mt-6 sm:mt-8 bg-gradient-to-br text-transparent text-balance bg-gradient-stop bg-clip-text max-w-sm sm:max-w-lg xl:max-w-md from-white/70 via-white/70 via-40% to-white/30 text-sm sm:text-base">
        Redefined API management for developers. Quickly add API keys, rate limiting, and usage
        analytics to your API at any scale.
      </p>

      <div className="flex items-center gap-6 mt-16">
        <Link href="https://app.unkey.dev" className="group">
          <PrimaryButton shiny IconLeft={LogIn} label="Get started" className="h-10" />
        </Link>

        <Link href="/docs" className="hidden sm:flex">
          <SecondaryButton IconLeft={BookOpen} label="Documentation" IconRight={ChevronRight} />
        </Link>
      </div>
    </div>
  );
}
