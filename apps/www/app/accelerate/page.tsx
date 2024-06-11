import {
  BookText,
  ChevronRight,
  LockIcon,
  LockKeyhole,
  MicIcon,
  SquarePlay,
  VideoIcon,
} from "lucide-react";
import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";

import { cn } from "@/lib/utils";

import s from "./accelerate.module.css";
import { AccelerateFooterIllustration } from "./components/footer-illustration";
import { RiveAccelerate } from "./components/rive";
import SVGAccelerateMini from "./components/svg-accelerate-mini";
import { AccelerateToolboxIcon, AccelerateToolboxIllustration } from "./components/toolbox";

const pageConfig = {
  name: "Unkey Accelerate | 24-29 June 2024",
  description:
    "A week of new features that redefines API Management, allowing you to create performant and scalable APIs with ease.",
  ogImage: "https://unkey.dev/assets/accelerate/og.png",
};

export const metadata: Metadata = {
  title: {
    default: pageConfig.name,
    template: pageConfig.name,
  },
  metadataBase: new URL("https://unkey.dev"),
  description: pageConfig.description,
  keywords: ["unkey", "api", "service", "accelerate"],
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://unkey.dev/accelerate",
    title: pageConfig.name,
    description: pageConfig.description,
    siteName: pageConfig.name,
    images: [
      {
        url: pageConfig.ogImage,
        width: 1200,
        height: 630,
        alt: pageConfig.name,
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: pageConfig.name,
    description: pageConfig.description,
    images: [pageConfig.ogImage],
    creator: "@unkeydev",
  },
};

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
const WEEKDAYS_LABELS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const MAX_DAYS = 6;

export default function AcceleratePage() {
  const startDate = new Date("2024-06-24T08:00:00-07:00");
  const msSinceStart = new Date().getTime() - startDate.getTime();
  const daysSinceStart = msSinceStart / (1000 * 60 * 60 * 24);

  // IMPORTANT: needs to be capped between 0 and MAX
  const dayNumber = Math.min(MAX_DAYS, Math.max(Math.ceil(daysSinceStart), 0));

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
          {WEEKDAYS_LABELS.slice(0, MAX_DAYS).map((label, idx) => (
            <Link
              href="/accelerate#day_1"
              key={label}
              className={cn(dayNumber < idx + 1 && "opacity-20", "")}
            >
              {label}
            </Link>
          ))}
        </div>

        {/* Heading */}
        <div className="relative z-10 w-full max-w-[774px] flex flex-col items-center gap-4 lg:gap-6 pt-10 lg:pt-20 pb-8 lg:pb-16 opacity-0 animate-fade-in-right [animation-delay:1.5s]">
          <h1 className="text-white text-4xl lg:text-[64px] lg:leading-[72px] text-center lg:text-left">
            Unkey{" "}
            <span className="relative italic text-white">
              <div
                aria-hidden
                className="hidden lg:[display:unset] pointer-events-none absolute inset-0 bg-clip-text [-webkit-text-fill-color:transparent] blur-[18px] bg-[linear-gradient(90deg,#20C5F3_0%,#7002FC_22.5%,#FF4200_100%)]"
              >
                Accelerate
              </div>
              <span className="relative">
                Accelerate{" "}
                <div
                  aria-hidden
                  className="lg:hidden pointer-events-none absolute inset-0 bg-clip-text [-webkit-text-fill-color:transparent] blur-[18px] bg-[linear-gradient(90deg,#20C5F3_0%,#7002FC_22.5%,#FF4200_100%)]"
                >
                  Accelerate
                </div>
              </span>
            </span>
          </h1>
          <h2 className="text-white/30 text-base flex flex-col items-center text-center lg:flex-row">
            <span>Launch Week</span>
            <span className="hidden lg:[display:unset]">&nbsp;|&nbsp;</span>
            <span>Jun 24-29 | 8am PT</span>
          </h2>
        </div>

        {/* Hero Art */}
        <div className="relative z-0 w-full flex items-center justify-center my-[3vh] mb-[0vh] md:mt-[20vh] md:mb-[10vh] lg:my-0">
          <div className="-ml-[11.531%] relative w-full aspect-[1252/2000] lg:aspect-[1252/874] pointer-events-none lg:pointer-events-auto">
            {/* Cropper */}
            <div className="absolute inset-[-50%] lg:inset-0 scale-[1.3] lg:scale-100 -translate-x-[4.4%] lg:translate-x-0 [mask-image:linear-gradient(to_bottom,black_50%,transparent_80%)] lg:[mask-image:none]">
              <RiveAccelerate day={dayNumber} />
            </div>
          </div>
        </div>

        <div className="relative z-10 mt-10 lg:mt-20 text-white text-sm lg:text-[1.18rem] lg:leading-8 w-full">
          <p className="max-w-[440px] text-balance text-white/20">
            <span className="text-white opacity-0 animate-fade-in [animation-delay:3s]">
              Welcome to Unkey Accelerate.
            </span>{" "}
            <span className="opacity-0 animate-fade-in [animation-delay:3.4s] text-white/30">
              A week of new features that redefines API Management, allowing you to create
              performant and scalable APIs with ease.
            </span>
          </p>
        </div>

        <div className="mt-24 lg:mt-[10.5rem] py-4 lg:py-6 text-white w-full flex justify-between items-center border-b-[1px] border-white/10 opacity-0 animate-fade-in [animation-delay:4s]">
          <h3 className="max-w-[30%]">Unkey Accelerate 2024</h3>
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
            // TODO: Componentize
            <div
              key={day.weekday}
              className="relative flex flex-col lg:flex-row gap-10 lg:gap-24 items-center pb-10 lg:pb-20 border-b-[1px] border-white/10 [&:not(:first-child)]:mt-20 opacity-0 animate-fade-in [animation-delay:4s]"
            >
              <span className="absolute w-px h-px left-0 -top-[100px]" id={`day_${idx + 1}`} />

              <div className="relative flex flex-col w-full lg:max-w-[335px] gap-10">
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

                  <div className="flex flex-col gap-6 text-white [stroke-width:1px] lg:text-nowrap leading-tight">
                    <day.IconComponent />
                    {dayNumber >= idx + 1 ? (
                      <div className="text-[2rem]">{day.title}</div>
                    ) : (
                      <div className="text-[2rem]">Available on {day.dayAndMonth}</div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Pre-footer */}
        <div className="mt-12 lg:mt-24 w-full flex flex-col lg:flex-row gap-10 lg:gap-0 items-center lg:justify-between text-sm leading-6 text-center opacity-0 animate-fade-in [animation-delay:5s]">
          <div className="flex flex-col lg:text-left">
            <span className="text-white">FAST TRACK ALERT: Speed Ahead with Caution.</span>
            <span>Launches May Include Bursts of Productivity.</span>
          </div>
          <div className="flex flex-col lg:text-right">
            <span className="text-white">UNKEY LABs</span>
            <span>HIGH-VELOCITY INNOVATIONS DEPT</span>
          </div>
        </div>

        {/* Footer Illustration */}
        <div className="relative mt-24 lg:mt-44 max-w-[800px] w-full aspect-[2/1] overflow-hidden opacity-0 animate-fade-in [animation-delay:6s]">
          <div className={cn("w-full aspect-square", s.footer_illustration)}>
            <AccelerateFooterIllustration />
          </div>
        </div>
      </div>
    </div>
  );
}
