import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { GithubSvg } from "./svg/github";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export function OpenSource() {
  return (
    <div className="pt-[150px] flex items-center justify-center relative">
      {/* TODO: add additional line SVGs from Figma – current export is broken */}
      <div className="absolute top-[-260px] right-[240px]">
        <OssLight />
      </div>
      <div className="pr-24">
        <p className="font-mono text-white/50 ">Open-source</p>
        <h1 className="text-[52px] leading-[64px] text-white max-w-[463px] pt-4 open-source-heading-gradient">
          Empowering the community
        </h1>
        <p className="text-white leading-7 max-w-[461px] pt-[26px]">
          Unkey allows open-source contributions through Github, enabing collaboration and knowledge
          sharing with all the developers in the world.
        </p>
        <Link
          href="/app"
          className="shadow-md mt-[50px] font-medium text-sm bg-white inline-flex items-center border border-white px-4 py-2 rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
        >
          Star on Github <ChevronRight className="w-4 h-4" />
        </Link>
      </div>
      <div className="relative  overflow-hidden">
        <GithubSvg />
        <div className="absolute">
          <OssChip />
        </div>
      </div>
    </div>
  );
}
