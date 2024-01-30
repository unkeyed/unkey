import { SectionTitle } from "@/app/section-title";
import { Star } from "lucide-react";
import Link from "next/link";
import { PrimaryButton } from "./button";
import { GithubSvg } from "./svg/github";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export function OpenSource() {
  return (
    <div className="pt-[00px] flex items-center flex-col md:flex-row justify-center relative">
      {/* TODO: add additional line SVGs from Figma – current export is broken */}
      <div className="absolute top-[-460px] md:right-[240px] z-[-1]">
        <OssLight />
      </div>
      <div className="flex flex-col items-center xl:flex-row ">
        <SectionTitle
          align="left"
          title="Empowering the community"
          text="Unkey allows open-source contributions through Github, enabing collaboration and
        knowledge sharing with all the developers in the world."
          titleWidth={463}
          contentWidth={461}
          label="oss/acc"
        >
          <div className="flex mt-10 space-x-6">
            <Link href="/app" className="group">
              <PrimaryButton IconLeft={Star} label="Star us on GitHub" />
            </Link>
          </div>
        </SectionTitle>
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
