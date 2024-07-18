"use client";
import { SectionTitle } from "@/components/section";
import { motion } from "framer-motion";
import { Star } from "lucide-react";
import Link from "next/link";
import GithubSvg from "../images/unkey-github.svg";
import { PrimaryButton } from "./button";
import { ImageWithBlur } from "./image-with-blur";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";
import { MeteorLines } from "./ui/meteorLines";

export const OpenSource: React.FC = () => {
  return (
    <div className="pt-[00px] flex items-center flex-col md:flex-row relative">
      <div className="absolute top-[-320px] md:top-[-480px] xl:right-[120px] -z-[10]">
        {/* TODO: horizontal scroll */}
        <OssLight className="scale-[2]" />
        <div className="absolute right-[270px] top-[250px] -z-50">
          <MeteorLines className="ml-2 fade-in-0" delay={2} number={1} />
          <MeteorLines className="ml-10 fade-in-40" number={1} delay={0} />
          <MeteorLines className="ml-16 fade-in-100" delay={4} number={1} />
        </div>
      </div>
      <div className="flex flex-col items-center justify-center w-full xl:flex-row xl:justify-between">
        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          transition={{ duration: 1, ease: "easeInOut" }}
          viewport={{ once: true, amount: 0.5 }}
        >
          <SectionTitle
            align="left"
            title="Open-source"
            text="We believe strongly in the value of open source: our codebase and development process is available to learn from and contribute to."
            label="oss/acc"
          >
            <div className="flex mt-10 space-x-6">
              <Link href="https://github.com/unkeyed/unkey" className="group">
                <PrimaryButton IconLeft={Star} label="Star us on GitHub" shiny />
              </Link>
            </div>
          </SectionTitle>
        </motion.div>
        <div className="relative">
          <motion.div
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true, amount: 0.5 }}
            transition={{ duration: 1, ease: "easeInOut" }}
          >
            <ImageWithBlur alt="Github logo" src={GithubSvg} className="mt-24" />
            <div className="absolute -z-50 top-[150px] left-[-50px] lg:w-[1000px] lg:h-[400px] lg:top-[400px] lg:left-[150px]">
              <OssChip className="flex" />
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
};
