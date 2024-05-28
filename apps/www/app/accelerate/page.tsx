import { cn } from "@/lib/utils";
import Image from "next/image";
import s from "./accelerate.module.css";
import { SVGHeading } from "./components/heading";
import { RiveAccelerate } from "./components/rive";

export default function AcceleratePage() {
  return (
    <div className="flex flex-col">
      <div className="container min-h-[100dvh] flex flex-col justify-between">
        <header className="relative flex w-full justify-between items-center h-[64.5px]">
          {/* <div>Left</div> */}
          {/* <div>Right</div> */}
        </header>

        <div className="w-full flex items-center justify-center">
          <div className="relative w-full aspect-[1252/842]">
            {/* <div className="absolute inset-[-400%]"> */}
            <RiveAccelerate />

            <div
              className={cn(
                s.heading,
                "absolute pointer-events-none aspect-[385/216] w-[30.67092652%] left-0 top-[25.58685446%]",
              )}
            >
              <SVGHeading />
            </div>
            {/* </div> */}
          </div>
          {/* <div className="relative w-full aspect-[1440/960] scale-150">
            <AnimatedSpeedometer />
          </div> */}
        </div>

        <footer className="relative flex w-full justify-between items-center h-[64.5px]">
          {/* <div>Left</div> */}
          {/* <div>Right</div> */}
        </footer>

        <div aria-hidden className="absolute inset-0 pointer-events-none overflow-hidden">
          <div
            className={cn(
              s.toplight,
              "absolute aspect-[936/908] max-w-[936px] w-full top-0 left-1/2 -translate-x-1/2 -translate-y-[40%]",
            )}
          >
            <div className={cn(s.toplight_inner, "absolute inset-0")}>
              {/* TODO: use SVG */}
              {/* <SVGLightFromAbove /> */}
              <Image src="/images/accelerate/toplight.png" alt="Toplight" fill />
            </div>
          </div>
          <div
            className={cn(
              s.bottomlight,
              "absolute top-full left-1/2 -translate-x-1/2 -translate-y-1/3 max-w-[744px] w-full aspect-[744/430]",
            )}
          >
            <SVGBottomLight />
          </div>
        </div>
      </div>
    </div>
  );
}

function SVGBottomLight() {
  return (
    <svg
      width="100%"
      height="100%  "
      viewBox="0 0 744 430"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g filter="url(#filter0_f_4722_56)">
        <rect x="152" y="165" width="440" height="100" rx="50" fill="url(#paint0_linear_4722_56)" />
      </g>
      <g filter="url(#filter1_f_4722_56)">
        <rect x="152" y="165" width="440" height="100" rx="50" fill="url(#paint1_linear_4722_56)" />
      </g>
      <defs>
        <filter
          id="filter0_f_4722_56"
          x="8"
          y="21"
          width="728"
          height="388"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="72" result="effect1_foregroundBlur_4722_56" />
        </filter>
        <filter
          id="filter1_f_4722_56"
          x="8"
          y="21"
          width="728"
          height="388"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="72" result="effect1_foregroundBlur_4722_56" />
        </filter>
        <linearGradient
          id="paint0_linear_4722_56"
          x1="152"
          y1="215"
          x2="592"
          y2="215"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#20C5F3" />
          <stop offset="0.225" stop-color="#7002FC" />
          <stop offset="1" stop-color="#FF4200" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_4722_56"
          x1="152"
          y1="215"
          x2="592"
          y2="215"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#20C5F3" />
          <stop offset="0.225" stop-color="#7002FC" />
          <stop offset="1" stop-color="#FF4200" />
        </linearGradient>
      </defs>
    </svg>
  );
}
