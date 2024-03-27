import Link from "next/link";

import { PrimaryButton, RainbowDarkButton, SecondaryButton } from "@/components/button";
import { ArrowRight, BookOpen, ChevronRight, LogIn } from "lucide-react";

export function HeroMainSection() {
  return (
    <div className="relative flex flex-col items-center text-center xl:text-left xl:items-start">
      <Link href="/careers" target="">
        <RainbowDarkButton className="mb-4" label="We are hiring!" IconRight={ArrowRight} />
      </Link>

      <h1 className="bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white max-w-[546px] via-30% to-white/30 font-medium text-[32px] leading-[48px]  sm:text-[56px] sm:leading-[72px] md:text-[64px] md:leading-[80px] xl:text-[64px] xl:leading-[80px]  ">
        Build your API, not auth
      </h1>

      <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg text-sm sm:text-[15px] sm:text-base leading-[28px]">
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
