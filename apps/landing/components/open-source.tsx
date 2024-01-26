import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { GithubSvg } from "./svg/github";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export function OpenSource() {
  return (
    <div className="pt-[240px] flex items-center flex-col md:flex-row justify-center relative">
      {/* TODO: add additional line SVGs from Figma – current export is broken */}
      <div className="absolute top-[-260px] md:right-[240px] z-[-1]">
        <OssLight />
      </div>
      <div className="flex-col xl:flex-row flex items-center">
        <div className="xl:pr-24 flex flex-col items-center xl:items-start">
          <p className="font-mono text-white/50 text-center xl:text-left">Open-source</p>
          <h1 className="text-[28px] leading-9 md:leading-[64px] md:text-[52px] text-white md:max-w-[463px] pt-4 section-title-heading-gradient text-center xl:text-left">
            Empowering the community
          </h1>
          <p className="text-white leading-7 max-w-[461px] pt-[26px] text-center xl:text-left">
            Unkey allows open-source contributions through Github, enabing collaboration and
            knowledge sharing with all the developers in the world.
          </p>
          <Link
            href="/app"
            className="shadow-md mt-[50px] font-medium text-sm bg-white inline-flex items-center border border-white px-4 py-2 rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
          >
            Star on Github <ChevronRight className="w-4 h-4" />
          </Link>
        </div>
        <div className="relative">
          <GithubSvg className="w-[380px] md:w-[640px] mt-24 xl:mt-0" />
          <div className="absolute w-[1000px] h-[400px] top-[400px] left-[150px]">
            <OssChip />
          </div>
        </div>
      </div>
    </div>
  );
}
