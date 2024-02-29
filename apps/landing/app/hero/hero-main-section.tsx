"use client";
import { motion } from "framer-motion";
import Link from "next/link";

import { PrimaryButton, SecondaryButton } from "@/components/button";
import { BookOpen, ChevronRight, LogIn } from "lucide-react";

export function HeroMainSection() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 1, ease: "easeOut" }}
      whileInView="visible"
      className="relative text-center xl:text-left flex flex-col items-center xl:items-start"
    >
      <div className="absolute top-[-50px] hero-hiring-gradient text-white text-sm flex space-x-2 py-1.5 px-2 items-center">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <g clip-path="url(#clip0_840_5284)">
            <path
              d="M13 0.75C13 1.89705 11.8971 3 10.75 3C11.8971 3 13 4.10295 13 5.25C13 4.10295 14.1029 3 15.25 3C14.1029 3 13 1.89705 13 0.75Z"
              stroke="white"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <path
              d="M13 10.75C13 11.8971 11.8971 13 10.75 13C11.8971 13 13 14.1029 13 15.25C13 14.1029 14.1029 13 15.25 13C14.1029 13 13 11.8971 13 10.75Z"
              stroke="white"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
            <path
              d="M5 3.75C5 5.91666 2.91666 8 0.75 8C2.91666 8 5 10.0833 5 12.25C5 10.0833 7.0833 8 9.25 8C7.0833 8 5 5.91666 5 3.75Z"
              stroke="white"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </g>
          <defs>
            <clipPath id="clip0_840_5284">
              <rect width="16" height="16" fill="white" />
            </clipPath>
          </defs>
        </svg>
        <Link href="/careers">We are hiring!</Link>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <path d="M12 8.5L8 4.5M12 8.5L8 12.5M12 8.5H4" stroke="white" />
        </svg>
      </div>

      <h1 className="bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white max-w-[546px] via-30% to-white/30 font-medium text-[32px] leading-[48px] xs:text-[48px] xs:leading-[56px] sm:text-[56px] sm:leading-[72px] md:text-[64px] md:leading-[80px] xl:text-[64px] xl:leading-[80px]  ">
        Build your API, not auth
      </h1>

      <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg text-sm xs:text-[15px] sm:text-base leading-[28px]">
        Unkey allows you to quickly add API keys, rate limiting, and usage analytics to your API, so
        that you can safely expose it to developers.
      </p>

      <div className="flex items-center gap-6 mt-16">
        <Link href="/app" className="group">
          <PrimaryButton IconLeft={LogIn} label="Get started" className="h-10" />
        </Link>

        <Link href="/docs" className="hidden sm:flex">
          <SecondaryButton IconLeft={BookOpen} label="Documentation" IconRight={ChevronRight} />
        </Link>
      </div>
    </motion.div>
  );
}
