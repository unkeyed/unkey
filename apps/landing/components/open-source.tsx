"use client";
import { SectionTitle } from "@/app/section-title";
import { motion } from "framer-motion";
import { Star } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import GithubSvg from "../images/unkey-github.svg";
import { PrimaryButton } from "./button";
import { OssChip } from "./svg/oss-chip";
import { OssLight } from "./svg/oss-light";

export const OpenSource: React.FC = () => {
  return (
    <div className="pt-[00px] flex items-center flex-col md:flex-row relative">
      <div className="absolute top-[-460px] md:right-[120px] -z-[10]">
        {/* TODO: horizontal scroll */}
        <OssLight />
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
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true, amount: 0.5 }}
            transition={{ duration: 1, ease: "easeInOut" }}
          >
            <Image alt="Github logo" src={GithubSvg} className="mt-24" />
            <div className="lg:absolute lg:w-[1000px] lg:h-[400px] lg:top-[400px] lg:left-[150px]">
              <OssChip className="hidden md:flex" />
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
};
