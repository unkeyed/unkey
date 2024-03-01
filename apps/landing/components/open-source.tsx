"use client";
import { SectionTitle } from "@/app/section-title";
import { motion } from "framer-motion";
import { Star } from "lucide-react";
import Link from "next/link";
import { PrimaryButton } from "./button";
import { Github, GithubMobile } from "./svg/github";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export const OpenSource: React.FC = () => {
  return (
    <div className="pt-[00px] flex items-center flex-col md:flex-row relative">
      {/* TODO: add additional line SVGs from Figma – current export is broken */}
      <div className="absolute top-[-460px] md:right-[120px] z-[-1]">
        <OssLight />
      </div>
      <div className="container flex flex-col items-center xl:flex-row xl:w-full xl:justify-between">
        <motion.div
          initial={{ opacity: 0 }} // Start with the component invisible
          whileInView={{ opacity: 1 }} // Animate to fully visible when in view
          transition={{ duration: 1, ease: "easeOut" }} // Define the transition
          viewport={{ once: true, amount: 0.5 }}
        >
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
        </motion.div>
        <div className="relative">
          <motion.div
            initial={{ opacity: 0 }} // Start with the component invisible
            whileInView={{ opacity: 1 }} // Animate to fully visible when in view
            viewport={{ once: true, amount: 0.5 }}
            transition={{ duration: 1, ease: "easeOut" }} // Define the transition
          >
            <GithubMobile className="flex mt-24 sm:hidden" />
            <Github className="hidden sm:flex w-[380px] md:w-[640px] mt-24 xl:mt-0" />
            <div className="absolute w-[1000px] h-[400px] top-[400px] left-[150px]">
              <OssChip />
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
};
