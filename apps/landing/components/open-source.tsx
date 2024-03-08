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
        <OssLight />
      </div>
      <div className="flex flex-col items-center xl:flex-row w-full justify-center xl:justify-between">
        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          transition={{ duration: 1, ease: "easeOut" }}
          viewport={{ once: true, amount: 0.5 }}
        >
          <SectionTitle
            align="left"
            title="Developer-first"
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
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true, amount: 0.5 }}
            transition={{ duration: 1, ease: "easeOut" }}
          >
            <Github className="w-[380px] md:w-[640px] mt-24 xl:mt-0" />
            <Image alt="Github logo" src={GithubSvg} className="mt-24" />
            <div className="absolute w-[1000px] h-[400px] top-[400px] left-[150px]">
              <OssChip />
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
};
