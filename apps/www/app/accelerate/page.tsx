import { cn } from "@/lib/utils";
import { BookText, ChevronRight, LockKeyhole, MicIcon, SquarePlay, VideoIcon } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import s from "./accelerate.module.css";
import { AccelerateFooterIllustration } from "./components/footer-illustration";
import { RiveAccelerate } from "./components/rive";
import SVGAccelerateMini from "./components/svg-accelerate-mini";
import { AccelerateToolboxIcon, AccelerateToolboxIllustration } from "./components/toolbox";

type AccelerateLaunchDay = {
  dateTime: string;
  dayAndMonth: string;
  weekday: string;
  blog?: string;
  video?: string;
  audioLive?: string;
  documentation?: string;
  title: string;
  IconComponent: React.ComponentType<any>;
  IllustrationComponent: React.ComponentType<any>;
};
const DAYS: AccelerateLaunchDay[] = [
  {
    dateTime: "2024-06-24",
    dayAndMonth: "24 Jun",
    weekday: "Mon",
    title: "Toolbox",
    blog: "https://unkey.dev/blog/toolbox",
    IconComponent: AccelerateToolboxIcon,
    IllustrationComponent: AccelerateToolboxIllustration,
  },
];

export default function AcceleratePage() {
  const startDate = new Date("2024-06-24T08:00:00-07:00");
  const msSinceStart = new Date().getTime() - startDate.getTime();
  const daysSinceStart = msSinceStart / (1000 * 60 * 60 * 24);
  const dayNumber = Math.min(7, Math.max(Math.ceil(daysSinceStart), 0));

  return (
    <div className="flex flex-col">
      <div className="isolate container flex flex-col items-center mt-[64.5px] font-mono uppercase text-white/30 text-base">
        {/* Day Schedule Representation */}
        <div
          className={cn(
            "relative flex w-full justify-center items-center p-6 text-sm leading-6 text-white gap-6 border-y-[1px] border-white/10",
            "opacity-0 animate-fade-in [animation-delay:0s]",
          )}
        >
          {["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map((label, idx) => (
            <span key={label} className={cn(dayNumber < idx + 1 && "opacity-20", "")}>
              {label}
            </span>
          ))}
        </div>

        {/* Heading */}
        <div className="relative z-10 w-full max-w-[774px] flex flex-col items-center gap-4 lg:gap-6 pt-10 lg:pt-20 pb-8 lg:pb-16 opacity-0 animate-fade-in-right [animation-delay:1.5s]">
          <h1 className="text-white text-4xl lg:text-[64px] lg:leading-[72px]">
            Unkey{" "}
            <span className="relative italic text-white">
              <div
                aria-hidden
                className="pointer-events-none absolute inset-0 bg-clip-text [-webkit-text-fill-color:transparent] blur-[18px] bg-[linear-gradient(90deg,#20C5F3_0%,#7002FC_22.5%,#FF4200_100%)]"
              >
                Accelerate
              </div>
              <span className="relative">Accelerate</span>
            </span>
          </h1>
          <h2 className="text-white/30 text-base">Launch Week | Jun 24-29 | 8am PT</h2>
        </div>

        {/* Hero Art */}
        <div className="z-0 w-full flex items-center justify-center relative my-12 lg:my-0">
          <div className="-ml-[11.531%] relative w-full aspect-[1252/1137] lg:aspect-[1252/874]">
            {/* Cropper */}
            <div className="absolute inset-[-30%] lg:inset-0 -translate-x-[12.5%] lg:translate-x-0">
              {/* Rotator */}
              {/* <div id="accelerate_rotator" className={cn(s.accelerate_rotator,"absolute inset-0")}> */}
              <RiveAccelerate day={dayNumber} />
              {/* </div> */}
            </div>
          </div>
        </div>

        <div className="relative z-10 mt-10 lg:mt-20 text-white text-sm lg:text-[1.18rem] lg:leading-8 w-full">
          <p className="max-w-[440px] text-balance text-white/20">
            <span className="text-white opacity-0 animate-fade-in [animation-delay:3s]">
              Welcome to Unkey Accelerate.
            </span>{" "}
            <span className="opacity-0 animate-fade-in [animation-delay:3.4s]">
              A week of new features that redefines API Management, allowing you to create
              performant and scalable APIs with ease.
            </span>
          </p>
        </div>

        <div className="mt-24 lg:mt-[10.5rem] py-4 lg:py-6 text-white w-full flex justify-between items-center border-b-[1px] border-white/10">
          <h3>Unkey Accelerate 2024</h3>
          <Link
            className="flex items-center gap-3 hover:opacity-50 transition"
            href="https://x.com/unkeydev"
            target="_blank"
          >
            Learn more <ChevronRight size={16} />
          </Link>
        </div>

        <div className="w-full flex flex-col mt-14 lg:mt-[7.5rem]">
          {DAYS.map((day, idx) => (
            <div
              key={day.weekday}
              className="relative flex flex-col lg:flex-row gap-10 lg:gap-24 items-center pb-10 lg:pb-20 border-b-[1px] border-white/10 [&:not(:first-child)]:mt-20"
            >
              <span className="absolute w-px h-px left-0 -top-[100px]" id={`anchor_d${idx + 1}`} />

              <div className="relative flex flex-col w-full max-w-[335px] gap-10">
                <div className="flex justify-between lg:justify-start lg:flex-col lg:gap-10 max-h-6 lg:max-h-[unset]">
                  <time dateTime={day.dateTime}>
                    Day {String(idx + 1).padStart(2, "0")} | {day.weekday}, {day.dayAndMonth}
                  </time>

                  <div className="w-16 h-16">
                    <SVGAccelerateMini className="-ml-6 -mt-4" />
                  </div>
                </div>

                <div className="flex w-full gap-6 whitespace-nowrap">
                  {dayNumber >= idx + 1 ? (
                    <>
                      {day.blog && (
                        <Link href={day.blog} className="flex items-center gap-2 text-nowrap">
                          <BookText size={16} className="text-white" />
                          <span>Blog Post</span>
                        </Link>
                      )}
                      {day.video && (
                        <Link href={day.video} className="flex items-center gap-2 text-nowrap">
                          <SquarePlay size={16} className="text-white" />
                          <span>Video</span>
                        </Link>
                      )}
                      {day.audioLive && (
                        <Link href={day.audioLive} className="flex items-center gap-2 text-nowrap">
                          <MicIcon size={16} className="text-white" />
                          <span>Audio Live</span>
                        </Link>
                      )}
                    </>
                  ) : (
                    <div className="flex items-center gap-2 text-nowrap">
                      <LockKeyhole size={16} />
                      <span>Unlocks at {day.dayAndMonth}</span>
                    </div>
                  )}
                </div>
              </div>

              <div className="relative max-w-[760px] flex w-full lg:flex-1 h-[20rem] border-[1px] border-white/10 rounded-3xl p-6 flex-col overflow-hidden">
                <div className="pointer-events-none absolute right-0 aspect-square w-full max-w-[500px] top-1/2 -translate-y-[85%] lg:-translate-y-1/2 translate-x-[35%] lg:translate-x-[15%] scale-[1.8] lg:scale-100 [mask-image:radial-gradient(black,rgba(0,0,0,0.2))]">
                  <day.IllustrationComponent />
                </div>

                <div className="relative flex flex-col justify-between w-full h-full max-w-[300px]">
                  <div className="flex items-center gap-2 h-6">
                    {day.documentation && (
                      <>
                        <BookText size={16} className="text-white [stroke-width:1px]" />
                        <span>Documentation</span>
                      </>
                    )}
                  </div>

                  <div className="flex flex-col gap-6 text-white [stroke-width:1px]">
                    <day.IconComponent />
                    {dayNumber >= idx + 1 ? (
                      <div className="text-[2rem] leading-6">{day.title}</div>
                    ) : (
                      <div className="text-[2rem] leading-6">Coming {day.dayAndMonth}</div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Pre-footer */}
        <div className="mt-12 lg:mt-24 w-full flex flex-col lg:flex-row gap-10 lg:gap-0 items-center lg:justify-between text-xs leading-5 text-center">
          <div className="flex flex-col lg:text-left">
            <span className="text-white">FAST TRACK ALERT: Speed Ahead with Caution.</span>
            <span>LAUNCHEs May Include Bursts of Performance and Minor Bugs.</span>
          </div>
          <div className="flex flex-col lg:text-right">
            <span className="text-white">UNKEY LABs</span>
            <span>HIGH-VELOCITY INNOVATIONS DEPT</span>
          </div>
        </div>

        {/* Footer Illustration */}
        <div className="relative mt-24 lg:mt-44 max-w-[800px] w-full aspect-[2/1] overflow-hidden">
          <div className="w-full aspect-square">
            <AccelerateFooterIllustration />
          </div>
        </div>
      </div>
    </div>
  );
}
