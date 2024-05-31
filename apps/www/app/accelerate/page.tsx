import { cn } from "@/lib/utils";
import { BookText, ChevronRight, MicIcon, SquarePlay, VideoIcon } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import s from "./accelerate.module.css";
import { RiveAccelerate } from "./components/rive";
import SVGAccelerateMini from "./components/svg-accelerate-mini";

const CURRENT_DAY = -1;

export default function AcceleratePage() {
  return (
    <div className="flex flex-col">
      <div className="container flex flex-col items-center mt-[64.5px] font-mono uppercase text-white/30 text-base">
        {/* Day Schedule Representation */}
        <div className="relative flex w-full justify-center items-center p-6 text-sm leading-6 text-white gap-6 border-y-[1px] border-white/[.08]">
          {["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map((label, idx) => (
            <span key={label} className={cn(CURRENT_DAY !== idx && "opacity-20", "")}>
              {label}
            </span>
          ))}
        </div>

        {/* Heading */}
        <div className="w-full max-w-[774px] flex flex-col items-center gap-4 md:gap-6 pt-10 md:pt-20 pb-8 md:pb-16">
          <h1 className="text-white text-4xl md:text-[64px] md:leading-[72px]">
            Unkey <span className="italic">Accelerate</span>
          </h1>
          <h2 className="text-white/30 text-base">Launch Week | Jun 17-23 | 8am PT</h2>
        </div>

        {/* Hero Art */}
        <div className="w-full flex items-center justify-center">
          <div className="-ml-[11.531%] relative w-full aspect-[1252/842]">
            {/* <div className="absolute inset-[-400%]"> */}
            <RiveAccelerate />

            {/* </div> */}
          </div>
        </div>

        <div className="mt-10 md:mt-20 text-white text-sm md:text-[1.18rem] md:leading-8 w-full">
          <p className="max-w-[440px] text-balance text-white/20">
            <span className="text-white">Welcome to Unkey Accelerate.</span> discover how we're
            redefining the speed and efficiency of API Management. Join us this week to see how
            every millisecond enhances your API performance.
          </p>
        </div>

        <div className="mt-24 md:mt-[10.5rem] py-4 md:py-6 text-white w-full flex justify-between items-center border-b-[1px] border-white/[.08]">
          <h3>Unkey Accelerate 2024</h3>
          <Link className="flex items-center gap-3" href="/accelerate#learn-more">
            Learn more <ChevronRight size={16} />
          </Link>
        </div>

        <div className="w-full flex flex-col mt-14 md:mt-[7.5rem]">
          {[0, 1, 2, 3, 4, 5, 6].map((day) => (
            <div
              key={day}
              className="flex gap-24 items-center h-[20rem] pb-10 md:pb-20 border-b-[1px] border-white/[.08] [&:not(:first-child)]:mt-20"
            >
              <div className="flex flex-col w-full max-w-[335px] gap-10">
                <time dateTime="2024-06-17">Day 01 | Mon, 17 June</time>
                <SVGAccelerateMini />
                <div className="flex w-full gap-6 whitespace-nowrap">
                  <div className="flex items-center gap-2 text-nowrap">
                    <BookText size={16} className="text-white" />
                    <span>Blog Post</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <SquarePlay size={16} className="text-white" />
                    <span>Video</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <MicIcon size={16} className="text-white" />
                    <span>Audio Live</span>
                  </div>
                </div>
              </div>

              <div className="max-w-[760px] flex flex-1 h-full border-[1px] border-white/[.08] rounded-3xl p-6 flex flex-col">
                <div className="flex flex-col justify-between w-full h-full max-w-[300px]">
                  <div className="flex items-center gap-2">
                    <BookText size={16} className="text-white [stroke-width:1px]" />
                    <span>Documentation</span>
                  </div>

                  <div className="flex flex-col gap-6 text-white [stroke-width:1px]">
                    <BookText size={36} />
                    <div className="text-[2rem] leading-6">Toolbox</div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
