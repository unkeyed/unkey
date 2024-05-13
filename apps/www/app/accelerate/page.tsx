import { cn } from "@/lib/utils";
import Image from "next/image";
import s from "./accelerate.module.css";

export default function AcceleratePage() {
  return (
    <div className="flex flex-col">
      <div className="container min-h-[100dvh] flex flex-col justify-between">
        <div aria-hidden className="absolute inset-0 pointer-events-none overflow-hidden">
          <div
            className={cn(
              s.toplight,
              "absolute aspect-[936/908] max-w-[936px] w-full top-0 left-1/2 -translate-x-1/2 -translate-y-[40%]",
            )}
          >
            {/* TODO: use SVG */}
            {/* <SVGLightFromAbove /> */}
            <Image src="/images/accelerate/toplight.png" alt="Toplight" fill />
          </div>
          <div className="absolute top-full left-1/2 -translate-x-1/2 -translate-y-1/2 max-w-[744px] w-full aspect-[744/430]">
            <SVGBottomLight />
          </div>
        </div>

        <header className="relative flex w-full justify-between items-center h-[64.5px]">
          {/* <div>Left</div> */}
          {/* <div>Right</div> */}
        </header>

        <div className="w-full flex items-center justify-center">
          <div className="relative w-full aspect-[1440/960] scale-150">
            <AnimatedSpeedometer />
          </div>
        </div>

        <footer className="relative flex w-full justify-between items-center h-[64.5px]">
          {/* <div>Left</div> */}
          {/* <div>Right</div> */}
        </footer>
      </div>
    </div>
  );
}

const ANIMATED_SPEEDOMETER_STYLE = `
@keyframes a0_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a1_t { 0% { transform: translate(-200px,-40px); animation-timing-function: cubic-bezier(.6,0,.4,1); } 75% { transform: translate(1320px,-40px); } 100% { transform: translate(1320px,-40px); } }
@keyframes a2_t { 0% { transform: translate(-200px,-40px); animation-timing-function: cubic-bezier(.6,0,.4,1); } 75% { transform: translate(1320px,-40px); } 100% { transform: translate(1320px,-40px); } }
@keyframes a3_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a4_f { 0% { fill: url(#Gradient-3); animation-timing-function: steps(1); } 2.837% { fill: rgba(0,0,0,0.60); animation-timing-function: steps(1); } 84.161% { fill: rgba(0,0,0,0.64); } 93.617% { fill: rgba(0,0,0,0.30); } 100% { fill: rgba(0,0,0,0.50); } }
@keyframes a5_f { 0% { fill: url(#Gradient-3); animation-timing-function: steps(1); } 84.161% { fill: rgba(0,0,0,0.64); } 93.617% { fill: rgba(0,0,0,0.30); } 100% { fill: rgba(0,0,0,0.50); } }
@keyframes a6_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a7_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a8_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-195px,-154.5px); } 8.333% { transform: translate(210px,210px) rotate(10.4deg) translate(-195px,-154.5px); } 16.667% { transform: translate(210px,210px) rotate(19.6deg) translate(-195px,-154.5px); } 25% { transform: translate(210px,210px) rotate(26.9deg) translate(-195px,-154.5px); } 33.333% { transform: translate(210px,210px) rotate(32.1deg) translate(-195px,-154.5px); } 41.667% { transform: translate(210px,210px) rotate(36.2deg) translate(-195px,-154.5px); } 50% { transform: translate(210px,210px) rotate(39.1deg) translate(-195px,-154.5px); } 58.333% { transform: translate(210px,210px) rotate(41deg) translate(-195px,-154.5px); } 66.667% { transform: translate(210px,210px) rotate(42.9deg) translate(-195px,-154.5px); } 75% { transform: translate(210px,210px) rotate(44deg) translate(-195px,-154.5px); } 83.333% { transform: translate(210px,210px) rotate(44.6deg) translate(-195px,-154.5px); } 100% { transform: translate(210px,210px) rotate(45deg) translate(-195px,-154.5px); } }
@keyframes a8_o { 0% { opacity: 1; } 100% { opacity: 0; } }
@keyframes a9_t { 0% { transform: translate(210px,210px) rotate(45deg) translate(-195px,-154.5px); } 33.333% { transform: translate(210px,210px) rotate(42.3deg) translate(-195px,-154.5px); } 66.667% { transform: translate(210px,210px) rotate(45.1deg) translate(-195px,-154.5px); } 100% { transform: translate(210px,210px) rotate(42deg) translate(-195px,-154.5px); } }
@keyframes a9_o { 0% { opacity: 0; } 100% { opacity: 1; } }
@keyframes a10_o { 0% { opacity: .2; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a11_t { 0% { transform: translate(210.5px,209.5px) rotate(0deg) translate(-155.5px,-154.5px); } 100% { transform: translate(210.5px,209.5px) rotate(51.8deg) translate(-155.5px,-154.5px); } }
@keyframes a12_t { 0% { transform: translate(210.5px,209.5px) rotate(0deg) translate(-155.5px,-154.5px); } 50% { transform: translate(210.5px,209.5px) rotate(51.8deg) translate(-155.5px,-154.5px); } 100% { transform: translate(210.5px,209.5px) rotate(98.2deg) translate(-155.5px,-154.5px); } }
@keyframes a13_t { 0% { transform: translate(210.5px,209.5px) rotate(0deg) translate(-155.5px,-154.5px); } 20% { transform: translate(210.5px,209.5px) rotate(51.8deg) translate(-155.5px,-154.5px); } 40% { transform: translate(210.5px,209.5px) rotate(98.2deg) translate(-155.5px,-154.5px); } 100% { transform: translate(210.5px,209.5px) rotate(182.7deg) translate(-155.5px,-154.5px); } }
@keyframes a14_t { 0% { transform: translate(210.5px,209.5px) rotate(0deg) translate(-155.5px,-154.5px); } 10% { transform: translate(210.5px,209.5px) rotate(51.8deg) translate(-155.5px,-154.5px); } 20% { transform: translate(210.5px,209.5px) rotate(98.2deg) translate(-155.5px,-154.5px); } 50% { transform: translate(210.5px,209.5px) rotate(182.7deg) translate(-155.5px,-154.5px); } 100% { transform: translate(210.5px,209.5px) rotate(223deg) translate(-155.5px,-154.5px); } }
@keyframes a15_o { 0% { opacity: .2; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: .8; } }
@keyframes a16_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 72.222% { opacity: 1; } 100% { opacity: 0; } }
@keyframes a17_o { 0% { opacity: 0; } 100% { opacity: 1; } }
@keyframes a18_o { 0% { opacity: 1; animation-timing-function: cubic-bezier(.6,.1,.3,1); } 13.131% { opacity: .3; animation-timing-function: cubic-bezier(.6,0,.4,1); } 32.657% { opacity: 1; animation-timing-function: cubic-bezier(.6,0,.4,1); } 71.131% { opacity: .7; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a19_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-136px,-154.5px); } 100% { transform: translate(210px,210px) rotate(52deg) translate(-136px,-154.5px); } }
@keyframes a20_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-136px,-154.5px); } 50% { transform: translate(210px,210px) rotate(52deg) translate(-136px,-154.5px); } 100% { transform: translate(210px,210px) rotate(98deg) translate(-136px,-154.5px); } }
@keyframes a21_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-136px,-154.5px); } 33.333% { transform: translate(210px,210px) rotate(52deg) translate(-136px,-154.5px); } 66.667% { transform: translate(210px,210px) rotate(98deg) translate(-136px,-154.5px); } 100% { transform: translate(210px,210px) rotate(134.7deg) translate(-136px,-154.5px); } }
@keyframes a22_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-136px,-154.5px); } 25% { transform: translate(210px,210px) rotate(52deg) translate(-136px,-154.5px); } 50% { transform: translate(210px,210px) rotate(98deg) translate(-136px,-154.5px); } 75% { transform: translate(210px,210px) rotate(134.7deg) translate(-136px,-154.5px); } 100% { transform: translate(210px,210px) rotate(162.3deg) translate(-136px,-154.5px); } }
@keyframes a23_t { 0% { transform: translate(210px,210px) rotate(-0.3deg) translate(-136px,-154.5px); } 10% { transform: translate(210px,210px) rotate(52deg) translate(-136px,-154.5px); } 20% { transform: translate(210px,210px) rotate(98deg) translate(-136px,-154.5px); } 30% { transform: translate(210px,210px) rotate(134.7deg) translate(-136px,-154.5px); } 40% { transform: translate(210px,210px) rotate(162.3deg) translate(-136px,-154.5px); } 50% { transform: translate(210px,210px) rotate(182.4deg) translate(-136px,-154.5px); } 60% { transform: translate(210px,210px) rotate(197deg) translate(-136px,-154.5px); } 70% { transform: translate(210px,210px) rotate(207deg) translate(-136px,-154.5px); } 80% { transform: translate(210px,210px) rotate(214.5deg) translate(-136px,-154.5px); } 90% { transform: translate(210px,210px) rotate(219.6deg) translate(-136px,-154.5px); } 100% { transform: translate(210px,210px) rotate(222.9deg) translate(-136px,-154.5px); } }
@keyframes a24_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a25_o { 0% { opacity: 0; } 50% { opacity: 0; animation-timing-function: cubic-bezier(.6,.1,.7,.2); } 100% { opacity: 1; } }
@keyframes a26_t { 0% { transform: translate(271.2px,271.2px); animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { transform: translate(211.1px,211.1px); } }
@keyframes a26_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.7,0,.4,1); } 60.526% { opacity: 1; } 63.158% { opacity: 1; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 0; } }
@keyframes a27_t { 0% { transform: translate(211.1px,211.1px) scale(1,1); animation-timing-function: cubic-bezier(.6,0,.4,1); } 18.518% { transform: translate(211.1px,211.1px) scale(.9,.9); animation-timing-function: cubic-bezier(.6,0,.4,1); } 46.296% { transform: translate(211.1px,211.1px) scale(1,1); animation-timing-function: cubic-bezier(.6,0,.4,1); } 61.111% { transform: translate(211.1px,211.1px) scale(1,1); animation-timing-function: cubic-bezier(.6,0,.4,1); } 77.778% { transform: translate(211.1px,211.1px) scale(.9,.9); animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { transform: translate(211.1px,211.1px) scale(1,1); } }
@keyframes a27_o { 0% { opacity: 0; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a28_o { 0% { opacity: .2; animation-timing-function: cubic-bezier(.6,0,.4,1); } 100% { opacity: 1; } }
@keyframes a29_t { 0% { transform: translate(210px,210px) rotate(-225deg) translate(-210px,-210px); animation-timing-function: cubic-bezier(.2,.6,.4,1); } 100% { transform: translate(210px,210px) rotate(-180deg) translate(-210px,-210px); } }
@keyframes a29_o { 0% { opacity: .3; } 92.308% { opacity: 1; } 100% { opacity: 0; } }
@keyframes a30_t { 0% { transform: translate(210px,210px) rotate(-180deg) translate(-210px,-210px); } 33.333% { transform: translate(210px,210px) rotate(-183deg) translate(-210px,-210px); } 66.667% { transform: translate(210px,210px) rotate(-180deg) translate(-210px,-210px); } 100% { transform: translate(210px,210px) rotate(-183deg) translate(-210px,-210px); } }
@keyframes a30_o { 0% { opacity: 0; } 100% { opacity: 1; } }
@keyframes a31_t { 0% { transform: translate(210px,210px) rotate(-225deg) translate(-210px,-210px); animation-timing-function: cubic-bezier(.2,.6,.4,1); } 100% { transform: translate(210px,210px) rotate(0deg) translate(-210px,-210px); } }
@keyframes a31_o { 0% { opacity: .3; } 92.308% { opacity: 1; } 100% { opacity: 0; } }
@keyframes a32_t { 0% { transform: translate(210px,210px) rotate(0deg) translate(-210px,-210px); } 33.333% { transform: translate(210px,210px) rotate(-3deg) translate(-210px,-210px); } 66.667% { transform: translate(210px,210px) rotate(0deg) translate(-210px,-210px); } 100% { transform: translate(210px,210px) rotate(-3deg) translate(-210px,-210px); } }
@keyframes a32_o { 0% { opacity: 0; } 100% { opacity: 1; } }
`;

function AnimatedSpeedometer() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 1440 960">
      <style>{ANIMATED_SPEEDOMETER_STYLE}</style>
      <defs>
        <filter
          id="filter0_d_4667_1690"
          x="51.5"
          y="55.5"
          width="317"
          height="317"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
          fill="none"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feColorMatrix
            in="SourceAlpha"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
            result="hardAlpha"
          />
          <feOffset dy="4" />
          <feGaussianBlur stdDeviation="2" />
          <feComposite in2="hardAlpha" operator="out" />
          <feColorMatrix values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0" />
          <feBlend in2="BackgroundImageFix" result="effect1_dropShadow_4667_1690" />
          <feBlend in="SourceGraphic" in2="effect1_dropShadow_4667_1690" result="shape" />
        </filter>
        <linearGradient
          id="Gradient-0"
          x1="0"
          y1="0"
          x2="1200"
          y2="0"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#fff" stop-opacity="0" />
          <stop offset=".2" stop-color="#fff" stop-opacity=".1" />
          <stop offset=".5" stop-color="#fff" stop-opacity=".15" />
          <stop offset=".8" stop-color="#fff" stop-opacity=".1" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="Gradient-1"
          x1="0"
          y1="40"
          x2="80"
          y2="40"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#7002fc" stop-opacity="0" />
          <stop offset="1" stop-color="#ff4200" />
        </linearGradient>
        <linearGradient
          id="Gradient-2"
          x1="0"
          y1="0"
          x2="1200"
          y2="0"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#fff" stop-opacity="0" />
          <stop offset=".2" stop-color="#fff" stop-opacity=".9" />
          <stop offset=".5" stop-color="#fff" />
          <stop offset=".8" stop-color="#fff" stop-opacity=".9" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </linearGradient>
        <radialGradient
          id="Gradient-3"
          cx="-1"
          cy="1"
          r="1"
          fx="-1"
          fy="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(0 210 -210 0 210 0)"
        >
          <stop offset="0" stop-color="#fff" stop-opacity=".08" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </radialGradient>
        <linearGradient
          id="Gradient-4"
          x1="0"
          y1="-210"
          x2="0"
          y2="210"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#fff" />
          <stop offset="1" stop-color="#fff" stop-opacity=".5" />
        </linearGradient>
        <radialGradient
          id="Gradient-5"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(0 210 -210 0 210 210)"
        >
          <stop offset="0" stop-color="#7000ff" />
          <stop offset="1" stop-color="#7000ff" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-6"
          cx="1"
          cy="1"
          r="1"
          fx="1"
          fy="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-202 0 0 -202 412 210)"
        >
          <stop offset="0" stop-color="#7002fc" />
          <stop offset="1" stop-color="#7002fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-7"
          cx="0"
          cy="1.5"
          r="1"
          fx="0"
          fy="1.5"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-143.5 143.5 -143.5 -143.5 353.5 66.5)"
        >
          <stop offset="0" stop-color="#d902fc" />
          <stop offset="1" stop-color="#7002fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-8"
          cx="-1.3"
          cy=".7"
          r="1"
          fx="-1.3"
          fy=".7"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(59 197 -197 59 151 13)"
        >
          <stop offset="0" stop-color="#02defc" />
          <stop offset="1" stop-color="#02defc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-9"
          cx="-1.4"
          cy="-0.7"
          r="1"
          fx="-1.4"
          fy="-0.7"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(176.5 57 -57 176.5 33.5 153)"
        >
          <stop offset="0" stop-color="#0239fc" />
          <stop offset=".4" stop-color="#0239fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-10"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-181 -43 67.3 -283.1 391 253)"
        >
          <stop offset="0" stop-color="#7002fc" />
          <stop offset="1" stop-color="#7002fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-11"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-137.8 137.8 -137.8 -137.8 347.8 72.2)"
        >
          <stop offset="0" stop-color="#d902fc" />
          <stop offset="1" stop-color="#7002fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-12"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-7 188 -209.7 -7.8 217 22)"
        >
          <stop offset="0" stop-color="#02defc" />
          <stop offset="1" stop-color="#02defc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-13"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(154 109 -181.3 256.1 56 101)"
        >
          <stop offset="0" stop-color="#0239fc" />
          <stop offset=".4" stop-color="#0239fc" stop-opacity="0" />
        </radialGradient>
        <radialGradient
          id="Gradient-14"
          cx="0"
          cy="0"
          r="1"
          fx="0"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-155.5 424.7 -401.6 -147 362 355)"
        >
          <stop offset="0" stop-color="#fff" stop-opacity=".6" />
          <stop offset="1" stop-color="#fff" stop-opacity=".05" />
        </radialGradient>
        <radialGradient
          id="Gradient-15"
          cx="302.5"
          cy="278"
          r="187"
          fx="302.5"
          fy="278"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset=".8" stop-color="#7002fc" />
          <stop offset="1" stop-color="#ff1d4f" stop-opacity="0" />
        </radialGradient>
        <linearGradient
          id="Gradient-16"
          x1="-210"
          y1="-210"
          x2="210"
          y2="210"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#7002fc" stop-opacity="0" />
          <stop offset="1" stop-color="#7002fc" stop-opacity=".3" />
        </linearGradient>
        <radialGradient
          id="Gradient-17"
          cx="-0.7"
          cy="0"
          r="1"
          fx="-0.7"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(296.9 296.9 -263.5 263.5 141.4 141.4)"
        >
          <stop offset=".6" stop-color="#ff4200" stop-opacity="0" />
          <stop offset=".7" stop-color="#ff4200" />
          <stop offset=".7" stop-color="#ff4200" />
        </radialGradient>
        <radialGradient
          id="Gradient-18"
          cx=".5"
          cy="0"
          r="1"
          fx=".5"
          fy="0"
          gradientUnits="userSpaceOnUse"
          gradientTransform="matrix(-420 -420 185 -185 420 420)"
        >
          <stop offset="0" stop-color="#fff" />
          <stop offset=".5" stop-color="#fff" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </radialGradient>
        <linearGradient
          id="Gradient-19"
          x1="358.2"
          y1="358.5"
          x2="61.8"
          y2="61.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#fff" />
          <stop offset=".1" stop-color="#fff" stop-opacity="0" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="Gradient-20"
          x1="312.3"
          y1="312.6"
          x2="107.5"
          y2="107.6"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stop-color="#fff" />
          <stop offset=".1" stop-color="#fff" stop-opacity="0" />
          <stop offset="1" stop-color="#fff" stop-opacity="0" />
        </linearGradient>
        <mask id="Mask-1" style={{ maskType: "alpha" }}>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M1200 .8l-1200-0.1v-0.7h1200v.7Z"
            fill="url(#Gradient-2)"
          />
        </mask>
        <mask id="Mask-2" style={{ maskType: "alpha" }}>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M210.6 15.4c-51.4 0-100.8 20.4-137.2 56.8c-36.3 36.4-56.8 85.7-56.8 137.2h194l137.2 137.2c18-18.1 32.3-39.4 42.1-63c9.7-23.5 14.7-48.8 14.7-74.2c0-25.5-5-50.7-14.7-74.3c-9.8-23.5-24.1-44.9-42.1-62.9c-18-18-39.4-32.3-62.9-42.1c-23.6-9.7-48.8-14.7-74.3-14.7v194v-194Zm0 388c0 0 0 0 0 0c0 0 0 0 0 0c0 0 0 0 0 0Z"
            fill="#fff"
            opacity="0"
          />
          <path
            d="M195 348.5c-25.5 0-50.7-5-74.2-14.8c-23.6-9.7-45-24-63-42c-18-18-32.3-39.4-42-63c-9.8-23.5-14.8-48.7-14.8-74.2h194v194Z"
            fill="#ff1d4f"
            transform="translate(210,210) rotate(-0.3) translate(-195,-154.5)"
            style={{ animation: "1.2s linear 1.2s both a8_t, .1s linear 2.4s both a8_o" }}
          />
          <path
            d="M195 348.5c-25.5 0-50.7-5-74.2-14.8c-23.6-9.7-45-24-63-42c-18-18-32.3-39.4-42-63c-9.8-23.5-14.8-48.7-14.8-74.2h194v194Z"
            fill="#ff1d4f"
            opacity="0"
            transform="translate(210,210) rotate(45) translate(-195,-154.5)"
            style={{ animation: ".3s linear 2.5s infinite both a9_t, .1s linear 2.3s both a9_o" }}
          />
        </mask>
        <mask id="Mask-3" style={{ maskType: "alpha" }}>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M210.6 15.4c-51.4 0-100.8 20.4-137.2 56.8c-36.3 36.4-56.8 85.7-56.8 137.2h194l137.2 137.2c18-18.1 32.3-39.4 42.1-63c9.7-23.5 14.7-48.8 14.7-74.2c0-25.5-5-50.7-14.7-74.3c-9.8-23.5-24.1-44.9-42.1-62.9c-18-18-39.4-32.3-62.9-42.1c-23.6-9.7-48.8-14.7-74.3-14.7v194v-194Zm0 388c0 0 0 0 0 0c0 0 0 0 0 0c0 0 0 0 0 0Z"
            fill="#fff"
          />
        </mask>
        <mask id="Mask-4" style={{ maskType: "alpha" }}>
          <path
            d="M211 55v2.5c-0.2 0-0.3 0-0.5 0c-0.2 0-0.3 0-0.5 0v-2.5c.2 0 .3 0 .5 0c.2 0 .3 0 .5 0Zm-7 2.6l-0.1-2.5c-0.4 .1-0.7 .1-1 .1l.1 2.5c.3 0 .6 0 1-0.1Zm14 .1l.1-2.5c-0.3 0-0.6 0-1-0.1l-0.1 2.5c.4 .1 .7 .1 1 .1Zm-21.3-2.1l.3 2.5c-0.4 0-0.7 .1-1 .1l-0.3-2.5c.4 0 .7-0.1 1-0.1Zm28.6 .1l-0.3 2.5c-0.3 0-0.6-0.1-1-0.1l.3-2.5c.3 0 .6 .1 1 .1Zm-35.3 3.2l-0.3-2.5c-0.4 0-0.7 .1-1.1 .1l.4 2.5c.3 0 .7-0.1 1-0.1Zm42 .1l.4-2.5c-0.4 0-0.7-0.1-1.1-0.1l-0.3 2.5c.3 0 .7 .1 1 .1Zm-49.4-1.5l.4 2.5c-0.3 0-0.6 .1-0.9 .2l-0.5-2.5c.3-0.1 .7-0.1 1-0.2Zm56.8 .2l-0.5 2.5c-0.3-0.1-0.6-0.2-0.9-0.2l.4-2.5c.3 .1 .7 .1 1 .2Zm-63.2 3.7l-0.6-2.4c-0.3 0-0.7 .1-1 .2l.6 2.4c.3-0.1 .6-0.1 1-0.2Zm69.6 .2l.6-2.4c-0.3-0.1-0.7-0.2-1-0.2l-0.6 2.4c.4 .1 .7 .1 1 .2Zm-77.1-0.9l.7 2.4c-0.4 .1-0.7 .2-1 .3l-0.7-2.4c.3-0.1 .7-0.2 1-0.3Zm84.6 .3l-0.7 2.4c-0.3-0.1-0.6-0.2-1-0.3l.7-2.4c.3 .1 .7 .2 1 .3Zm-90.6 4.2l-0.8-2.4c-0.4 .1-0.7 .2-1 .3l.8 2.4c.3-0.1 .6-0.2 1-0.3Zm96.6 .3l.8-2.4c-0.3-0.1-0.6-0.2-1-0.3l-0.8 2.4c.4 .1 .7 .2 1 .3Zm-104.2-0.3l.9 2.3c-0.3 .2-0.6 .3-0.9 .4l-0.9-2.3c.3-0.1 .6-0.3 .9-0.4Zm111.7 .4l-0.9 2.3c-0.3-0.1-0.6-0.2-0.9-0.4l.9-2.3c.3 .1 .6 .3 .9 .4Zm-117.3 4.6l-1-2.3c-0.3 .2-0.6 .3-0.9 .4l1 2.3c.3-0.1 .6-0.3 .9-0.4Zm122.9 .4l1-2.3c-0.3-0.1-0.6-0.2-0.9-0.4l-1 2.3c.3 .1 .6 .3 .9 .4Zm-130.3 .3l1.1 2.3c-0.3 .1-0.6 .3-0.9 .4l-1.1-2.2c.3-0.2 .6-0.3 .9-0.5Zm137.7 .5l-1.1 2.2c-0.3-0.1-0.6-0.3-0.9-0.4l1.1-2.3c.3 .2 .6 .3 .9 .5Zm-142.8 5l-1.2-2.1c-0.3 .1-0.6 .3-0.9 .5l1.2 2.1c.3-0.1 .6-0.3 .9-0.5Zm147.9 .5l1.2-2.1c-0.3-0.2-0.6-0.4-0.9-0.5l-1.2 2.1c.3 .2 .6 .4 .9 .5Zm-155.3 1l1.3 2.1c-0.3 .2-0.6 .3-0.9 .5l-1.3-2.1c.3-0.2 .6-0.4 .9-0.5Zm162.7 .5l-1.3 2.1c-0.3-0.2-0.6-0.3-0.9-0.5l1.3-2.1c.3 .1 .6 .3 .9 .5Zm-167.3 5.4l-1.4-2.1c-0.3 .2-0.6 .4-0.8 .6l1.4 2.1c.3-0.2 .5-0.4 .8-0.6Zm171.8 .6l1.4-2.1c-0.2-0.2-0.5-0.4-0.8-0.6l-1.4 2.1c.3 .2 .5 .4 .8 .6Zm-179 1.5l1.5 2c-0.3 .2-0.5 .4-0.8 .6l-1.5-2c.3-0.2 .5-0.4 .8-0.6Zm186.2 .6l-1.5 2c-0.3-0.2-0.5-0.4-0.8-0.6l1.5-2c.3 .2 .5 .4 .8 .6Zm-190.2 5.7l-1.6-1.9c-0.3 .2-0.5 .4-0.8 .7l1.6 1.9c.3-0.2 .5-0.4 .8-0.7Zm194.2 .7l1.6-1.9c-0.3-0.3-0.5-0.5-0.8-0.7l-1.6 1.9c.3 .3 .5 .5 .8 .7Zm-201.2 2.1l1.7 1.8c-0.3 .3-0.5 .5-0.8 .7l-1.7-1.8c.3-0.3 .5-0.5 .8-0.7Zm208.2 .7l-1.7 1.8c-0.3-0.2-0.5-0.4-0.8-0.7l1.7-1.8c.3 .2 .5 .4 .8 .7Zm-211.6 6l-1.8-1.8c-0.2 .2-0.5 .5-0.7 .7l1.8 1.8c.2-0.3 .4-0.5 .7-0.7Zm214.9 .7l1.8-1.8c-0.2-0.2-0.5-0.5-0.7-0.7l-1.8 1.8c.3 .2 .5 .4 .7 .7Zm-221.6 2.6l1.8 1.7c-0.2 .3-0.4 .5-0.7 .8l-1.8-1.7c.2-0.3 .4-0.5 .7-0.8Zm228.3 .8l-1.8 1.7c-0.3-0.3-0.5-0.5-0.7-0.8l1.8-1.7c.3 .3 .5 .5 .7 .8Zm-231.1 6.2l-1.9-1.6c-0.3 .3-0.5 .5-0.7 .8l1.9 1.6c.3-0.3 .5-0.5 .7-0.8Zm233.9 .8l1.9-1.6c-0.2-0.3-0.4-0.5-0.7-0.8l-1.9 1.6c.2 .3 .4 .5 .7 .8Zm-240.3 3.2l2 1.5c-0.2 .3-0.4 .5-0.6 .8l-2-1.5c.2-0.3 .4-0.5 .6-0.8Zm246.6 .8l-2 1.5c-0.2-0.3-0.4-0.5-0.6-0.8l2-1.5c.1 .1 .2 .3 .3 .4c.1 .1 .2 .3 .3 .4Zm-248.7 6.4l-2.1-1.4c-0.2 .2-0.4 .5-0.6 .8l2.1 1.4c.2-0.3 .4-0.5 .6-0.8Zm250.8 .8l2.1-1.4c-0.2-0.3-0.4-0.6-0.6-0.8l-2.1 1.4c.2 .3 .4 .5 .6 .8Zm-256.8 3.7l2.1 1.3c-0.2 .3-0.3 .6-0.5 .9l-2.1-1.3c.1-0.3 .3-0.6 .5-0.9Zm262.7 .9l-2.1 1.3c-0.1-0.1-0.2-0.3-0.2-0.4c-0.1-0.2-0.2-0.3-0.3-0.5l2.1-1.3c.2 .3 .4 .6 .5 .9Zm-264.2 6.5l-2.1-1.2c-0.2 .3-0.4 .6-0.5 .9l2.1 1.2c.2-0.3 .4-0.6 .5-0.9Zm265.7 .9l2.1-1.2c0-0.2-0.1-0.3-0.2-0.4c-0.1-0.2-0.2-0.4-0.3-0.5l-2.1 1.2c.1 .3 .3 .6 .5 .9Zm-271.2 4.2l2.2 1.1c-0.1 .3-0.3 .6-0.4 .9l-2.3-1.1c.2-0.3 .3-0.6 .5-0.9Zm276.7 .9l-2.3 1.1c-0.1-0.3-0.3-0.6-0.4-0.9l2.2-1.1c.2 .3 .3 .6 .5 .9Zm-277.5 6.5l-2.3-1c-0.1 .3-0.2 .6-0.4 .9l2.3 1c.1-0.3 .3-0.6 .4-0.9Zm278.2 .9l2.3-1c-0.2-0.3-0.3-0.6-0.4-0.9l-2.3 1c.1 .3 .3 .6 .4 .9Zm-283.2 4.7l2.3 .9c-0.1 .3-0.2 .6-0.4 .9l-2.3-0.9c.1-0.3 .3-0.6 .4-0.9Zm288.2 .9l-2.3 .9c-0.2-0.3-0.3-0.6-0.4-0.9l2.3-0.9c.1 .3 .3 .6 .4 .9Zm-288.3 6.6l-2.4-0.8c-0.1 .3-0.2 .6-0.3 1l2.4 .8c.1-0.4 .2-0.7 .3-1Zm288.3 1l2.4-0.8c-0.1-0.4-0.2-0.7-0.3-1l-2.4 .8c.1 .3 .2 .6 .3 1Zm-292.8 5l2.4 .7c-0.1 .3-0.2 .6-0.3 1l-2.4-0.7c.1-0.3 .2-0.7 .3-1Zm297.3 1l-2.4 .7c-0.1-0.4-0.2-0.7-0.3-1l2.4-0.7c.1 .3 .2 .7 .3 1Zm-296.7 6.5l-2.4-0.6c-0.1 .3-0.2 .7-0.2 1l2.4 .6c.1-0.4 .1-0.7 .2-1Zm296 1l2.4-0.6c0-0.2 0-0.3-0.1-0.5c0-0.1 0-0.3-0.1-0.5l-2.4 .6c.1 .3 .1 .6 .2 1Zm-299.9 5.4l2.5 .5c-0.1 .3-0.2 .6-0.2 .9l-2.5-0.4c.1-0.3 .1-0.7 .2-1Zm303.8 1l-2.5 .4c0-0.1 0-0.3-0.1-0.5c0-0.2 0-0.3-0.1-0.4l2.5-0.5c.1 .3 .1 .7 .2 1Zm-302.5 6.4l-2.5-0.4c0 .4-0.1 .7-0.1 1.1l2.5 .3c0-0.3 .1-0.7 .1-1Zm301.1 1l2.5-0.3c0-0.4-0.1-0.7-0.1-1.1l-2.5 .4c0 .3 .1 .7 .1 1Zm-304.4 5.7l2.5 .3c0 .3-0.1 .6-0.1 1l-2.5-0.3c0-0.3 .1-0.6 .1-1Zm307.7 1l-2.5 .3c0-0.4-0.1-0.7-0.1-1l2.5-0.3c0 .2 0 .4 .1 .6c0 .1 0 .3 0 .4Zm-305.7 6.3l-2.5-0.1c0 .3 0 .6-0.1 1l2.5 .1c.1-0.4 .1-0.7 .1-1Zm303.7 1l2.5-0.1c0-0.2-0.1-0.3-0.1-0.4c0-0.2 0-0.4 0-0.6l-2.5 .1c0 .3 0 .6 .1 1Zm-303.9 6.5c0-0.2 0-0.3 0-0.5h-2.5c0 .2 0 .3 0 .5Zm304 .5v-0.1v-0.4c0-0.2 0-0.3 0-0.5h2.5c0 .1 0 .2 0 .3v.2c0 .2 0 .3 0 .5Zm-0.2 7l2.5 .1c0-0.3 0-0.6 .1-1l-2.5-0.1c-0.1 .4-0.1 .7-0.1 1Zm2 7.3l-2.5-0.3c0-0.3 .1-0.6 .1-1l2.5 .3c0 .3-0.1 .6-0.1 1Zm-3.3 6.7l2.5 .4c0-0.4 .1-0.7 .1-1.1l-2.5-0.3c0 .3-0.1 .7-0.1 1Zm1.3 7.4l-2.5-0.5c.1-0.3 .2-0.6 .2-0.9l2.5 .4c0 .2-0.1 .4-0.1 .6c0 .1-0.1 .3-0.1 .4Zm-3.9 6.4l2.4 .6c.1-0.3 .2-0.7 .2-1l-2.4-0.6c-0.1 .4-0.1 .7-0.2 1Zm.6 7.5l-2.4-0.7c.1-0.3 .2-0.6 .3-1l2.4 .7l-0.1 .2c0 .1 0 .1 0 .2c-0.1 .2-0.1 .4-0.2 .6Zm-4.5 6l2.4 .8c.1-0.3 .2-0.6 .3-1l-2.4-0.8c-0.1 .4-0.2 .7-0.3 1Zm-0.1 7.5l-2.3-0.9c.1-0.3 .2-0.6 .4-0.9l2.3 .9c-0.1 .1-0.1 .2-0.2 .3c0 .1 0 .2-0.1 .3c0 .1 0 .2-0.1 .3Zm-5 5.6l2.3 1c.1-0.3 .2-0.6 .4-0.9l-2.3-1c-0.1 .3-0.3 .6-0.4 .9Zm-0.8 7.4l-2.2-1.1c.1-0.3 .3-0.6 .4-0.9l2.3 1.1c-0.2 .3-0.3 .6-0.5 .9Zm-5.5 5.1l2.1 1.2c.1-0.1 .2-0.2 .2-0.3c.1-0.2 .2-0.4 .3-0.6l-2.1-1.2c-0.2 .3-0.4 .6-0.5 .9Zm-1.5 7.4l-2.1-1.3c.2-0.3 .3-0.6 .5-0.9l2.1 1.3c-0.1 .3-0.3 .6-0.5 .9Zm-6 4.5l2.1 1.4c.2-0.2 .4-0.5 .6-0.8l-2.1-1.4c-0.2 .3-0.4 .5-0.6 .8Zm-2.1 7.2l-2-1.5c.1-0.2 .3-0.3 .4-0.5c.1-0.1 .1-0.2 .2-0.3l2 1.5c-0.2 .3-0.4 .5-0.6 .8Zm-6.4 4l1.9 1.6c.3-0.3 .5-0.5 .7-0.8l-1.9-1.6c-0.1 .1-0.3 .3-0.4 .4c-0.1 .1-0.2 .3-0.3 .4Zm-2.8 7l-1.8-1.7c.2-0.3 .4-0.5 .7-0.8l1.8 1.7c-0.2 .3-0.4 .5-0.7 .8Zm-4.2 4.4c-0.1 .1-0.2 .2-0.4 .3l-1.7-1.7c.1-0.1 .2-0.3 .3-0.4Z"
            fill="#fff"
            fill-rule="evenodd"
          />
        </mask>
        <mask id="Mask-5" style={{ maskType: "alpha" }}>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M209.4 75c.2 0 .4 0 .6 0c.2 0 .3 0 .5 0c.2 0 .4 0 .6 0l-0.3 40c-0.3 0-0.5 0-0.8 0c-0.3 0-0.5 0-0.8 0l-0.3-40c.2 0 .3 0 .5 0Zm-9.6 .4c.7-0.1 1.4-0.1 2.1-0.2l2.4 40c-0.5 0-1 0-1.5 .1Zm-9.2 1c.7-0.1 1.4-0.2 2.1-0.3l5.1 39.7c-0.5 0-1 .1-1.5 .2Zm-9.1 1.6c.7-0.1 1.4-0.3 2.1-0.4l7.8 39.2c-0.5 .1-1 .2-1.5 .3Zm-8.9 2.3c.6-0.2 1.3-0.4 2-0.6l10.5 38.6c-0.5 .1-1 .3-1.5 .4Zm-8.8 2.9c.7-0.3 1.3-0.5 2-0.8l13.1 37.8c-0.5 .2-1 .3-1.4 .5Zm-8.6 3.4c.7-0.3 1.3-0.6 2-0.8l15.6 36.7c-0.4 .2-0.9 .4-1.3 .6Zm-8.3 4c.6-0.3 1.3-0.6 1.9-0.9l18.2 35.6c-0.5 .2-0.9 .4-1.4 .7Zm-8 4.6c.6-0.3 1.2-0.7 1.8-1.1l20.6 34.3c-0.5 .3-0.9 .5-1.3 .8Zm-7.6 5.1c.5-0.4 1.1-0.8 1.7-1.2l22.8 32.9c-0.4 .2-0.8 .5-1.2 .8Zm-7.3 5.7c.5-0.5 1.1-0.9 1.6-1.4l25 31.2c-0.4 .3-0.7 .7-1.1 1Zm-6.9 6.1c.5-0.5 1-1 1.5-1.5l27.1 29.5c-0.4 .3-0.7 .6-1.1 1Zm-6.5 6.5c.5-0.5 1-1 1.5-1.5l29 27.5c-0.4 .4-0.7 .7-1 1.1Zm-6 7c.5-0.5 .9-1.1 1.4-1.6l30.8 25.5c-0.3 .4-0.7 .7-1 1.1Zm-5.5 7.4c.4-0.6 .8-1.2 1.2-1.7l32.5 23.3c-0.3 .4-0.6 .8-0.8 1.2Zm-5 7.7c.4-0.6 .8-1.2 1.1-1.8l34 21.1c-0.3 .4-0.5 .8-0.8 1.3Zm-4.4 8.1c.3-0.6 .6-1.3 .9-1.9l35.4 18.7c-0.3 .5-0.5 .9-0.7 1.4Zm-3.9 8.4c.2-0.7 .5-1.3 .8-2l36.5 16.3c-0.2 .4-0.4 .9-0.6 1.3Zm-3.4 8.6c.3-0.7 .5-1.3 .8-2l37.5 13.7c-0.2 .4-0.3 .9-0.5 1.4Zm-2.7 8.8c.2-0.7 .4-1.3 .6-2l38.4 11c-0.1 .5-0.3 1-0.4 1.5Zm-2.1 9c.1-0.7 .3-1.4 .4-2.1l39.1 8.4c-0.1 .5-0.2 1-0.3 1.5Zm-1.5 9.1c.1-0.7 .2-1.4 .3-2.1l39.6 5.7c-0.1 .5-0.2 1-0.2 1.5Zm-0.9 9.2c.1-0.7 .1-1.4 .2-2.1l39.9 3c-0.1 .5-0.1 1-0.1 1.5Zm-0.2 7c0 .2 0 .4 0 .6c0 .2 0 .3 0 .5h40c0-0.3 0-0.5 0-0.8Zm234.4 92.5c-0.5 .5-1 1-1.5 1.5l-29-27.5c.4-0.4 .7-0.7 1-1.1Zm6-7c-0.5 .5-0.9 1.1-1.4 1.6l-30.8-25.5c.3-0.4 .7-0.7 1-1.1Zm5.5-7.4c-0.4 .6-0.8 1.2-1.2 1.7l-32.5-23.3c.3-0.4 .6-0.8 .8-1.2Zm5-7.7c-0.4 .6-0.8 1.2-1.1 1.8l-34-21.1c.3-0.4 .5-0.8 .8-1.3Zm4.4-8.1c-0.3 .6-0.6 1.3-0.9 1.9l-35.4-18.7c.3-0.5 .5-0.9 .7-1.3Zm3.9-8.4c-0.2 .7-0.5 1.3-0.8 2l-36.5-16.3c.2-0.4 .4-0.9 .6-1.3Zm3.4-8.6c-0.3 .7-0.5 1.3-0.8 2l-37.5-13.7c.2-0.4 .3-0.9 .5-1.4Zm2.7-8.8c-0.2 .7-0.4 1.3-0.6 2l-38.4-11c.1-0.5 .3-1 .4-1.5Zm2.1-9c-0.1 .7-0.3 1.4-0.4 2.1l-39.1-8.4c.1-0.5 .2-1 .3-1.5Zm1.5-9.1c-0.1 .7-0.2 1.4-0.3 2.1l-39.6-5.7c.1-0.5 .2-1 .2-1.5Zm.9-9.2c-0.1 .7-0.1 1.4-0.2 2.1l-39.9-3c.1-0.5 .1-1 .1-1.5Zm.2-7c0-0.4 0-0.7 0-1.1c0-0.4 0-0.7 0-1.1l-40 .3c0 .3 0 .5 0 .8c0 .3 0 .5 0 .8l40 .3Zm-0.4-11.3c.1 .7 .1 1.4 .2 2.1l-40 2.4c0-0.5 0-1-0.1-1.5Zm-1-9.2c.1 .7 .2 1.4 .3 2.1l-39.7 5.1c0-0.5-0.1-1-0.2-1.5Zm-1.6-9.1c.1 .7 .3 1.4 .4 2.1l-39.2 7.8c-0.1-0.5-0.2-1-0.3-1.5Zm-2.3-8.9c.2 .6 .4 1.3 .6 2l-38.6 10.5c-0.1-0.5-0.3-1-0.4-1.5Zm-2.9-8.8c.3 .7 .5 1.3 .8 2l-37.8 13.1c-0.2-0.5-0.3-1-0.5-1.4Zm-3.4-8.6c.3 .7 .6 1.3 .8 2l-36.7 15.6c-0.2-0.4-0.4-0.9-0.6-1.3Zm-4-8.3c.3 .6 .6 1.3 .9 1.9l-35.6 18.2c-0.2-0.5-0.4-0.9-0.7-1.4Zm-4.6-8c.3 .6 .7 1.2 1.1 1.8l-34.3 20.6c-0.3-0.5-0.5-0.9-0.8-1.3Zm-5.1-7.6c.4 .5 .8 1.1 1.2 1.7l-32.9 22.8c-0.2-0.4-0.5-0.8-0.8-1.2Zm-5.7-7.3c.5 .5 .9 1.1 1.4 1.6l-31.2 25c-0.3-0.4-0.7-0.7-1-1.1Zm-6.1-6.9c.5 .5 1 1 1.5 1.5l-29.5 27.1c-0.3-0.4-0.6-0.7-1-1.1Zm-6.5-6.5c.5 .5 1 1 1.5 1.5l-27.5 29c-0.4-0.4-0.7-0.7-1.1-1Zm-7-6c.5 .5 1.1 .9 1.6 1.4l-25.5 30.8c-0.4-0.3-0.7-0.7-1.1-1Zm-7.4-5.5c.6 .4 1.2 .8 1.7 1.2l-23.3 32.5c-0.4-0.3-0.8-0.6-1.2-0.8Zm-7.7-5c.6 .4 1.2 .8 1.8 1.1l-21.1 34c-0.4-0.3-0.8-0.5-1.3-0.8Zm-8.1-4.4c.6 .3 1.3 .6 1.9 .9l-18.7 35.4c-0.5-0.3-0.9-0.5-1.3-0.7Zm-8.4-3.9c.7 .2 1.3 .5 2 .8l-16.3 36.5c-0.4-0.2-0.9-0.4-1.3-0.6Zm-8.6-3.4c.7 .3 1.3 .5 2 .8l-13.7 37.5c-0.4-0.2-0.9-0.3-1.4-0.5Zm-8.8-2.7c.7 .2 1.4 .4 2 .6l-11 38.4c-0.5-0.1-1-0.3-1.5-0.4Zm-9-2.1c.7 .1 1.4 .3 2.1 .4l-8.4 39.1c-0.5-0.1-1-0.2-1.5-0.3Zm-9.1-1.5c.7 .1 1.4 .2 2.1 .3l-5.7 39.6c-0.5-0.1-1-0.2-1.5-0.2Zm-9.2-0.9c.7 .1 1.4 .1 2.1 .2l-3 39.9c-0.5-0.1-1-0.1-1.5-0.1Z"
            fill="#fff"
          />
        </mask>
        <mask id="Mask-6" style={{ maskType: "alpha" }}>
          <ellipse cx="210" cy="210" fill="#fff" rx="210" ry="210" />
        </mask>
      </defs>
      <g
        fill="none"
        opacity="0"
        transform="translate(120,440)"
        style={{ animation: ".8s linear both a0_o" }}
      >
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          d="M1200 .8l-1200-0.1v-0.7h1200v.7Z"
          fill="url(#Gradient-0)"
          transform="translate(0,40)"
        />
        <g mask="url(#Mask-1)" transform="translate(0,40)">
          <rect
            width="80"
            height="80"
            fill="url(#Gradient-1)"
            transform="translate(-160,0) translate(-40,-40)"
            style={{ animation: "3.2s linear .8s infinite both a1_t" }}
          />
          <rect
            width="80"
            height="80"
            fill="url(#Gradient-1)"
            transform="translate(-160,0) translate(-40,-40)"
            style={{ animation: "3.2s linear 1.6s infinite both a2_t" }}
          />
        </g>
      </g>
      <g
        fill="none"
        opacity="0"
        transform="translate(510,270)"
        style={{ animation: ".8s linear both a3_o" }}
      >
        <g>
          <ellipse
            fill="url(#Gradient-3)"
            rx="210"
            ry="210"
            transform="translate(210,210)"
            style={{ animation: "42.3s linear both a4_f" }}
          />
          <ellipse
            fill="url(#Gradient-3)"
            rx="210"
            ry="210"
            transform="translate(210,210)"
            style={{ animation: "42.3s linear both a5_f" }}
          />
          <ellipse fill="url(#Gradient-3)" rx="210" ry="210" transform="translate(210,210)" />
          <ellipse
            stroke="url(#Gradient-4)"
            stroke-opacity=".1"
            rx="209.5"
            ry="209.5"
            transform="translate(210,210)"
          />
        </g>
        <path
          d="M0 210c0-55.7 22.1-109.1 61.5-148.5c39.4-39.4 92.8-61.5 148.5-61.5c55.7 0 109.1 22.1 148.5 61.5c39.4 39.4 61.5 92.8 61.5 148.5h-210h-210Z"
          fill="url(#Gradient-5)"
          opacity=".02"
          style={{ animation: "2s linear 1.2s both a6_o" }}
        />
        <g
          opacity=".02"
          transform="translate(7.2,7.2)"
          style={{ animation: "2s linear 1.4s both a7_o" }}
        >
          <ellipse
            stroke="url(#Gradient-6)"
            stroke-width="1.5"
            rx="202.8"
            ry="202.8"
            transform="translate(202.8,202.8)"
          />
          <ellipse
            stroke="url(#Gradient-7)"
            stroke-width="1.5"
            rx="202.8"
            ry="202.8"
            transform="translate(202.8,202.8)"
          />
          <ellipse
            stroke="url(#Gradient-8)"
            stroke-width="1.5"
            rx="202.8"
            ry="202.8"
            transform="translate(202.8,202.8)"
          />
          <ellipse
            stroke="url(#Gradient-9)"
            stroke-width="1.5"
            rx="202.8"
            ry="202.8"
            transform="translate(202.8,202.8)"
          />
        </g>
        <g mask="url(#Mask-2)">
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M404 210c0 107.1-86.9 194-194 194c-107.1 0-194-86.9-194-194c0-107.1 86.9-194 194-194c107.1 0 194 86.9 194 194Zm-194 170c93.9 0 170-76.1 170-170c0-93.9-76.1-170-170-170c-93.9 0-170 76.1-170 170c0 93.9 76.1 170 170 170Z"
            fill="url(#Gradient-10)"
          />
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M404 210c0 107.1-86.9 194-194 194c-107.1 0-194-86.9-194-194c0-107.1 86.9-194 194-194c107.1 0 194 86.9 194 194Zm-194 170c93.9 0 170-76.1 170-170c0-93.9-76.1-170-170-170c-93.9 0-170 76.1-170 170c0 93.9 76.1 170 170 170Z"
            fill="url(#Gradient-11)"
          />
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M404 210c0 107.1-86.9 194-194 194c-107.1 0-194-86.9-194-194c0-107.1 86.9-194 194-194c107.1 0 194 86.9 194 194Zm-194 170c93.9 0 170-76.1 170-170c0-93.9-76.1-170-170-170c-93.9 0-170 76.1-170 170c0 93.9 76.1 170 170 170Z"
            fill="url(#Gradient-12)"
          />
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M404 210c0 107.1-86.9 194-194 194c-107.1 0-194-86.9-194-194c0-107.1 86.9-194 194-194c107.1 0 194 86.9 194 194Zm-194 170c93.9 0 170-76.1 170-170c0-93.9-76.1-170-170-170c-93.9 0-170 76.1-170 170c0 93.9 76.1 170 170 170Z"
            fill="url(#Gradient-13)"
          />
        </g>
        <g opacity=".2" mask="url(#Mask-3)" style={{ animation: "2s linear 1.2s both a10_o" }}>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M380 210.8c0-0.3 0-0.5 0-0.8c0-0.3 0-0.5 0-0.8l24-0.1c0 .3 0 .6 0 .9c0 .3 0 .6 0 .9l-24-0.1Zm-0.1-6.6l24-0.8c0-0.5 0-1.1-0.1-1.7l-24 1c.1 .5 .1 1 .1 1.5Zm-0.3-6.5l23.9-1.7c0-0.6-0.1-1.1-0.1-1.7l-23.9 1.9c0 .5 0 1 .1 1.5Zm-0.6-6.5l23.8-2.6c0-0.5-0.1-1.1-0.2-1.7l-23.8 2.8c.1 .5 .1 1 .2 1.5Zm-0.9-6.4l23.8-3.5c-0.1-0.6-0.2-1.2-0.3-1.7l-23.7 3.7c.1 .5 .2 1 .2 1.5Zm-1-6.4l23.5-4.5c-0.1-0.5-0.2-1.1-0.3-1.7l-23.5 4.7c.1 .5 .2 1 .3 1.5Zm-1.4-6.4l23.4-5.4c-0.1-0.5-0.2-1.1-0.4-1.6l-23.3 5.5c.1 .5 .2 1 .3 1.5Zm-1.5-6.3l23.1-6.3c-0.1-0.5-0.3-1.1-0.4-1.6l-23.1 6.4c.1 .5 .2 1 .4 1.5Zm-1.9-6.3l23-7.1c-0.2-0.6-0.4-1.1-0.6-1.7l-22.8 7.4c.1 .5 .3 .9 .4 1.4Zm-2-6.2l22.6-8c-0.2-0.5-0.4-1.1-0.6-1.6l-22.5 8.2c.2 .5 .3 1 .5 1.4Zm-2.3-6.1l22.3-8.9c-0.2-0.5-0.4-1-0.6-1.5l-22.3 9c.2 .5 .4 1 .6 1.4Zm-2.5-6l21.9-9.7c-0.2-0.5-0.5-1.1-0.7-1.6l-21.8 10c.2 .4 .4 .9 .6 1.3Zm-2.8-5.9l21.6-10.5c-0.3-0.6-0.5-1.1-0.8-1.6l-21.5 10.8c.3 .4 .5 .9 .7 1.3Zm-3-5.8l21.2-11.4c-0.3-0.5-0.6-1-0.9-1.5l-21 11.6c.3 .5 .5 .9 .7 1.3Zm-3.2-5.6l20.7-12.2c-0.3-0.5-0.6-1-0.9-1.5l-20.5 12.4c.2 .4 .5 .8 .7 1.3Zm-3.4-5.6l20.2-13c-0.3-0.4-0.6-0.9-0.9-1.4l-20.1 13.2c.3 .4 .5 .8 .8 1.2Zm-3.6-5.4l19.7-13.7c-0.4-0.5-0.7-1-1-1.4l-19.6 13.9c.3 .4 .6 .8 .9 1.2Zm-3.8-5.3l19.1-14.4c-0.3-0.5-0.7-1-1-1.4l-19 14.6c.3 .4 .6 .8 .9 1.2Zm-4.1-5.1l18.6-15.2c-0.4-0.4-0.7-0.9-1.1-1.3l-18.4 15.4c.3 .3 .6 .7 .9 1.1Zm-4.2-4.9l18-15.9c-0.4-0.5-0.8-0.9-1.1-1.3l-17.9 16c.4 .4 .7 .8 1 1.2Zm-4.4-4.8l17.4-16.6c-0.4-0.4-0.8-0.8-1.2-1.2l-17.2 16.7c.3 .3 .7 .7 1 1.1Zm-4.6-4.7l16.7-17.2c-0.4-0.4-0.8-0.8-1.2-1.2l-16.6 17.4c.4 .3 .8 .7 1.1 1Zm-4.7-4.4l16-17.9c-0.4-0.3-0.8-0.7-1.3-1.1l-15.9 18c.4 .3 .8 .7 1.2 1Zm-5-4.3l15.4-18.4c-0.4-0.4-0.9-0.7-1.3-1.1l-15.2 18.6c.4 .3 .8 .6 1.1 .9Zm-5-4.1l14.6-19c-0.4-0.3-0.9-0.7-1.4-1l-14.4 19.1c.4 .3 .8 .6 1.2 .9Zm-5.3-3.8l13.9-19.6c-0.4-0.3-0.9-0.6-1.4-1l-13.7 19.7c.4 .3 .8 .6 1.2 .9Zm-5.4-3.7l13.2-20.1c-0.5-0.3-1-0.6-1.4-0.9l-13 20.2c.4 .3 .8 .5 1.2 .8Zm-5.5-3.5l12.4-20.5c-0.5-0.3-1-0.6-1.5-0.9l-12.2 20.7c.5 .2 .9 .5 1.3 .7Zm-5.6-3.2l11.6-21c-0.5-0.3-1-0.6-1.5-0.9l-11.4 21.2c.4 .2 .8 .4 1.3 .7Zm-5.8-3l10.8-21.5c-0.5-0.3-1-0.5-1.6-0.8l-10.5 21.6c.4 .2 .9 .4 1.3 .7Zm-5.9-2.9l10-21.8c-0.5-0.2-1.1-0.5-1.6-0.7l-9.7 21.9c.4 .2 .9 .4 1.3 .6Zm-5.9-2.5l9-22.3c-0.5-0.2-1-0.4-1.5-0.6l-8.9 22.3c.4 .2 .9 .4 1.4 .6Zm-6.1-2.4l8.2-22.5c-0.5-0.2-1.1-0.4-1.6-0.6l-8 22.6c.4 .2 .9 .3 1.4 .5Zm-6.2-2.1l7.4-22.8c-0.6-0.2-1.1-0.4-1.7-0.6l-7.1 23c.5 .1 .9 .3 1.4 .4Zm-6.2-1.9l6.4-23.1c-0.5-0.1-1.1-0.3-1.6-0.4l-6.3 23.1c.5 .2 1 .3 1.5 .4Zm-6.3-1.6l5.5-23.3c-0.5-0.2-1.1-0.3-1.6-0.4l-5.4 23.4c.5 .1 1 .2 1.5 .3Zm-6.4-1.4l4.7-23.5c-0.6-0.1-1.2-0.2-1.7-0.3l-4.5 23.5c.5 .1 1 .2 1.5 .3Zm-6.4-1.1l3.7-23.7c-0.5-0.1-1.1-0.2-1.7-0.3l-3.5 23.8c.5 0 1 .1 1.5 .2Zm-6.4-0.9l2.8-23.8c-0.6-0.1-1.2-0.2-1.7-0.2l-2.6 23.8c.5 .1 1 .1 1.5 .2Zm-6.5-0.6l1.9-24c-0.6 0-1.1-0.1-1.7-0.1l-1.7 23.9c.5 .1 1 .1 1.5 .2Zm-6.5-0.4l1-24c-0.6 0-1.2-0.1-1.7-0.1l-0.8 24c.5 0 1 0 1.5 .1Zm-6.5-0.2l.1-24c-0.3 0-0.6 0-0.9 0c-0.3 0-0.6 0-0.9 0l.1 24c.3 0 .5 0 .8 0c.3 0 .5 0 .8 0Zm-6.6 .1l-0.8-24c-0.5 0-1.1 .1-1.7 .1l1 24c.5-0.1 1-0.1 1.5-0.1Zm-6.5 .3l-1.7-23.9c-0.6 0-1.1 .1-1.7 .1l1.9 24c.5-0.1 1-0.1 1.5-0.2Zm-6.5 .6l-2.6-23.8c-0.5 0-1.1 .1-1.7 .2l2.8 23.8c.5-0.1 1-0.1 1.5-0.2Zm-6.4 .9l-3.5-23.8c-0.6 .1-1.2 .2-1.7 .3l3.7 23.7c.5-0.1 1-0.2 1.5-0.2Zm-6.4 1l-4.5-23.5c-0.5 .1-1.1 .2-1.7 .3l4.7 23.5c.5-0.1 1-0.2 1.5-0.3Zm-6.4 1.4l-5.4-23.4c-0.5 .1-1.1 .2-1.6 .4l5.5 23.3c.5-0.1 1-0.2 1.5-0.3Zm-6.3 1.5l-6.3-23.1c-0.5 .1-1.1 .3-1.6 .4l6.4 23.1c.5-0.1 1-0.2 1.5-0.4Zm-6.3 1.9l-7.1-23c-0.6 .2-1.1 .4-1.7 .6l7.4 22.8c.5-0.1 .9-0.3 1.4-0.4Zm-6.2 2l-8-22.6c-0.5 .2-1.1 .4-1.6 .6l8.2 22.5c.5-0.2 1-0.3 1.4-0.5Zm-6.1 2.3l-8.9-22.3c-0.5 .2-1 .4-1.5 .6l9 22.3c.5-0.2 1-0.4 1.4-0.6Zm-6 2.5l-9.7-21.9c-0.5 .2-1.1 .5-1.6 .7l10 21.8c.4-0.2 .9-0.4 1.3-0.6Zm-5.9 2.8l-10.5-21.6c-0.6 .3-1.1 .5-1.6 .8l10.8 21.5c.4-0.3 .9-0.5 1.3-0.7Zm-5.8 3l-11.4-21.2c-0.5 .3-1 .6-1.5 .9l11.6 21c.5-0.3 .9-0.5 1.3-0.7Zm-5.6 3.2l-12.2-20.7c-0.5 .3-1 .6-1.5 .9l12.4 20.5c.4-0.2 .8-0.5 1.3-0.7Zm-5.6 3.4l-13-20.2c-0.4 .3-0.9 .6-1.4 .9l13.2 20.1c.4-0.3 .8-0.5 1.2-0.8Zm-5.4 3.6l-13.7-19.7c-0.5 .4-1 .7-1.4 1l13.9 19.6c.4-0.3 .8-0.6 1.2-0.9Zm-5.3 3.8l-14.4-19.1c-0.5 .3-1 .7-1.4 1l14.6 19c.4-0.3 .8-0.6 1.2-0.9Zm-5.1 4.1l-15.2-18.6c-0.4 .4-0.9 .7-1.3 1.1l15.4 18.4c.3-0.3 .7-0.6 1.1-0.9Zm-4.9 4.2l-15.9-18c-0.5 .4-0.9 .8-1.3 1.1l16 17.9c.4-0.3 .8-0.7 1.2-1Zm-4.8 4.4l-16.6-17.4c-0.4 .4-0.8 .8-1.2 1.2l16.7 17.2c.3-0.3 .7-0.7 1.1-1Zm-4.7 4.6l-17.2-16.7c-0.4 .4-0.8 .8-1.2 1.2l17.4 16.6c.3-0.4 .7-0.8 1-1.1Zm-4.4 4.7l-17.9-16c-0.3 .4-0.7 .8-1.1 1.3l18 15.9c.3-0.4 .7-0.8 1-1.2Zm-4.3 5l-18.4-15.4c-0.4 .4-0.7 .9-1.1 1.3l18.6 15.2c.3-0.4 .6-0.8 .9-1.1Zm-4.1 5l-19-14.6c-0.3 .4-0.7 .9-1 1.4l19.1 14.4c.3-0.4 .6-0.8 .9-1.2Zm-3.8 5.3l-19.6-13.9c-0.3 .4-0.6 .9-1 1.4l19.7 13.7c.3-0.4 .6-0.8 .9-1.2Zm-3.7 5.4l-20.1-13.2c-0.3 .5-0.6 1-0.9 1.4l20.2 13c.3-0.4 .5-0.8 .8-1.2Zm-3.5 5.5l-20.5-12.4c-0.3 .5-0.6 1-0.9 1.5l20.7 12.2c.2-0.5 .5-0.9 .7-1.3Zm-3.2 5.6l-21-11.6c-0.3 .5-0.6 1-0.9 1.5l21.2 11.4c.2-0.4 .4-0.8 .7-1.3Zm-3 5.8l-21.5-10.8c-0.3 .5-0.5 1-0.8 1.6l21.6 10.5c.2-0.4 .4-0.9 .7-1.3Zm-2.9 5.9l-21.8-10c-0.2 .5-0.5 1.1-0.7 1.6l21.9 9.7c.2-0.4 .4-0.9 .6-1.3Zm-2.5 5.9l-22.3-9c-0.2 .5-0.4 1-0.6 1.5l22.3 8.9c.2-0.4 .4-0.9 .6-1.4Zm-2.4 6.1l-22.5-8.2c-0.2 .5-0.4 1.1-0.6 1.6l22.6 8c.2-0.4 .3-0.9 .5-1.4Zm-2.1 6.2l-22.8-7.4c-0.2 .6-0.4 1.1-0.6 1.7l23 7.1c.1-0.5 .3-0.9 .4-1.4Zm-1.9 6.2l-23.1-6.4c-0.1 .5-0.3 1.1-0.4 1.6l23.1 6.3c.2-0.5 .3-1 .4-1.5Zm-1.6 6.3l-23.3-5.5c-0.2 .5-0.3 1.1-0.4 1.6l23.4 5.4c.1-0.5 .2-1 .3-1.5Zm-1.4 6.4l-23.5-4.7c-0.1 .6-0.2 1.2-0.3 1.7l23.5 4.5c.1-0.5 .2-1 .3-1.5Zm-1.1 6.4l-23.7-3.7c-0.1 .5-0.2 1.1-0.3 1.7l23.8 3.5c0-0.5 .1-1 .2-1.5Zm-0.9 6.4l-23.8-2.8c-0.1 .6-0.2 1.2-0.2 1.7l23.8 2.6c.1-0.5 .1-1 .2-1.5Zm-0.6 6.5l-24-1.9c0 .6-0.1 1.1-0.1 1.7l23.9 1.7c.1-0.5 .1-1 .2-1.5Zm-0.4 6.5l-24-1c0 .6-0.1 1.2-0.1 1.7l24 .8c0-0.5 0-1 .1-1.5Zm-0.2 6.5c0 .3 0 .5 0 .8c0 .3 0 .5 0 .8l-24 .1c0-0.3 0-0.6 0-0.9c0-0.3 0-0.6 0-0.9l24 .1Zm.1 6.6l-24 .8c0 .5 .1 1.1 .1 1.7l24-1c-0.1-0.5-0.1-1-0.1-1.5Zm.3 6.5l-23.9 1.7c0 .6 .1 1.1 .1 1.7l24-1.9c-0.1-0.5-0.1-1-0.2-1.5Zm.6 6.5l-23.8 2.6c0 .5 .1 1.1 .2 1.7l23.8-2.8c-0.1-0.5-0.1-1-0.2-1.5Zm.9 6.4l-23.8 3.5c.1 .6 .2 1.2 .3 1.7l23.7-3.7c-0.1-0.5-0.2-1-0.2-1.5Zm1 6.4l-23.5 4.5c.1 .5 .2 1.1 .3 1.7l23.5-4.7c-0.1-0.5-0.2-1-0.3-1.5Zm1.4 6.4l-23.4 5.4c.1 .5 .2 1.1 .4 1.6l23.3-5.5c-0.1-0.5-0.2-1-0.3-1.5Zm1.5 6.3l-23.1 6.3c.1 .5 .3 1.1 .4 1.6l23.1-6.4c-0.1-0.5-0.2-1-0.4-1.5Zm1.9 6.3l-23 7.1c.2 .6 .4 1.1 .6 1.7l22.8-7.4c-0.1-0.5-0.3-0.9-0.4-1.4Zm2 6.2l-22.6 8c.2 .5 .4 1.1 .6 1.6l22.5-8.2c-0.2-0.5-0.3-1-0.5-1.4Zm2.3 6.1l-22.3 8.9c.2 .5 .4 1 .6 1.5l22.3-9c-0.2-0.5-0.4-1-0.6-1.4Zm2.5 6l-21.9 9.7c.2 .5 .5 1.1 .7 1.6l21.8-10c-0.2-0.4-0.4-0.9-0.6-1.3Zm2.8 5.9l-21.6 10.5c.3 .6 .5 1.1 .8 1.6l21.5-10.8c-0.3-0.4-0.5-0.9-0.7-1.3Zm3 5.8l-21.2 11.4c.3 .5 .6 1 .9 1.5l21-11.6c-0.3-0.5-0.5-0.9-0.7-1.3Zm3.2 5.6l-20.7 12.2c.3 .5 .6 1 .9 1.5l20.5-12.4c-0.2-0.4-0.5-0.8-0.7-1.3Zm3.4 5.6l-20.2 13c.3 .4 .6 .9 .9 1.4l20.1-13.2c-0.3-0.4-0.5-0.8-0.8-1.2Zm3.6 5.4l-19.7 13.7c.4 .5 .7 1 1 1.4l19.6-13.9c-0.3-0.4-0.6-0.8-0.9-1.2Zm3.8 5.3l-19.1 14.4c.3 .5 .7 1 1 1.4l19-14.6c-0.3-0.4-0.6-0.8-0.9-1.2Zm4.1 5.1l-18.6 15.2c.4 .4 .7 .9 1.1 1.3l18.4-15.4c-0.3-0.3-0.6-0.7-0.9-1.1Zm4.2 4.9l-18 15.9c.4 .5 .8 .9 1.1 1.3l17.9-16c-0.3-0.4-0.7-0.8-1-1.2Zm4.4 4.8l-17.4 16.6c.4 .4 .8 .8 1.2 1.2l17.2-16.7c-0.3-0.3-0.7-0.7-1-1.1Zm4.6 4.7l-16.7 17.2c.4 .4 .8 .8 1.2 1.2l16.6-17.4c-0.4-0.3-0.8-0.7-1.1-1Zm4.7 4.4l-16 17.9c.4 .3 .8 .7 1.3 1.1l15.9-18c-0.4-0.3-0.8-0.7-1.2-1Zm5 4.3l-15.4 18.4c.4 .4 .9 .7 1.3 1.1l15.2-18.6c-0.4-0.3-0.8-0.6-1.1-0.9Zm5 4.1l-14.6 19c.4 .3 .9 .7 1.4 1l14.4-19.1c-0.4-0.3-0.8-0.6-1.2-0.9Zm5.3 3.8l-13.9 19.6c.4 .3 .9 .6 1.4 1l13.7-19.7c-0.4-0.3-0.8-0.6-1.2-0.9Zm5.4 3.7l-13.2 20.1c.5 .3 1 .6 1.4 .9l13-20.2c-0.4-0.3-0.8-0.5-1.2-0.8Zm5.5 3.5l-12.4 20.5c.5 .3 1 .6 1.5 .9l12.2-20.7c-0.5-0.2-0.9-0.5-1.3-0.7Zm5.6 3.2l-11.6 21c.5 .3 1 .6 1.5 .9l11.4-21.2c-0.4-0.2-0.8-0.4-1.3-0.7Zm5.8 3l-10.8 21.5c.5 .3 1 .5 1.6 .8l10.5-21.6c-0.4-0.2-0.9-0.4-1.3-0.7Zm5.9 2.9l-10 21.8c.5 .2 1.1 .5 1.6 .7l9.7-21.9c-0.4-0.2-0.9-0.4-1.3-0.6Zm5.9 2.5l-9 22.2c.5 .3 1 .5 1.5 .7l8.9-22.3c-0.4-0.2-0.9-0.4-1.4-0.6Zm6.1 2.4l-8.2 22.5c.5 .2 1.1 .4 1.6 .6l8-22.6c-0.4-0.2-0.9-0.3-1.4-0.5Zm6.2 2.1l-7.4 22.8c.6 .2 1.1 .4 1.7 .6l7.1-23c-0.5-0.1-0.9-0.3-1.4-0.4Zm6.2 1.9l-6.4 23.1c.5 .1 1.1 .3 1.6 .4l6.3-23.1c-0.5-0.2-1-0.3-1.5-0.4Zm6.3 1.6l-5.5 23.3c.5 .2 1.1 .3 1.6 .4l5.4-23.4c-0.5-0.1-1-0.2-1.5-0.3Zm6.4 1.4l-4.7 23.5c.6 .1 1.2 .2 1.7 .3l4.5-23.5c-0.5-0.1-1-0.2-1.5-0.3Zm6.4 1.1l-3.7 23.7c.5 .1 1.1 .2 1.7 .3l3.5-23.8c-0.5 0-1-0.1-1.5-0.2Zm6.4 .9l-2.8 23.8c.6 .1 1.2 .2 1.7 .2l2.6-23.8c-0.5-0.1-1-0.1-1.5-0.2Zm6.5 .6l-1.9 24c.6 0 1.1 .1 1.7 .1l1.7-23.9c-0.5-0.1-1-0.1-1.5-0.2Zm6.5 .4l-1 24c.6 .1 1.2 .1 1.7 .1l.8-24c-0.5 0-1 0-1.5-0.1Zm6.5 .2l-0.1 24c.3 0 .6 0 .9 0c.3 0 .6 0 .9 0l-0.1-24c-0.3 0-0.5 0-0.8 0c-0.3 0-0.5 0-0.8 0Zm6.6-0.1l.8 24c.5 0 1.1 0 1.7-0.1l-1-24c-0.5 .1-1 .1-1.5 .1Zm6.5-0.3l1.7 23.9c.6 0 1.1-0.1 1.7-0.1l-1.9-23.9c-0.5 0-1 0-1.5 .1Zm6.5-0.6l2.6 23.8c.5 0 1.1-0.1 1.7-0.2l-2.8-23.8c-0.5 .1-1 .1-1.5 .2Zm6.4-0.9l3.5 23.8c.6-0.1 1.2-0.2 1.7-0.3l-3.7-23.7c-0.5 .1-1 .2-1.5 .2Zm6.4-1l4.5 23.5c.5-0.1 1.1-0.2 1.7-0.3l-4.7-23.5c-0.5 .1-1 .2-1.5 .3Zm6.4-1.4l5.4 23.4c.5-0.1 1.1-0.2 1.6-0.4l-5.5-23.3c-0.5 .1-1 .2-1.5 .3Zm6.3-1.5l6.3 23.1c.5-0.1 1.1-0.3 1.6-0.4l-6.4-23.1c-0.5 .1-1 .2-1.5 .4Zm6.3-1.9l7.1 23c.3-0.1 .5-0.2 .8-0.3c.3-0.1 .6-0.2 .9-0.3l-7.4-22.8c-0.5 .1-0.9 .3-1.4 .4Zm6.2-2l8 22.6c.5-0.2 1.1-0.4 1.6-0.6l-8.2-22.5c-0.5 .2-1 .3-1.4 .5Zm6.1-2.3l8.9 22.3c.5-0.2 1-0.4 1.5-0.6l-9-22.3c-0.5 .2-1 .4-1.4 .6Zm6-2.5l9.7 21.9c.5-0.2 1.1-0.5 1.6-0.7l-10-21.8c-0.4 .2-0.9 .4-1.3 .6Zm5.9-2.8l10.5 21.6c.6-0.3 1.1-0.5 1.6-0.8l-10.8-21.5c-0.4 .3-0.9 .5-1.3 .7Zm5.8-3l11.4 21.2c.5-0.3 1-0.6 1.5-0.9l-11.6-21c-0.5 .3-0.9 .5-1.3 .7Zm5.6-3.2l12.2 20.7c.5-0.3 1-0.6 1.5-0.9l-12.4-20.5c-0.4 .2-0.8 .5-1.3 .7Zm5.6-3.4l13 20.2c.4-0.3 .9-0.6 1.4-0.9l-13.2-20.1c-0.4 .3-0.8 .5-1.2 .8Zm5.4-3.6l13.7 19.7c.5-0.4 1-0.7 1.4-1l-13.9-19.6c-0.4 .3-0.8 .6-1.2 .9Zm5.3-3.8l14.4 19.1c.5-0.3 1-0.7 1.4-1l-14.6-19c-0.4 .3-0.8 .6-1.2 .9Zm5.1-4.1l15.2 18.6c.4-0.4 .9-0.7 1.3-1.1l-15.4-18.4c-0.3 .3-0.7 .6-1.1 .9Zm4.9-4.2l15.9 18c.5-0.4 .9-0.8 1.3-1.1l-16-17.9c-0.4 .4-0.8 .7-1.2 1Zm4.8-4.4l16.6 17.4c.4-0.4 .8-0.8 1.2-1.2l-16.7-17.2c-0.3 .3-0.7 .7-1.1 1Zm4.7-4.6l17.2 16.7c.4-0.4 .8-0.8 1.2-1.2l-17.4-16.6c-0.3 .4-0.7 .8-1 1.1Zm4.4-4.7l17.9 16c.3-0.4 .7-0.8 1.1-1.3l-18-15.9c-0.3 .4-0.6 .8-1 1.2Zm4.3-5l18.4 15.4c.4-0.4 .7-0.9 1.1-1.3l-18.6-15.2c-0.3 .4-0.6 .8-0.9 1.1Zm4.1-5l19 14.6c.3-0.4 .7-0.9 1-1.4l-19.1-14.4c-0.3 .4-0.6 .8-0.9 1.2Zm3.8-5.3l19.6 13.9c.3-0.4 .6-0.9 1-1.4l-19.7-13.7c-0.3 .4-0.6 .8-0.9 1.2Zm3.7-5.4l20.1 13.2c.3-0.5 .6-1 .9-1.4l-20.2-13c-0.3 .4-0.5 .8-0.8 1.2Zm3.5-5.5l20.5 12.4c.3-0.5 .6-1 .9-1.5l-20.7-12.2c-0.2 .5-0.5 .9-0.7 1.3Zm3.2-5.6l21 11.6c.3-0.5 .6-1 .9-1.5l-21.2-11.4c-0.2 .4-0.4 .8-0.7 1.3Zm3-5.8l21.5 10.8c.3-0.5 .5-1 .8-1.6l-21.6-10.5c-0.2 .4-0.4 .9-0.7 1.3Zm2.9-5.9l21.8 10c.2-0.5 .5-1.1 .7-1.6l-21.9-9.7c-0.2 .4-0.4 .9-0.6 1.3Zm2.5-5.9l22.3 9c.2-0.5 .4-1 .6-1.5l-22.3-8.9c-0.2 .4-0.4 .9-0.6 1.4Zm2.4-6.1l22.5 8.2c.2-0.5 .4-1.1 .6-1.6l-22.6-8c-0.2 .4-0.3 .9-0.5 1.4Zm2.1-6.2l22.8 7.4c.2-0.6 .4-1.1 .6-1.7l-23-7.1c-0.1 .5-0.3 .9-0.4 1.4Zm1.9-6.2l23.1 6.4c.1-0.5 .3-1.1 .4-1.6l-23.1-6.3c-0.2 .5-0.3 1-0.4 1.5Zm1.6-6.3l23.3 5.5c.2-0.5 .3-1.1 .4-1.6l-23.4-5.4c-0.1 .5-0.2 1-0.3 1.5Zm1.4-6.4l23.5 4.7c.1-0.6 .2-1.2 .3-1.7l-23.5-4.5c-0.1 .5-0.2 1-0.3 1.5Zm1.1-6.4l23.7 3.7c.1-0.5 .2-1.1 .3-1.7l-23.8-3.5c0 .5-0.1 1-0.2 1.5Zm.9-6.4l23.8 2.8c.1-0.6 .2-1.2 .2-1.7l-23.8-2.6c-0.1 .5-0.1 1-0.2 1.5Zm.7-6.5l23.9 1.9c0-0.6 .1-1.1 .1-1.7l-23.9-1.7c-0.1 .5-0.1 1-0.1 1.5Zm.3-6.5l24 1c.1-0.6 .1-1.2 .1-1.7l-24-0.8c0 .5 0 1-0.1 1.5Z"
            fill="url(#Gradient-14)"
          />
        </g>
        <g transform="translate(55.5,55)">
          <g filter="url(#filter0_d_4667_1690)" transform="translate(-55.5,-55)">
            <path
              fill-rule="evenodd"
              clip-rule="evenodd"
              d="M362 210.5c0-0.2 0-0.3 0-0.5c0-0.2 0-0.3 0-0.5h2.5c0 .1 0 .2 0 .3v.2c0 .2 0 .3 0 .5h-2.5Zm-0.1-7l2.5-0.1c-0.1-0.4-0.1-0.7-0.1-1l-2.5 .1c0 .3 0 .6 .1 1Zm-0.5-7l2.5-0.3c0-0.3-0.1-0.6-0.1-1l-2.5 .3c0 .3 .1 .6 .1 1Zm-0.8-7l2.5-0.3c0-0.4-0.1-0.7-0.1-1.1l-2.5 .4c0 .3 .1 .7 .1 1Zm-1.1-7l2.5-0.4c-0.1-0.3-0.1-0.7-0.2-1l-2.5 .5c.1 .3 .2 .6 .2 .9Zm-1.4-6.8l2.4-0.6c0-0.2 0-0.3-0.1-0.5c0-0.1 0-0.3-0.1-0.5l-2.4 .6c.1 .3 .1 .6 .2 1Zm-1.7-6.8l2.4-0.7c-0.1-0.3-0.2-0.7-0.3-1l-2.4 .7c.1 .3 .2 .6 .3 1Zm-2.1-6.7l2.4-0.8c-0.1-0.4-0.2-0.7-0.3-1l-2.4 .8c.1 .3 .2 .6 .3 1Zm-2.3-6.7l2.3-0.9c-0.1-0.3-0.3-0.6-0.4-0.9l-2.3 .9c.1 .3 .2 .6 .4 .9Zm-2.7-6.5l2.3-1c-0.2-0.3-0.3-0.6-0.4-0.9l-2.3 1c.1 .3 .3 .6 .4 .9Zm-3-6.3l2.3-1.1c-0.2-0.3-0.3-0.6-0.5-0.9l-2.2 1.1c.1 .3 .3 .6 .4 .9Zm-3.2-6.2l2.1-1.2c0-0.2-0.1-0.3-0.2-0.4c-0.1-0.2-0.2-0.4-0.3-0.5l-2.1 1.2c.1 .3 .3 .6 .5 .9Zm-3.6-6.1l2.1-1.3c-0.1-0.3-0.3-0.6-0.5-0.9l-2.1 1.3c.2 .3 .3 .6 .5 .9Zm-3.8-5.9l2.1-1.4c-0.2-0.3-0.4-0.6-0.6-0.8l-2.1 1.4c.2 .3 .4 .5 .6 .8Zm-4.1-5.7l2-1.5c-0.2-0.3-0.4-0.5-0.6-0.8l-2 1.5c.2 .3 .4 .5 .6 .8Zm-4.3-5.5l1.9-1.6c-0.2-0.3-0.4-0.5-0.7-0.8l-1.9 1.6c.2 .3 .5 .5 .7 .8Zm-4.6-5.3l1.8-1.7c-0.2-0.3-0.4-0.5-0.7-0.8l-1.8 1.7c.2 .3 .4 .5 .7 .8Zm-4.9-5.1l1.8-1.8c-0.2-0.2-0.5-0.5-0.7-0.7l-1.8 1.8c.3 .2 .5 .4 .7 .7Zm-5-4.9l1.7-1.8c-0.3-0.3-0.5-0.5-0.8-0.7l-1.7 1.8c.3 .3 .5 .5 .8 .7Zm-5.3-4.6l1.6-1.9c-0.3-0.3-0.5-0.5-0.8-0.7l-1.6 1.9c.3 .3 .5 .5 .8 .7Zm-5.5-4.4l1.5-2c-0.3-0.2-0.5-0.4-0.8-0.6l-1.5 2c.3 .2 .5 .4 .8 .6Zm-5.7-4.1l1.4-2.1c-0.2-0.2-0.5-0.4-0.8-0.6l-1.4 2.1c.3 .2 .5 .4 .8 .6Zm-5.8-3.9l1.3-2.1c-0.3-0.2-0.6-0.4-0.9-0.5l-1.3 2.1c.3 .2 .6 .3 .9 .5Zm-6.1-3.6l1.2-2.1c-0.3-0.2-0.6-0.4-0.9-0.5l-1.2 2.1c.3 .2 .6 .4 .9 .5Zm-6.2-3.3l1.1-2.2c-0.3-0.2-0.6-0.3-0.9-0.5l-1.1 2.3c.3 .1 .6 .3 .9 .4Zm-6.3-3l1-2.3c-0.3-0.1-0.6-0.2-0.9-0.4l-1 2.3c.3 .1 .6 .3 .9 .4Zm-6.5-2.7l.9-2.3c-0.3-0.1-0.6-0.3-0.9-0.4l-0.9 2.3c.3 .2 .6 .3 .9 .4Zm-6.6-2.4l.8-2.4c-0.3-0.1-0.6-0.2-1-0.3l-0.8 2.4c.4 .1 .7 .2 1 .3Zm-6.7-2.1l.7-2.4c-0.3-0.1-0.7-0.2-1-0.3l-0.7 2.4c.4 .1 .7 .2 1 .3Zm-6.8-1.8l.6-2.4c-0.3-0.1-0.7-0.2-1-0.2l-0.6 2.4c.4 .1 .7 .1 1 .2Zm-6.9-1.4l.5-2.5c-0.3-0.1-0.7-0.1-1-0.2l-0.4 2.5c.3 0 .6 .1 .9 .2Zm-6.9-1.2l.4-2.5c-0.4 0-0.7-0.1-1.1-0.1l-0.3 2.5c.3 0 .7 .1 1 .1Zm-7-0.8l.3-2.5c-0.4 0-0.7-0.1-1-0.1l-0.3 2.5c.4 0 .7 .1 1 .1Zm-7-0.5l.1-2.5c-0.3 0-0.6 0-1-0.1l-0.1 2.5c.4 .1 .7 .1 1 .1Zm-7-0.2v-2.5c-0.2 0-0.3 0-0.5 0c-0.2 0-0.3 0-0.5 0v2.5c.2 0 .3 0 .5 0c.2 0 .3 0 .5 0Zm-7 .1l-0.1-2.5c-0.4 .1-0.7 .1-1 .1l.1 2.5c.3 0 .6 0 1-0.1Zm-7 .5l-0.3-2.5c-0.3 0-0.6 .1-1 .1l.3 2.5c.3 0 .6-0.1 1-0.1Zm-7 .8l-0.3-2.5c-0.4 0-0.7 .1-1.1 .1l.4 2.5c.3 0 .7-0.1 1-0.1Zm-7 1.1l-0.4-2.5c-0.3 .1-0.7 .1-1 .2l.5 2.5c.3-0.1 .6-0.2 .9-0.2Zm-6.8 1.4l-0.6-2.4c-0.3 0-0.7 .1-1 .2l.6 2.4c.3-0.1 .6-0.1 1-0.2Zm-6.8 1.7l-0.7-2.4c-0.3 .1-0.7 .2-1 .3l.7 2.4c.3-0.1 .6-0.2 1-0.3Zm-6.7 2.1l-0.8-2.4c-0.4 .1-0.7 .2-1 .3l.8 2.4c.3-0.1 .6-0.2 1-0.3Zm-6.7 2.3l-0.9-2.3c-0.3 .1-0.6 .3-0.9 .4l.9 2.3c.3-0.1 .6-0.2 .9-0.4Zm-6.5 2.7l-1-2.3c-0.3 .2-0.6 .3-0.9 .4l1 2.3c.3-0.1 .6-0.3 .9-0.4Zm-6.3 3l-1.1-2.3c-0.3 .2-0.6 .3-0.9 .5l1.1 2.2c.3-0.1 .6-0.3 .9-0.4Zm-6.2 3.2l-1.2-2.1c-0.3 .1-0.6 .3-0.9 .5l1.2 2.1c.3-0.1 .6-0.3 .9-0.5Zm-6.1 3.6l-1.3-2.1c-0.3 .1-0.6 .3-0.9 .5l1.3 2.1c.3-0.2 .6-0.3 .9-0.5Zm-5.9 3.8l-1.4-2.1c-0.3 .2-0.6 .4-0.8 .6l1.4 2.1c.3-0.2 .5-0.4 .8-0.6Zm-5.7 4.1l-1.5-2c-0.3 .2-0.5 .4-0.8 .6l1.5 2c.3-0.2 .5-0.4 .8-0.6Zm-5.5 4.3l-1.6-1.9c-0.3 .2-0.5 .4-0.8 .7l1.6 1.9c.3-0.2 .5-0.4 .8-0.7Zm-5.3 4.6l-1.7-1.8c-0.3 .2-0.5 .4-0.8 .7l1.7 1.8c.3-0.2 .5-0.4 .8-0.7Zm-5.1 4.9l-1.8-1.8c-0.2 .2-0.5 .5-0.7 .7l1.8 1.8c.2-0.3 .4-0.5 .7-0.7Zm-4.9 5l-1.8-1.7c-0.3 .3-0.5 .5-0.7 .8l1.8 1.7c.3-0.3 .5-0.5 .7-0.8Zm-4.6 5.3l-1.9-1.6c-0.3 .3-0.5 .5-0.7 .8l1.9 1.6c.3-0.3 .5-0.5 .7-0.8Zm-4.4 5.5l-2-1.5c-0.2 .3-0.4 .5-0.6 .8l2 1.5c.2-0.3 .4-0.5 .6-0.8Zm-4.1 5.7l-2.1-1.4c-0.2 .2-0.4 .5-0.6 .8l2.1 1.4c.2-0.3 .4-0.5 .6-0.8Zm-3.9 5.8l-2.1-1.3c-0.2 .3-0.4 .6-0.5 .9l2.1 1.3c.2-0.3 .3-0.6 .5-0.9Zm-3.6 6.1l-2.1-1.2c-0.2 .3-0.4 .6-0.5 .9l2.1 1.2c.2-0.3 .4-0.6 .5-0.9Zm-3.3 6.2l-2.2-1.1c-0.2 .3-0.3 .6-0.5 .9l2.3 1.1c.1-0.3 .3-0.6 .4-0.9Zm-3 6.3l-2.3-1c-0.1 .3-0.2 .6-0.4 .9l2.3 1c.1-0.3 .3-0.6 .4-0.9Zm-2.7 6.5l-2.3-0.9c-0.1 .3-0.3 .6-0.4 .9l2.3 .9c.2-0.3 .3-0.6 .4-0.9Zm-2.4 6.6l-2.4-0.8c-0.1 .3-0.2 .6-0.3 1l2.4 .8c.1-0.4 .2-0.7 .3-1Zm-2.1 6.7l-2.4-0.7c-0.1 .3-0.2 .7-0.3 1l2.4 .7c.1-0.4 .2-0.7 .3-1Zm-1.8 6.8l-2.4-0.6c-0.1 .3-0.2 .7-0.2 1l2.4 .6c.1-0.4 .1-0.7 .2-1Zm-1.4 6.9l-2.5-0.5c-0.1 .3-0.1 .7-0.2 1l2.5 .4c0-0.3 .1-0.6 .2-0.9Zm-1.2 6.9l-2.5-0.4c0 .4-0.1 .7-0.1 1.1l2.5 .3c0-0.3 .1-0.7 .1-1Zm-0.8 7l-2.5-0.3c0 .4-0.1 .7-0.1 1l2.5 .3c0-0.4 .1-0.7 .1-1Zm-0.5 7l-2.5-0.1c0 .3 0 .6-0.1 1l2.5 .1c.1-0.4 .1-0.7 .1-1Zm-0.2 7c0 .2 0 .3 0 .5c0 .2 0 .3 0 .5h-2.5c0-0.2 0-0.3 0-0.5c0-0.2 0-0.3 0-0.5h2.5Zm.1 7l-2.5 .1c.1 .4 .1 .7 .1 1l2.5-0.1c0-0.3 0-0.6-0.1-1Zm.5 7l-2.5 .3c0 .3 .1 .6 .1 1l2.5-0.3c0-0.3-0.1-0.6-0.1-1Zm.8 7l-2.5 .3c0 .4 .1 .7 .1 1.1l2.5-0.4c0-0.3-0.1-0.7-0.1-1Zm1.1 7l-2.5 .4c.1 .3 .1 .7 .2 1l2.5-0.5c-0.1-0.3-0.2-0.6-0.2-0.9Zm1.4 6.8l-2.4 .6c0 .3 .1 .7 .2 1l2.4-0.6c-0.1-0.3-0.1-0.6-0.2-1Zm1.7 6.8l-2.4 .7c.1 .3 .2 .7 .3 1l2.4-0.7c-0.1-0.3-0.2-0.6-0.3-1Zm2.1 6.7l-2.4 .8c.1 .4 .2 .7 .3 1l2.4-0.8c-0.1-0.3-0.2-0.6-0.3-1Zm2.3 6.7l-2.3 .9c.1 .3 .3 .6 .4 .9l2.3-0.9c-0.1-0.3-0.2-0.6-0.4-0.9Zm2.7 6.5l-2.3 1c.2 .3 .3 .6 .4 .9l2.3-1c-0.1-0.3-0.3-0.6-0.4-0.9Zm3 6.3l-2.3 1.1c.2 .3 .3 .6 .5 .9l2.2-1.1c-0.1-0.3-0.3-0.6-0.4-0.9Zm3.2 6.2l-2.1 1.2c.1 .3 .3 .6 .5 .9l2.1-1.2c-0.1-0.3-0.3-0.6-0.5-0.9Zm3.6 6.1l-2.1 1.3c.1 .3 .3 .6 .5 .9l2.1-1.3c-0.2-0.3-0.3-0.6-0.5-0.9Zm3.8 5.9l-2.1 1.4c.2 .3 .4 .6 .6 .8l2.1-1.4c-0.2-0.3-0.4-0.5-0.6-0.8Zm4.1 5.7l-2 1.5c.2 .3 .4 .5 .6 .8l2-1.5c-0.2-0.3-0.4-0.5-0.6-0.8Zm4.3 5.5l-1.9 1.6c.2 .3 .4 .5 .7 .8l1.9-1.6c-0.2-0.3-0.4-0.5-0.7-0.8Zm4.6 5.3l-1.8 1.7c.2 .3 .4 .5 .7 .8l1.8-1.7c-0.2-0.3-0.4-0.5-0.7-0.8Zm4.9 5.1l-1.8 1.8c.2 .2 .5 .5 .7 .7l1.8-1.8c-0.3-0.2-0.5-0.4-0.7-0.7Zm5 4.9l-1.7 1.8c.3 .3 .5 .5 .8 .7l1.7-1.8c-0.3-0.3-0.5-0.5-0.8-0.7Zm5.3 4.6l-1.6 1.9c.3 .3 .5 .5 .8 .7l1.6-1.9c-0.3-0.2-0.5-0.5-0.8-0.7Zm5.5 4.4l-1.5 2c.3 .2 .5 .4 .8 .6l1.5-2c-0.3-0.2-0.5-0.4-0.8-0.6Zm5.7 4.1l-1.4 2.1c.2 .2 .5 .4 .8 .6l1.4-2.1c-0.3-0.2-0.5-0.4-0.8-0.6Zm5.8 3.9l-1.3 2.1c.3 .2 .6 .4 .9 .5l1.3-2.1c-0.3-0.2-0.6-0.3-0.9-0.5Zm6.1 3.6l-1.2 2.1c.3 .2 .6 .4 .9 .5l1.2-2.1c-0.3-0.2-0.6-0.4-0.9-0.5Zm6.2 3.3l-1.1 2.2c.3 .2 .6 .3 .9 .5l1.1-2.3c-0.3-0.1-0.6-0.3-0.9-0.4Zm6.3 3l-1 2.3c.3 .1 .6 .2 .9 .4l1-2.3c-0.3-0.1-0.6-0.3-0.9-0.4Zm6.5 2.7l-0.9 2.3c.3 .1 .6 .3 .9 .4l.9-2.3c-0.3-0.2-0.6-0.3-0.9-0.4Zm6.6 2.4l-0.8 2.4c.3 .1 .6 .2 1 .3l.8-2.4c-0.4-0.1-0.7-0.2-1-0.3Zm6.7 2.1l-0.7 2.4c.3 .1 .7 .2 1 .3l.7-2.4c-0.4-0.1-0.7-0.2-1-0.3Zm6.8 1.8l-0.6 2.4c.3 .1 .7 .2 1 .2l.6-2.4c-0.4-0.1-0.7-0.1-1-0.2Zm6.9 1.4l-0.5 2.5c.3 .1 .7 .1 1 .2l.4-2.5c-0.3 0-0.6-0.1-0.9-0.2Zm6.9 1.2l-0.4 2.5c.4 0 .7 .1 1.1 .1l.3-2.5c-0.3 0-0.7-0.1-1-0.1Zm7 .8l-0.3 2.5c.4 0 .7 .1 1 .1l.3-2.5c-0.4 0-0.7-0.1-1-0.1Zm7 .5l-0.1 2.5c.3 0 .6 0 1 .1l.1-2.5c-0.4-0.1-0.7-0.1-1-0.1Zm7 .2v2.5c.2 0 .3 0 .5 0h.2c.1 0 .2 0 .3 0v-2.5c-0.2 0-0.3 0-0.5 0c-0.2 0-0.3 0-0.5 0Zm7-0.1l.1 2.5c.4-0.1 .7-0.1 1-0.1l-0.1-2.5c-0.3 0-0.6 0-1 .1Zm7-0.5l.3 2.5c.3 0 .6-0.1 1-0.1l-0.3-2.5c-0.3 0-0.6 .1-1 .1Zm7-0.8l.3 2.5c.4 0 .7-0.1 1.1-0.1l-0.4-2.5c-0.3 0-0.7 .1-1 .1Zm7-1.1l.4 2.5c.3-0.1 .7-0.1 1-0.2l-0.5-2.5c-0.3 .1-0.6 .2-0.9 .2Zm6.8-1.4l.6 2.4c.3 0 .7-0.1 1-0.2l-0.6-2.4c-0.3 .1-0.6 .1-1 .2Zm6.8-1.7l.7 2.4c.3-0.1 .7-0.2 1-0.3l-0.7-2.4c-0.3 .1-0.6 .2-1 .3Zm6.7-2.1l.8 2.4c.2-0.1 .3-0.1 .5-0.2c.1 0 .3-0.1 .5-0.1l-0.8-2.4c-0.3 .1-0.6 .2-1 .3Zm6.7-2.3l.9 2.3c.3-0.1 .6-0.3 .9-0.4l-0.9-2.3c-0.3 .1-0.6 .2-0.9 .4Zm6.5-2.7l1 2.3c.3-0.2 .6-0.3 .9-0.4l-1-2.3c-0.3 .1-0.6 .3-0.9 .4Zm6.3-3l1.1 2.3c.3-0.2 .6-0.3 .9-0.5l-1.1-2.2c-0.3 .1-0.6 .3-0.9 .4Zm6.2-3.2l1.2 2.1c.3-0.1 .6-0.3 .9-0.5l-1.2-2.1c-0.3 .1-0.6 .3-0.9 .5Zm6.1-3.6l1.3 2.1c.2-0.1 .3-0.1 .5-0.2c.1-0.1 .2-0.2 .4-0.3l-1.3-2.1c-0.3 .2-0.6 .3-0.9 .5Zm5.9-3.8l1.4 2.1c.1-0.1 .3-0.2 .4-0.3c.2-0.1 .3-0.2 .4-0.3l-1.4-2.1c-0.3 .2-0.5 .4-0.8 .6Zm5.7-4.1l1.5 2c.3-0.2 .5-0.4 .8-0.6l-1.5-2c-0.3 .2-0.5 .4-0.8 .6Zm5.5-4.3l1.6 1.9c.3-0.2 .5-0.4 .8-0.7l-1.6-1.9c-0.3 .2-0.5 .5-0.8 .7Zm5.3-4.6l1.7 1.8c.3-0.2 .5-0.4 .8-0.7l-1.7-1.8c-0.3 .2-0.5 .4-0.8 .7Zm5.1-4.9l1.8 1.8c.2-0.2 .5-0.5 .7-0.7l-1.8-1.8c-0.2 .3-0.4 .5-0.7 .7Zm4.9-5l1.8 1.7c.3-0.3 .5-0.5 .7-0.8l-1.8-1.7c-0.3 .3-0.5 .5-0.7 .8Zm4.6-5.3l1.9 1.6c.3-0.3 .5-0.5 .7-0.8l-1.9-1.6c-0.2 .3-0.5 .5-0.7 .8Zm4.4-5.5l2 1.5c.2-0.3 .4-0.5 .6-0.8l-2-1.5c-0.2 .3-0.4 .5-0.6 .8Zm4.1-5.7l2.1 1.4c.2-0.2 .4-0.5 .6-0.8l-2.1-1.4c-0.2 .3-0.4 .5-0.6 .8Zm3.9-5.8l2.1 1.3c.2-0.3 .4-0.6 .5-0.9l-2.1-1.3c-0.2 .3-0.3 .6-0.5 .9Zm3.6-6.1l2.1 1.2c.2-0.3 .4-0.6 .5-0.9l-2.1-1.2c-0.2 .3-0.4 .6-0.5 .9Zm3.3-6.2l2.2 1.1c.2-0.3 .3-0.6 .5-0.9l-2.3-1.1c-0.1 .3-0.3 .6-0.4 .9Zm3-6.3l2.3 1c.1-0.3 .2-0.6 .4-0.9l-2.3-1c-0.1 .3-0.3 .6-0.4 .9Zm2.7-6.5l2.3 .9c.1-0.1 .1-0.2 .1-0.3c.1-0.2 .2-0.4 .3-0.6l-2.3-0.9c-0.2 .3-0.3 .6-0.4 .9Zm2.4-6.6l2.4 .8c.1-0.3 .2-0.6 .3-1l-2.4-0.8c-0.1 .4-0.2 .7-0.3 1Zm2.1-6.7l2.4 .7c.1-0.2 .1-0.4 .2-0.6c0-0.1 0-0.3 .1-0.4l-2.4-0.7c-0.1 .4-0.2 .7-0.3 1Zm1.8-6.8l2.4 .6c.1-0.3 .2-0.7 .2-1l-2.4-0.6c-0.1 .4-0.1 .7-0.2 1Zm1.4-6.9l2.5 .5c0-0.1 .1-0.3 .1-0.4c0-0.2 .1-0.4 .1-0.6l-2.5-0.4c0 .3-0.1 .6-0.2 .9Zm1.2-6.9l2.5 .4c0-0.4 .1-0.7 .1-1.1l-2.5-0.3c0 .3-0.1 .7-0.1 1Zm.8-7l2.5 .3c0-0.4 .1-0.7 .1-1l-2.5-0.3c0 .4-0.1 .7-0.1 1Zm.5-7l2.5 .1c0-0.3 0-0.6 .1-1l-2.5-0.1c-0.1 .4-0.1 .7-0.1 1Z"
              fill="#7002fc"
              fill-opacity=".25"
              shape-rendering="crispEdges"
            />
          </g>
          <g mask="url(#Mask-4)" transform="translate(-55.5,-55)">
            <path
              d="M155.5 309c-20.3 0-40.4-4-59.1-11.8c-18.8-7.7-35.8-19.1-50.1-33.5c-14.4-14.3-25.8-31.3-33.5-50.1c-7.8-18.7-11.8-38.8-11.8-59.1h154.5v154.5Z"
              fill="#7002fc"
              transform="translate(210.5,209.5) translate(-155.5,-154.5)"
              style={{ animation: ".1s linear 1.2s both a11_t" }}
            />
            <path
              d="M155.5 309c-20.3 0-40.4-4-59.1-11.8c-18.8-7.7-35.8-19.1-50.1-33.5c-14.4-14.3-25.8-31.3-33.5-50.1c-7.8-18.7-11.8-38.8-11.8-59.1h154.5v154.5Z"
              fill="#7002fc"
              transform="translate(210.5,209.5) translate(-155.5,-154.5)"
              style={{ animation: ".2s linear 1.2s both a12_t" }}
            />
            <path
              d="M155.5 309c-20.3 0-40.4-4-59.1-11.8c-18.8-7.7-35.8-19.1-50.1-33.5c-14.4-14.3-25.8-31.3-33.5-50.1c-7.8-18.7-11.8-38.8-11.8-59.1h154.5v154.5Z"
              fill="#7002fc"
              transform="translate(210.5,209.5) translate(-155.5,-154.5)"
              style={{ animation: ".5s linear 1.2s both a13_t" }}
            />
            <path
              d="M155.5 309c-20.3 0-40.4-4-59.1-11.8c-18.8-7.7-35.8-19.1-50.1-33.5c-14.4-14.3-25.8-31.3-33.5-50.1c-7.8-18.7-11.8-38.8-11.8-59.1h154.5v154.5Z"
              fill="#7002fc"
              transform="translate(210.5,209.5) translate(-155.5,-154.5)"
              style={{ animation: "1s linear 1.2s both a14_t" }}
            />
          </g>
        </g>
        <g transform="translate(65,65)">
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M355 210c0 80.1-64.9 145-145 145c-80.1 0-145-64.9-145-145c0-80.1 64.9-145 145-145c80.1 0 145 64.9 145 145Zm-145 143c79 0 143-64 143-143c0-79-64-143-143-143c-79 0-143 64-143 143c0 79 64 143 143 143Z"
            fill="#0702fc"
            opacity=".2"
            transform="translate(-65,-65)"
            style={{ animation: "2s linear 1.4s both a15_o" }}
          />
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M355 210c0 80.1-64.9 145-145 145c-80.1 0-145-64.9-145-145c0-80.1 64.9-145 145-145c80.1 0 145 64.9 145 145Zm-145 143c79 0 143-64 143-143c0-79-64-143-143-143c-79 0-143 64-143 143c0 79 64 143 143 143Z"
            fill="#ff4200"
            opacity="0"
            transform="translate(-65,-65)"
            style={{ animation: "3.6s linear 3.4s both a16_o" }}
          />
          <g opacity="0" style={{ animation: ".27s linear 6s both a17_o" }}>
            <path
              fill-rule="evenodd"
              clip-rule="evenodd"
              d="M355 210c0 80.1-64.9 145-145 145c-80.1 0-145-64.9-145-145c0-80.1 64.9-145 145-145c80.1 0 145 64.9 145 145Zm-145 143c79 0 143-64 143-143c0-79-64-143-143-143c-79 0-143 64-143 143c0 79 64 143 143 143Z"
              fill="#ff4200"
              transform="translate(-65,-65)"
              style={{ animation: "9.9s linear 12.1s infinite both a18_o" }}
            />
          </g>
        </g>
        <g transform="translate(75,75)">
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M230 135.8c0-0.3 0-0.5 0-0.8c0-0.3 0-0.5 0-0.8l40-0.3c0 .4 0 .7 0 1.1c0 .4 0 .7 0 1.1l-40-0.3Zm-0.2-6.5l40-2.4c-0.1-0.7-0.1-1.4-0.2-2.1l-39.9 3c.1 .5 .1 1 .1 1.5Zm-0.6-6.5l39.7-5.1c-0.1-0.7-0.2-1.4-0.3-2.1l-39.6 5.7c.1 .5 .2 1 .2 1.5Zm-1-6.4l39.2-7.8c-0.1-0.7-0.3-1.4-0.4-2.1l-39.1 8.4c.1 .5 .2 1 .3 1.5Zm-1.5-6.3l38.6-10.5c-0.2-0.7-0.4-1.4-0.6-2l-38.4 11c.1 .5 .3 1 .4 1.5Zm-1.9-6.2l37.8-13.1c-0.3-0.7-0.5-1.4-0.7-2l-37.6 13.7c.2 .4 .3 .9 .5 1.4Zm-2.3-6.1l36.8-15.6c-0.3-0.7-0.6-1.3-0.9-2l-36.5 16.3c.2 .4 .4 .9 .6 1.3Zm-2.8-5.9l35.7-18.1c-0.4-0.6-0.7-1.3-1-1.9l-35.4 18.7c.3 .5 .5 .9 .7 1.3Zm-3.1-5.6l34.3-20.6c-0.4-0.6-0.7-1.2-1.1-1.8l-34 21.1c.3 .4 .5 .8 .8 1.3Zm-3.6-5.5l32.9-22.8c-0.4-0.6-0.8-1.2-1.2-1.7l-32.5 23.3c.3 .4 .6 .8 .8 1.2Zm-3.8-5.2l31.2-25c-0.5-0.5-0.9-1.1-1.4-1.6l-30.8 25.5c.3 .4 .7 .7 1 1.1Zm-4.3-4.9l29.5-27.1c-0.5-0.5-1-1-1.5-1.5l-29 27.5c.4 .4 .7 .7 1 1.1Zm-4.5-4.6l27.5-29c-0.5-0.5-1-1-1.5-1.5l-27.1 29.5c.4 .3 .7 .6 1.1 1Zm-4.9-4.3l25.5-30.8c-0.5-0.5-1.1-0.9-1.6-1.4l-25 31.2c.4 .3 .7 .7 1.1 1Zm-5.1-4l23.3-32.5c-0.5-0.4-1.1-0.8-1.7-1.2l-22.8 32.9c.4 .2 .8 .5 1.2 .8Zm-5.4-3.6l21.1-34c-0.6-0.4-1.2-0.7-1.8-1.1l-20.6 34.3c.5 .3 .9 .5 1.3 .8Zm-5.6-3.2l18.7-35.4c-0.6-0.3-1.3-0.6-1.9-1l-18.1 35.7c.4 .2 .8 .4 1.3 .7Zm-5.9-2.9l16.3-36.5c-0.7-0.3-1.3-0.6-2-0.9l-15.6 36.8c.4 .2 .9 .4 1.3 .6Zm-6-2.4l13.7-37.6c-0.6-0.2-1.3-0.4-2-0.7l-13.1 37.8c.5 .2 1 .3 1.4 .5Zm-6.1-2l11-38.4c-0.6-0.2-1.3-0.4-2-0.6l-10.5 38.6c.5 .1 1 .3 1.5 .4Zm-6.3-1.6l8.4-39.1c-0.7-0.1-1.4-0.3-2.1-0.4l-7.8 39.2c.5 .1 1 .2 1.5 .3Zm-6.4-1.1l5.7-39.6c-0.7-0.1-1.4-0.2-2.1-0.3l-5.1 39.7c.5 0 1 .1 1.5 .2Zm-6.5-0.7l3-39.9c-0.7-0.1-1.4-0.1-2.1-0.2l-2.4 40c.5 0 1 0 1.5 .1Zm-6.4-0.3l.3-40c-0.4 0-0.7 0-1.1 0c-0.4 0-0.7 0-1.1 0l.3 40c.3 0 .5 0 .8 0c.3 0 .5 0 .8 0Zm-6.5 .2l-2.4-40c-0.7 .1-1.4 .1-2.1 .2l3 39.9c.5-0.1 1-0.1 1.5-0.1Zm-6.5 .6l-5.1-39.7c-0.7 .1-1.4 .2-2.1 .3l5.7 39.6c.5-0.1 1-0.2 1.5-0.2Zm-6.4 1l-7.8-39.2c-0.7 .1-1.4 .3-2.1 .4l8.4 39.1c.5-0.1 1-0.2 1.5-0.3Zm-6.3 1.5l-10.5-38.6c-0.7 .2-1.4 .4-2 .6l11 38.4c.5-0.1 1-0.3 1.5-0.4Zm-6.2 1.9l-13.1-37.8c-0.7 .3-1.4 .5-2 .7l13.7 37.6c.4-0.2 .9-0.3 1.4-0.5Zm-6.1 2.3l-15.6-36.8c-0.7 .3-1.3 .6-2 .9l16.3 36.5c.4-0.2 .9-0.4 1.3-0.6Zm-5.8 2.8l-18.2-35.7c-0.6 .4-1.3 .7-1.9 1l18.7 35.4c.5-0.3 .9-0.5 1.4-0.7Zm-5.7 3.1l-20.6-34.3c-0.6 .4-1.2 .7-1.8 1.1l21.1 34c.4-0.3 .8-0.5 1.3-0.8Zm-5.5 3.6l-22.8-32.9c-0.6 .4-1.2 .8-1.7 1.2l23.3 32.5c.4-0.3 .8-0.6 1.2-0.8Zm-5.2 3.8l-25-31.2c-0.5 .5-1.1 .9-1.6 1.4l25.5 30.8c.4-0.3 .7-0.7 1.1-1Zm-4.9 4.3l-27.1-29.5c-0.5 .5-1 1-1.5 1.5l27.5 29c.4-0.4 .7-0.7 1.1-1Zm-4.6 4.5l-29-27.5c-0.5 .5-1 1-1.5 1.5l29.5 27.1c.3-0.4 .6-0.7 1-1.1Zm-4.3 4.9l-30.8-25.5c-0.5 .5-0.9 1.1-1.4 1.6l31.2 25c.3-0.4 .7-0.7 1-1.1Zm-4 5.1l-32.5-23.3c-0.4 .5-0.8 1.1-1.2 1.7l32.9 22.8c.2-0.4 .5-0.8 .8-1.2Zm-3.6 5.4l-34-21.1c-0.4 .6-0.7 1.2-1.1 1.8l34.3 20.6c.3-0.5 .5-0.9 .8-1.3Zm-3.2 5.6l-35.4-18.7c-0.3 .6-0.6 1.3-1 1.9l35.7 18.1c.2-0.4 .4-0.8 .7-1.3Zm-2.9 5.9l-36.5-16.3c-0.3 .7-0.6 1.3-0.9 2l36.8 15.6c.2-0.4 .4-0.9 .6-1.3Zm-2.4 6l-37.6-13.7c-0.2 .6-0.4 1.3-0.7 2l37.8 13.1c.2-0.5 .3-1 .5-1.4Zm-2 6.1l-38.4-11c-0.2 .6-0.4 1.3-0.6 2l38.6 10.5c.1-0.5 .3-1 .4-1.5Zm-1.6 6.3l-39.1-8.4c-0.1 .7-0.3 1.4-0.4 2.1l39.2 7.8c.1-0.5 .2-1 .3-1.5Zm-1.1 6.4l-39.6-5.7c-0.1 .7-0.2 1.4-0.3 2.1l39.7 5.1c0-0.5 .1-1 .2-1.5Zm-0.7 6.5l-39.9-3c-0.1 .7-0.1 1.4-0.2 2.1l40 2.4c0-0.5 0-1 .1-1.5Zm-0.3 6.4c0 .3 0 .5 0 .8c0 .3 0 .5 0 .8l-40 .3c0-0.4 0-0.7 0-1.1c0-0.4 0-0.7 0-1.1l40 .3Zm.2 6.5l-40 2.4c.1 .7 .1 1.4 .2 2.1l39.9-3c-0.1-0.5-0.1-1-0.1-1.5Zm.6 6.5l-39.7 5.1c.1 .7 .2 1.4 .3 2.1l39.6-5.7c-0.1-0.5-0.2-1-0.2-1.5Zm1 6.4l-39.2 7.8c.1 .7 .3 1.4 .4 2.1l39.1-8.4c-0.1-0.5-0.2-1-0.3-1.5Zm1.5 6.3l-38.6 10.5c.2 .7 .4 1.4 .6 2l38.4-11c-0.1-0.5-0.3-1-0.4-1.5Zm1.9 6.2l-37.8 13.1c.3 .7 .5 1.4 .7 2l37.6-13.7c-0.2-0.4-0.3-0.9-0.5-1.4Zm2.3 6.1l-36.8 15.6c.3 .7 .6 1.3 .9 2l36.5-16.3c-0.2-0.4-0.4-0.9-0.6-1.3Zm2.8 5.8l-35.7 18.2c.4 .6 .7 1.3 1 1.9l35.4-18.7c-0.3-0.5-0.5-0.9-0.7-1.4Zm3.1 5.7l-34.3 20.6c.4 .6 .7 1.2 1.1 1.8l34-21.1c-0.3-0.4-0.5-0.8-0.8-1.3Zm3.6 5.5l-32.9 22.8c.4 .6 .8 1.2 1.2 1.7l32.5-23.3c-0.3-0.4-0.6-0.8-0.8-1.2Zm3.8 5.2l-31.2 25c.5 .5 .9 1.1 1.4 1.6l30.8-25.5c-0.3-0.4-0.7-0.7-1-1.1Zm4.3 4.9l-29.5 27.1c.5 .5 1 1 1.5 1.5l29-27.5c-0.4-0.4-0.7-0.7-1-1.1Zm4.5 4.6l-27.5 29c.5 .5 1 1 1.5 1.5l27.1-29.5c-0.4-0.3-0.7-0.6-1.1-1Zm4.9 4.3l-25.5 30.8c.5 .5 1.1 .9 1.6 1.4l25-31.2c-0.4-0.3-0.7-0.7-1.1-1Zm5.1 4l-23.3 32.5c.5 .4 1.1 .8 1.7 1.2l22.8-32.9c-0.4-0.2-0.8-0.5-1.2-0.8Zm5.4 3.6l-21.1 34c.6 .4 1.2 .7 1.8 1.1l20.6-34.3c-0.5-0.3-0.9-0.5-1.3-0.8Zm5.6 3.2l-18.7 35.4c.6 .3 1.3 .6 1.9 1l18.2-35.7c-0.5-0.2-0.9-0.4-1.4-0.7Zm5.9 2.9l-16.3 36.5c.7 .3 1.3 .6 2 .9l15.6-36.8c-0.4-0.2-0.9-0.4-1.3-0.6Zm6 2.4l-13.7 37.6c.6 .2 1.3 .4 2 .7l13.1-37.8c-0.5-0.2-1-0.3-1.4-0.5Zm6.1 2l-11 38.4c.6 .2 1.3 .4 2 .6l10.5-38.6c-0.5-0.1-1-0.3-1.5-0.4Zm6.3 1.6l-8.4 39.1c.7 .1 1.4 .3 2.1 .4l7.8-39.2c-0.5-0.1-1-0.2-1.5-0.3Zm6.4 1.1l-5.7 39.6c.7 .1 1.4 .2 2.1 .3l5.1-39.7c-0.5 0-1-0.1-1.5-0.2Zm6.5 .7l-3 39.9c.7 .1 1.4 .1 2.1 .2l2.4-40c-0.5 0-1 0-1.5-0.1Zm6.4 .3l-0.3 40c.4 0 .7 0 1.1 0c.4 0 .7 0 1.1 0l-0.3-40c-0.3 0-0.5 0-0.8 0c-0.3 0-0.5 0-0.8 0Zm6.5-0.2l2.4 40c.7-0.1 1.4-0.1 2.1-0.2l-3-39.9c-0.5 .1-1 .1-1.5 .1Zm6.5-0.6l5.1 39.7c.7-0.1 1.4-0.2 2.1-0.3l-5.7-39.6c-0.5 .1-1 .2-1.5 .2Zm6.4-1l7.8 39.2c.7-0.1 1.4-0.3 2.1-0.4l-8.4-39.1c-0.5 .1-1 .2-1.5 .3Zm6.3-1.5l10.5 38.6c.7-0.2 1.4-0.4 2-0.6l-11-38.4c-0.5 .1-1 .3-1.5 .4Zm6.2-1.9l13.1 37.8c.7-0.3 1.4-0.5 2-0.7l-13.7-37.6c-0.4 .2-0.9 .3-1.4 .5Zm6.1-2.3l15.6 36.8c.7-0.3 1.3-0.6 2-0.9l-16.3-36.5c-0.4 .2-0.9 .4-1.3 .6Zm5.9-2.8l18.1 35.7c.6-0.4 1.3-0.7 1.9-1l-18.7-35.4c-0.5 .3-0.9 .5-1.3 .7Zm5.6-3.1l20.6 34.3c.6-0.4 1.2-0.7 1.8-1.1l-21.1-34c-0.4 .3-0.8 .5-1.3 .8Zm5.5-3.6l22.8 32.9c.6-0.4 1.2-0.8 1.7-1.2l-23.3-32.5c-0.4 .3-0.8 .6-1.2 .8Zm5.2-3.8l25 31.2c.5-0.5 1.1-0.9 1.6-1.4l-25.5-30.8c-0.4 .3-0.7 .7-1.1 1Zm4.9-4.3l27.1 29.5c.5-0.5 1-1 1.5-1.5l-27.5-29c-0.4 .4-0.7 .7-1.1 1Zm4.6-4.5l29 27.5c.5-0.5 1-1 1.5-1.5l-29.5-27.1c-0.3 .4-0.6 .7-1 1.1Zm4.3-4.9l30.8 25.5c.5-0.5 .9-1.1 1.4-1.6l-31.2-25c-0.3 .4-0.7 .7-1 1.1Zm4-5.1l32.5 23.3c.4-0.5 .8-1.1 1.2-1.7l-32.9-22.8c-0.2 .4-0.5 .8-0.8 1.2Zm3.6-5.4l34 21.1c.4-0.6 .7-1.2 1.1-1.8l-34.3-20.6c-0.3 .5-0.5 .9-0.8 1.3Zm3.2-5.6l35.4 18.7c.3-0.6 .6-1.3 1-1.9l-35.7-18.1c-0.2 .4-0.4 .8-0.7 1.3Zm2.9-5.9l36.5 16.3c.3-0.7 .6-1.3 .9-2l-36.8-15.6c-0.2 .4-0.4 .9-0.6 1.3Zm2.4-6l37.6 13.7c.2-0.6 .4-1.3 .7-2l-37.8-13.1c-0.2 .5-0.3 1-0.5 1.4Zm2-6.1l38.4 11c.2-0.6 .4-1.3 .6-2l-38.6-10.5c-0.1 .5-0.3 1-0.4 1.5Zm1.6-6.3l39.1 8.4c.1-0.7 .3-1.4 .4-2.1l-39.2-7.8c-0.1 .5-0.2 1-0.3 1.5Zm1.1-6.4l39.6 5.7c.1-0.7 .2-1.4 .3-2.1l-39.7-5.1c0 .5-0.1 1-0.2 1.5Zm.7-6.5l39.9 3c.1-0.7 .1-1.4 .2-2.1l-40-2.4c0 .5 0 1-0.1 1.5Z"
            fill="#ff1d4f"
            fill-opacity=".1"
          />
          <g mask="url(#Mask-5)" transform="translate(-75,-75)">
            <path
              d="M136 289.5c-17.7 0-35.3-3.5-51.7-10.3c-16.3-6.8-31.2-16.7-43.8-29.2c-12.5-12.6-22.4-27.5-29.2-43.8c-6.8-16.4-10.3-34-10.3-51.7h135v135Z"
              fill="#ff1d4f"
              transform="translate(210,210) rotate(-0.3) translate(-136,-154.5)"
              style={{ animation: ".1s linear 1.2s both a19_t" }}
            />
            <path
              d="M136 289.5c-17.7 0-35.3-3.5-51.7-10.3c-16.3-6.8-31.2-16.7-43.8-29.2c-12.5-12.6-22.4-27.5-29.2-43.8c-6.8-16.4-10.3-34-10.3-51.7h135v135Z"
              fill="#ff1d4f"
              transform="translate(210,210) rotate(-0.3) translate(-136,-154.5)"
              style={{ animation: ".2s linear 1.2s both a20_t" }}
            />
            <path
              d="M136 289.5c-17.7 0-35.3-3.5-51.7-10.3c-16.3-6.8-31.2-16.7-43.8-29.2c-12.5-12.6-22.4-27.5-29.2-43.8c-6.8-16.4-10.3-34-10.3-51.7h135v135Z"
              fill="#ff1d4f"
              transform="translate(210,210) rotate(-0.3) translate(-136,-154.5)"
              style={{ animation: ".3s linear 1.2s both a21_t" }}
            />
            <path
              d="M136 289.5c-17.7 0-35.3-3.5-51.7-10.3c-16.3-6.8-31.2-16.7-43.8-29.2c-12.5-12.6-22.4-27.5-29.2-43.8c-6.8-16.4-10.3-34-10.3-51.7h135v135Z"
              fill="#ff1d4f"
              transform="translate(210,210) rotate(-0.3) translate(-136,-154.5)"
              style={{ animation: ".4s linear 1.2s both a22_t" }}
            />
            <path
              d="M136 289.5c-17.7 0-35.3-3.5-51.7-10.3c-16.3-6.8-31.2-16.7-43.8-29.2c-12.5-12.6-22.4-27.5-29.2-43.8c-6.8-16.4-10.3-34-10.3-51.7h135v135Z"
              fill="#ff1d4f"
              transform="translate(210,210) rotate(-0.3) translate(-136,-154.5)"
              style={{ animation: "1s linear 1.2s both a23_t" }}
            />
            <path
              fill-rule="evenodd"
              clip-rule="evenodd"
              d="M209.4 75c.2 0 .4 0 .6 0c.2 0 .3 0 .5 0c.2 0 .4 0 .6 0l-0.3 40c-0.3 0-0.5 0-0.8 0c-0.3 0-0.5 0-0.8 0l-0.3-40c.2 0 .3 0 .5 0Zm-9.6 .4c.7-0.1 1.4-0.1 2.1-0.2l2.4 40c-0.5 0-1 0-1.5 .1Zm-9.2 1c.7-0.1 1.4-0.2 2.1-0.3l5.1 39.7c-0.5 0-1 .1-1.5 .2Zm-9.1 1.6c.7-0.1 1.4-0.3 2.1-0.4l7.8 39.2c-0.5 .1-1 .2-1.5 .3Zm-8.9 2.3c.6-0.2 1.3-0.4 2-0.6l10.5 38.6c-0.5 .1-1 .3-1.5 .4Zm-8.8 2.9c.7-0.3 1.3-0.5 2-0.8l13.1 37.8c-0.5 .2-1 .3-1.4 .5Zm-8.6 3.4c.7-0.3 1.3-0.6 2-0.8l15.6 36.7c-0.4 .2-0.9 .4-1.3 .6Zm-8.3 4c.6-0.3 1.3-0.6 1.9-0.9l18.2 35.6c-0.5 .2-0.9 .4-1.4 .7Zm-8 4.6c.6-0.3 1.2-0.7 1.8-1.1l20.6 34.3c-0.5 .3-0.9 .5-1.3 .8Zm-7.6 5.1c.5-0.4 1.1-0.8 1.7-1.2l22.8 32.9c-0.4 .2-0.8 .5-1.2 .8Zm-7.3 5.7c.5-0.5 1.1-0.9 1.6-1.4l25 31.2c-0.4 .3-0.7 .7-1.1 1Zm-6.9 6.1c.5-0.5 1-1 1.5-1.5l27.1 29.5c-0.4 .3-0.7 .6-1.1 1Zm-6.5 6.5c.5-0.5 1-1 1.5-1.5l29 27.5c-0.4 .4-0.7 .7-1 1.1Zm-6 7c.5-0.5 .9-1.1 1.4-1.6l30.8 25.5c-0.3 .4-0.7 .7-1 1.1Zm-5.5 7.4c.4-0.6 .8-1.2 1.2-1.7l32.5 23.3c-0.3 .4-0.6 .8-0.8 1.2Zm-5 7.7c.4-0.6 .8-1.2 1.1-1.8l34 21.1c-0.3 .4-0.5 .8-0.8 1.3Zm-4.4 8.1c.3-0.6 .6-1.3 .9-1.9l35.4 18.7c-0.3 .5-0.5 .9-0.7 1.4Zm-3.9 8.4c.2-0.7 .5-1.3 .8-2l36.5 16.3c-0.2 .4-0.4 .9-0.6 1.3Zm-3.4 8.6c.3-0.7 .5-1.3 .8-2l37.5 13.7c-0.2 .4-0.3 .9-0.5 1.4Zm-2.7 8.8c.2-0.7 .4-1.3 .6-2l38.4 11c-0.1 .5-0.3 1-0.4 1.5Zm-2.1 9c.1-0.7 .3-1.4 .4-2.1l39.1 8.4c-0.1 .5-0.2 1-0.3 1.5Zm-1.5 9.1c.1-0.7 .2-1.4 .3-2.1l39.6 5.7c-0.1 .5-0.2 1-0.2 1.5Zm-0.9 9.2c.1-0.7 .1-1.4 .2-2.1l39.9 3c-0.1 .5-0.1 1-0.1 1.5Zm-0.2 7c0 .2 0 .4 0 .6c0 .2 0 .3 0 .5h40c0-0.3 0-0.5 0-0.8Zm234.4 92.5c-0.5 .5-1 1-1.5 1.5l-29-27.5c.4-0.4 .7-0.7 1-1.1Zm6-7c-0.5 .5-0.9 1.1-1.4 1.6l-30.8-25.5c.3-0.4 .7-0.7 1-1.1Zm5.5-7.4c-0.4 .6-0.8 1.2-1.2 1.7l-32.5-23.3c.3-0.4 .6-0.8 .8-1.2Zm5-7.7c-0.4 .6-0.8 1.2-1.1 1.8l-34-21.1c.3-0.4 .5-0.8 .8-1.3Zm4.4-8.1c-0.3 .6-0.6 1.3-0.9 1.9l-35.4-18.7c.3-0.5 .5-0.9 .7-1.3Zm3.9-8.4c-0.2 .7-0.5 1.3-0.8 2l-36.5-16.3c.2-0.4 .4-0.9 .6-1.3Zm3.4-8.6c-0.3 .7-0.5 1.3-0.8 2l-37.5-13.7c.2-0.4 .3-0.9 .5-1.4Zm2.7-8.8c-0.2 .7-0.4 1.3-0.6 2l-38.4-11c.1-0.5 .3-1 .4-1.5Zm2.1-9c-0.1 .7-0.3 1.4-0.4 2.1l-39.1-8.4c.1-0.5 .2-1 .3-1.5Zm1.5-9.1c-0.1 .7-0.2 1.4-0.3 2.1l-39.6-5.7c.1-0.5 .2-1 .2-1.5Zm.9-9.2c-0.1 .7-0.1 1.4-0.2 2.1l-39.9-3c.1-0.5 .1-1 .1-1.5Zm.2-7c0-0.4 0-0.7 0-1.1c0-0.4 0-0.7 0-1.1l-40 .3c0 .3 0 .5 0 .8c0 .3 0 .5 0 .8l40 .3Zm-0.4-11.3c.1 .7 .1 1.4 .2 2.1l-40 2.4c0-0.5 0-1-0.1-1.5Zm-1-9.2c.1 .7 .2 1.4 .3 2.1l-39.7 5.1c0-0.5-0.1-1-0.2-1.5Zm-1.6-9.1c.1 .7 .3 1.4 .4 2.1l-39.2 7.8c-0.1-0.5-0.2-1-0.3-1.5Zm-2.3-8.9c.2 .6 .4 1.3 .6 2l-38.6 10.5c-0.1-0.5-0.3-1-0.4-1.5Zm-2.9-8.8c.3 .7 .5 1.3 .8 2l-37.8 13.1c-0.2-0.5-0.3-1-0.5-1.4Zm-3.4-8.6c.3 .7 .6 1.3 .8 2l-36.7 15.6c-0.2-0.4-0.4-0.9-0.6-1.3Zm-4-8.3c.3 .6 .6 1.3 .9 1.9l-35.6 18.2c-0.2-0.5-0.4-0.9-0.7-1.4Zm-4.6-8c.3 .6 .7 1.2 1.1 1.8l-34.3 20.6c-0.3-0.5-0.5-0.9-0.8-1.3Zm-5.1-7.6c.4 .5 .8 1.1 1.2 1.7l-32.9 22.8c-0.2-0.4-0.5-0.8-0.8-1.2Zm-5.7-7.3c.5 .5 .9 1.1 1.4 1.6l-31.2 25c-0.3-0.4-0.7-0.7-1-1.1Zm-6.1-6.9c.5 .5 1 1 1.5 1.5l-29.5 27.1c-0.3-0.4-0.6-0.7-1-1.1Zm-6.5-6.5c.5 .5 1 1 1.5 1.5l-27.5 29c-0.4-0.4-0.7-0.7-1.1-1Zm-7-6c.5 .5 1.1 .9 1.6 1.4l-25.5 30.8c-0.4-0.3-0.7-0.7-1.1-1Zm-7.4-5.5c.6 .4 1.2 .8 1.7 1.2l-23.3 32.5c-0.4-0.3-0.8-0.6-1.2-0.8Zm-7.7-5c.6 .4 1.2 .8 1.8 1.1l-21.1 34c-0.4-0.3-0.8-0.5-1.3-0.8Zm-8.1-4.4c.6 .3 1.3 .6 1.9 .9l-18.7 35.4c-0.5-0.3-0.9-0.5-1.3-0.7Zm-8.4-3.9c.7 .2 1.3 .5 2 .8l-16.3 36.5c-0.4-0.2-0.9-0.4-1.3-0.6Zm-8.6-3.4c.7 .3 1.3 .5 2 .8l-13.7 37.5c-0.4-0.2-0.9-0.3-1.4-0.5Zm-8.8-2.7c.7 .2 1.4 .4 2 .6l-11 38.4c-0.5-0.1-1-0.3-1.5-0.4Zm-9-2.1c.7 .1 1.4 .3 2.1 .4l-8.4 39.1c-0.5-0.1-1-0.2-1.5-0.3Zm-9.1-1.5c.7 .1 1.4 .2 2.1 .3l-5.7 39.6c-0.5-0.1-1-0.2-1.5-0.2Zm-9.2-0.9c.7 .1 1.4 .1 2.1 .2l-3 39.9c-0.5-0.1-1-0.1-1.5-0.1Z"
              fill="url(#Gradient-15)"
              opacity="0"
              style={{ animation: "1.4s linear 2.2s both a24_o" }}
            />
          </g>
        </g>
        <g transform="translate(-169,-169)">
          <ellipse
            fill="url(#Gradient-16)"
            rx="210"
            ry="210"
            opacity="0"
            transform="translate(379,379)"
            style={{ animation: "2.4s linear both a25_o" }}
          />
          <g mask="url(#Mask-6)" transform="translate(169,169)">
            <ellipse
              fill="url(#Gradient-17)"
              rx="379"
              ry="379"
              opacity="0"
              transform="translate(271.2,271.2)"
              style={{ animation: "2.4s linear 1.2s both a26_t, 3.8s linear 1.2s both a26_o" }}
            />
            <ellipse
              fill="url(#Gradient-17)"
              rx="379"
              ry="379"
              opacity="0"
              transform="translate(211.1,211.1)"
              style={{
                animation: "10.8s linear 3.6s infinite both a27_t, .5s linear 3.5s both a27_o",
              }}
            />
          </g>
          <ellipse
            stroke="url(#Gradient-18)"
            stroke-opacity=".24"
            stroke-width="2"
            rx="209"
            ry="209"
            opacity=".2"
            transform="translate(379,379)"
            style={{ animation: "2s linear 1.2s both a28_o" }}
          />
        </g>
        <g>
          <path
            d="M61.5 61.5l297 297"
            stroke="url(#Gradient-19)"
            stroke-width="3"
            opacity=".3"
            transform="translate(210,210) rotate(-225) translate(-210,-210)"
            style={{ animation: "1.2s linear 1.2s both a29_t, 1.3s linear 1.2s both a29_o" }}
          />
          <path
            d="M61.5 61.5l297 297"
            stroke="url(#Gradient-19)"
            stroke-width="3"
            opacity="0"
            transform="translate(210,210) rotate(-180) translate(-210,-210)"
            style={{
              animation: ".3s linear 2.5s infinite both a30_t, .1s linear 2.4s both a30_o",
            }}
          />
          <path
            d="M107.5 107.5l205 205"
            stroke="url(#Gradient-20)"
            stroke-width="3"
            opacity=".3"
            transform="translate(210,210) rotate(-225) translate(-210,-210)"
            style={{ animation: "1.2s linear 1.2s both a31_t, 1.3s linear 1.2s both a31_o" }}
          />
          <path
            d="M107.5 107.5l205 205"
            stroke="url(#Gradient-20)"
            stroke-width="3"
            opacity="0"
            transform="translate(210,210) translate(-210,-210)"
            style={{
              animation: ".3s linear 2.5s infinite both a32_t, .1s linear 2.4s both a32_o",
            }}
          />
        </g>
      </g>
    </svg>
  );
}

// function SVGLightFromAbove() {
//   return (
//     <svg
//       xmlns="http://www.w3.org/2000/svg"
//       width="100%"
//       height="100%"
//       fill="none"
//       viewBox="0 0 936 908"
//     >
//       <g clipPath="url(#clip0_4722_124)">
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_4722_124)">
//           <ellipse
//             cx="443.447"
//             cy="468.6"
//             fill="url(#paint0_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="28.288"
//             ry="253.161"
//             transform="rotate(6.216 443.447 468.6)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter1_f_4722_124)">
//           <ellipse
//             cx="530.886"
//             cy="436.627"
//             fill="url(#paint1_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="22.87"
//             ry="253.079"
//             transform="rotate(-8.838 530.886 436.627)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_4722_124)">
//           <ellipse
//             cx="628.47"
//             cy="483.756"
//             fill="url(#paint2_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.314"
//             ry="329.053"
//             transform="rotate(-23.838 628.47 483.756)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_4722_124)">
//           <ellipse
//             cx="558.498"
//             cy="325.391"
//             fill="url(#paint3_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.314"
//             ry="155.918"
//             transform="rotate(-23.838 558.498 325.391)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_4722_124)">
//           <ellipse
//             cx="543.86"
//             cy="538.324"
//             fill="url(#paint4_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.202"
//             ry="329.24"
//             transform="rotate(-8.838 543.86 538.324)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_4722_124)">
//           <ellipse
//             cx="513.88"
//             cy="346.911"
//             fill="url(#paint5_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="277.459"
//             ry="161.815"
//             transform="rotate(-8.838 513.88 346.911)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_4722_124)">
//           <ellipse
//             cx="501.682"
//             cy="268.456"
//             fill="url(#paint6_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="138.514"
//             ry="82.418"
//             transform="rotate(-8.838 501.682 268.456)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_4722_124)">
//           <ellipse
//             cx="502.643"
//             cy="274.639"
//             fill="url(#paint7_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="116.507"
//             ry="69.257"
//             transform="rotate(-8.838 502.643 274.639)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter8_f_4722_124)">
//           <ellipse
//             cx="338.443"
//             cy="427.461"
//             fill="url(#paint8_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="28.288"
//             ry="253.161"
//             transform="rotate(26.964 338.443 427.461)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter9_f_4722_124)">
//           <ellipse
//             cx="431.538"
//             cy="428.537"
//             fill="url(#paint9_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="22.87"
//             ry="253.079"
//             transform="rotate(11.91 431.538 428.537)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter10_f_4722_124)">
//           <ellipse
//             cx="506.098"
//             cy="507.179"
//             fill="url(#paint10_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.314"
//             ry="329.053"
//             transform="rotate(-3.09 506.098 507.179)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter11_f_4722_124)">
//           <ellipse
//             cx="496.765"
//             cy="334.296"
//             fill="url(#paint11_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.314"
//             ry="155.918"
//             transform="rotate(-3.09 496.765 334.296)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter12_f_4722_124)">
//           <ellipse
//             cx="407.644"
//             cy="528.235"
//             fill="url(#paint12_linear_4722_124)"
//             fillOpacity="0.4"
//             rx="19.202"
//             ry="329.24"
//             transform="rotate(11.91 407.644 528.235)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter13_f_4722_124)">
//           <ellipse
//             cx="447.417"
//             cy="338.615"
//             fill="url(#paint13_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="277.459"
//             ry="161.815"
//             transform="rotate(11.91 447.417 338.615)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter14_f_4722_124)">
//           <ellipse
//             cx="463.802"
//             cy="260.927"
//             fill="url(#paint14_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="138.514"
//             ry="82.418"
//             transform="rotate(11.91 463.802 260.927)"
//           />
//         </g>
//         <g style={{ mixBlendMode: "lighten" }} filter="url(#filter15_f_4722_124)">
//           <ellipse
//             cx="462.512"
//             cy="267.049"
//             fill="url(#paint15_linear_4722_124)"
//             fillOpacity="0.3"
//             rx="116.507"
//             ry="69.257"
//             transform="rotate(11.91 462.512 267.049)"
//           />
//         </g>
//       </g>
//       <defs>
//         <filter
//           id="filter0_f_4722_124"
//           width="256.543"
//           height="681.384"
//           x="315.176"
//           y="127.908"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter1_f_4722_124"
//           width="267.959"
//           height="678.199"
//           x="396.906"
//           y="97.528"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter2_f_4722_124"
//           width="446.34"
//           height="780.172"
//           x="405.3"
//           y="93.67"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter3_f_4722_124"
//           width="308.921"
//           height="463.67"
//           x="404.037"
//           y="93.556"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter4_f_4722_124"
//           width="286.079"
//           height="828.689"
//           x="400.82"
//           y="123.979"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter5_f_4722_124"
//           width="850.614"
//           height="631.04"
//           x="88.573"
//           y="31.391"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <filter
//           id="filter6_f_4722_124"
//           width="574.925"
//           height="468.389"
//           x="214.22"
//           y="34.262"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <filter
//           id="filter7_f_4722_124"
//           width="531.243"
//           height="441.508"
//           x="237.021"
//           y="53.885"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <filter
//           id="filter8_f_4722_124"
//           width="413.104"
//           height="630.024"
//           x="131.892"
//           y="112.449"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter9_f_4722_124"
//           width="291.671"
//           height="673.354"
//           x="285.703"
//           y="91.86"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter10_f_4722_124"
//           width="230.404"
//           height="835.153"
//           x="390.896"
//           y="89.603"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter11_f_4722_124"
//           width="220.086"
//           height="489.391"
//           x="386.722"
//           y="89.601"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter12_f_4722_124"
//           width="319.028"
//           height="822.355"
//           x="248.13"
//           y="117.058"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="44.5" />
//         </filter>
//         <filter
//           id="filter13_f_4722_124"
//           width="847.121"
//           height="636.827"
//           x="23.856"
//           y="20.201"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <filter
//           id="filter14_f_4722_124"
//           width="573.22"
//           height="471.167"
//           x="177.192"
//           y="25.343"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <filter
//           id="filter15_f_4722_124"
//           width="529.807"
//           height="443.849"
//           x="197.608"
//           y="45.124"
//           colorInterpolationFilters="sRGB"
//           filterUnits="userSpaceOnUse"
//         >
//           <feFlood floodOpacity="0" result="BackgroundImageFix" />
//           <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
//           <feGaussianBlur result="effect1_foregroundBlur_4722_124" stdDeviation="75" />
//         </filter>
//         <linearGradient
//           id="paint0_linear_4722_124"
//           x1="443.447"
//           x2="443.447"
//           y1="215.438"
//           y2="721.761"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint1_linear_4722_124"
//           x1="530.886"
//           x2="530.886"
//           y1="183.548"
//           y2="689.706"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint2_linear_4722_124"
//           x1="628.47"
//           x2="628.47"
//           y1="154.703"
//           y2="812.81"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint3_linear_4722_124"
//           x1="558.498"
//           x2="558.498"
//           y1="169.472"
//           y2="481.309"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint4_linear_4722_124"
//           x1="543.86"
//           x2="543.86"
//           y1="209.084"
//           y2="867.564"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint5_linear_4722_124"
//           x1="513.88"
//           x2="513.88"
//           y1="185.096"
//           y2="508.726"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint6_linear_4722_124"
//           x1="501.682"
//           x2="501.682"
//           y1="186.038"
//           y2="350.874"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint7_linear_4722_124"
//           x1="502.643"
//           x2="502.643"
//           y1="205.382"
//           y2="343.896"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint8_linear_4722_124"
//           x1="338.443"
//           x2="338.443"
//           y1="174.3"
//           y2="680.623"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint9_linear_4722_124"
//           x1="431.538"
//           x2="431.538"
//           y1="175.458"
//           y2="681.616"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint10_linear_4722_124"
//           x1="506.098"
//           x2="506.098"
//           y1="178.126"
//           y2="836.233"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint11_linear_4722_124"
//           x1="496.765"
//           x2="496.765"
//           y1="178.378"
//           y2="490.214"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint12_linear_4722_124"
//           x1="407.644"
//           x2="407.644"
//           y1="198.995"
//           y2="857.475"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint13_linear_4722_124"
//           x1="447.417"
//           x2="447.417"
//           y1="176.8"
//           y2="500.43"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint14_linear_4722_124"
//           x1="463.802"
//           x2="463.802"
//           y1="178.509"
//           y2="343.344"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <linearGradient
//           id="paint15_linear_4722_124"
//           x1="462.512"
//           x2="462.512"
//           y1="197.792"
//           y2="336.306"
//           gradientUnits="userSpaceOnUse"
//         >
//           <stop stopColor="#fff" />
//           <stop offset="1" stopColor="#fff" stopOpacity="0" />
//         </linearGradient>
//         <clipPath id="clip0_4722_124">
//           <path fill="#fff" d="M0 0H936V908H0z" />
//         </clipPath>
//       </defs>
//     </svg>
//   );
// }

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
