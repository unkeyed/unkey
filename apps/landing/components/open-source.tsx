import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { GithubSvg } from "./svg/github";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export function OpenSource() {
  return (
    <div className="pt-[150px] flex items-center flex-col md:flex-row justify-center relative">
      {/* TODO: add additional line SVGs from Figma – current export is broken */}
      {/* <div className="absolute top-[-260px] right-[240px]">
        <OssLight />
      </div> */}
      <div className="md:pr-24 flex flex-col items-center md:items-start">
        <p className="font-mono text-white/50 text-center md:text-left">Open-source</p>
        <h1 className="text-[28px] leading-9 leading- md:text-[52px] text-white md:max-w-[463px] pt-4 open-source-heading-gradient text-center md:text-left">
          Empowering the community
        </h1>
        <p className="text-white leading-7 max-w-[461px] pt-[26px] text-center md:text-left">
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
      <div className="relative mt-20">
        <GithubSvg className="w-[380px] md:w-[400px]" />
        <div className="fixed w-[2000px] h-[400px]">
          <div className="absolute top-[-200px] left-[100px]">
            <OssChip />
          </div>
        </div>
      </div>
    </div>
  );
}
