"use client";

import {
  AnimatePresence,
  motion,
  useMotionValue,
  useTransform,
  useWillChange,
} from "framer-motion";
import { useEffect, useMemo, useRef, useState } from "react";

import { StarsSvg } from "@/components/svg/stars";
import { cn } from "@/lib/utils";

const randomStrings = [
  "9DQNgnuyEwoSUBvc8pgRQFurWW1Jp1YBcZtLZWvYsgBrB9X46HXgBiFqS9a8NUMj086OA2RiT5gbak0pyPmkAmrXS0XJ6xcSdkP3ZN5H0WR4ZX6h8g0fDBsDJggJXPHRHeMhBwcT2ZBZwC4R9MmBsbiWG1fnEDpbU5MF8aCm2exWE1LTi2aTF8iayph2zK4KbdsK4Ka4Y6kkZSrFqb5ZxS2pyGXu7fWRC3rMLtSgNkbphyr3Z0OLbz8ufG",
  "bt2my9U5vErf1w8eKSVpoxg886C4et6VVErNVfWCMRr350zx21hoohVfSp6Lbfbacl6VSDykBWFfK1K6EQLSTqNkLD6T2M3Z5gk91TtvCxPRv9fkeuMLmcSwuPxCaFeJaxE6KmJllFenaZySOVMjqQx4COqx52TtcF3OjmemBivyduCHN8qPs3M3kMpOX1hTXDRehRjQvFITSuWgjXHGeaHUBfhjKMOhk9kPBgxyTCwdtTseSryZSFBrkj",
  "yPQsISJGlMXzoQqIkg6cWMGbJYuAdK3Q0zTp7NZl4IxD6lkoigU8huCderWme7xfbzF8L5vmGj6l0XhZtm5JfyV9EWAoxtdtxwlkyb9Jdpg9gdxw9HD0NpwhJhWQgXmiZqsvsASurbvVN88XosArEvgrhQv9yxVu48ebVOtPZEIJUocLPLqaLGG3Fd4d31ZFmKaUCI5mQb57UcfsAdgfXhMH6a3rizjoAaU7szXznWkOsXqKDpCcCgVR3k",
  "AfVutbmoSwHDFaBFV3EXUUyT1QifaI1NeA06kxq9uyQ8zQIS1IoUTg0JEYBO7wJViTNiMy5J1p2aTV15tTeZpFwCBzp6uv71NVLJcaclhegX87DHmJwiUIau8l9bgbP2BLTtm9fLgJPLDOhNBkoARsP04hLkTD2eGswJHbzqUVbKYWuQQfjLNcQignmXZGB8W3lnJqwGzRSDpVYWyhR98xsHzXFE3dsjWtJYsUvebAft13Wi5uAaFVES8v",
  "s0V2MFqMgDGafXTsOKo2UDcgFIRSllbOGzHccH0Y7xKZxLAdfGya9WQeXVytdtbwLvpTZ5uN78mVmyAkGP3pIVQa0Yv5LgGdZZVFJDAY8Y0Jikj0PWh9IIZRE8P6Jlerqj0lgsj4rzkiz9RvqKNdvG1HqnNGbDUApGyG8sBOntJf7Xqkw08qnVRFnNBiIDFmsDlLTe8BDbXRyHdmojVxGrR8A8ms2DDMkTGd6Q7S82Bw2eqaY1tTuunqTg",
];
const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
export const generateRandomString = (length: number) => {
  let result = "";
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
};
export const getRandomString = () => {
  return randomStrings[Math.floor(Math.random() * 5)];
};

const normalizeXPos = (x: number) => {
  return Math.min(((x + 250) * 100) / 524);
};

// TODO: use CSS cubic-bezier
function fadeInOut(value: number) {
  if (value <= 50) {
    return 0;
  }
  if (value <= 70) {
    // Adjust this value as needed
    // Smooth fade-in from 50 to 70
    // Calculate how far into the fade-in range the value is, on a scale from 0 to 1
    const fadeInRange = (value - 50) / (70 - 50);
    // Linearly interpolate the opacity based on the fadeInRange
    return fadeInRange * 100;
  }
  if (value <= 95) {
    return 100;
  }
  // Quadratic ease out for a rapid decrease
  // Calculate how far into the final 10% the value is, on a scale from 0 to 1
  const finalStretch = (value - 95.1) / 10;
  // Reverse the ease in formula to decrease
  return Math.max(100 * (1 - finalStretch ** 2), 0);
}

const ITERATIONS = 50;
const GENERATED_STRINGS_CHARS = Array.from({ length: ITERATIONS }, getRandomString).map((str) =>
  str.split(""),
);

export function HashedKeys() {
  const willChange = useWillChange();

  const textContainerRef = useRef<HTMLParagraphElement>(null);

  const [hasReachedThreshold, setHasReachedThreshold] = useState(false);
  const opacity = useMotionValue(0);
  const transformedOpacity = useTransform(opacity, (n) => fadeInOut(n) / 100);

  useEffect(() => {
    // If the threshold is reached, start the second animation
    if (hasReachedThreshold) {
      let idx = 0;
      const interval = setInterval(() => {
        if (idx === ITERATIONS) {
          clearInterval(interval);
          return;
        }

        // TODO: Do not update DOM, stack <p> elements and update the opacity instead
        requestAnimationFrame(() => {
          if (!textContainerRef.current) {
            return;
          }
          textContainerRef.current.innerHTML = GENERATED_STRINGS_CHARS[idx]
            .map(
              (char) =>
                `<span style="color: ${
                  Math.random() < 0.12 && hasReachedThreshold ? "#3CEEAE" : "inherit"
                };">${char}</span>`,
            )
            .join("");

          idx++;
        });
      }, 20);
    }
  }, [hasReachedThreshold]);

  return (
    <div className="w-full relative flex items-center justify-end mb-[100px]">
      <StarsSvg className="absolute" />
      <AnimatePresence>
        <motion.div
          initial={{ x: "-100%" }}
          exit={{ x: "-100%" }}
          whileInView={{ x: 320 }}
          style={{ willChange }}
          transition={{
            type: "spring",
            damping: 80,
            stiffness: 100,
            mass: 12,
            delay: 1.5,
            repeat: Number.POSITIVE_INFINITY,
            duration: 1.5,
          }}
          onUpdate={(latest: { x: number }) => {
            opacity.set(normalizeXPos(latest.x));
            if (latest.x > 0) {
              setHasReachedThreshold(true);
            }
            if (latest.x < -100) {
              setHasReachedThreshold(false);
            }
          }}
        >
          <Key text="sk_TEwCE9AY9BFTq1XJdIO" className={cn("relative left-[-3px]")} />
        </motion.div>
      </AnimatePresence>
      <div className="line h-[300px] w-[0.75px] bg-gradient-to-b from-black to-black via-white relative z-50" />
      <div className="bg-[#080808] hk-radial rounded-lg w-[200px] h-[300px] overflow-hidden pt-8 pl-4 flex relative z-50 select-none">
        <motion.p
          className="text-white/40 break-words whitespace-pre-wrap w-[160px] h-[240px] overflow-hidden font-mono text-sm"
          ref={textContainerRef}
          style={{ opacity: transformedOpacity, willChange }}
        />
      </div>
    </div>
  );
}

function Key({ className, text }: { className?: string; text: string }) {
  return (
    <div className={className}>
      <div className="inline-flex items-center overflow-hidden mx-4 pr-4 h-[36px] text-white font-mono text-sm ratelimits-key-gradient border-[.75px] border-[#ffffff]/20 rounded">
        <div className="w-[62px] h-[36px]">
          <svg
            className="ratelimits-key-icon "
            xmlns="http://www.w3.org/2000/svg"
            width="62"
            height="36"
            viewBox="0 0 62 36"
            fill="none"
          >
            <g filter="url(#filter0_d_840_1930)">
              <rect
                x="8"
                y="6"
                width="24"
                height="24"
                rx="6"
                fill="#3CEEAE"
                shapeRendering="crispEdges"
              />
              <rect
                x="8"
                y="6"
                width="24"
                height="24"
                rx="6"
                fill="black"
                fillOpacity="0.15"
                shapeRendering="crispEdges"
              />
              <rect
                x="8.375"
                y="6.375"
                width="23.25"
                height="23.25"
                rx="5.625"
                stroke="white"
                strokeOpacity="0.1"
                strokeWidth="0.75"
                shapeRendering="crispEdges"
              />
              <path
                d="M21.5 15L23 16.5M14.5 23.5H17.5V21.5H19.5V20.5L21.5 18.5L19.5 16.5L14.5 21.5V23.5ZM18 15L23 20L26.5 16.5L21.5 11.5L18 15Z"
                stroke="white"
              />
            </g>
            <defs>
              <filter
                id="filter0_d_840_1930"
                x="-22"
                y="-24"
                width="84"
                height="84"
                filterUnits="userSpaceOnUse"
                colorInterpolationFilters="sRGB"
              >
                <feFlood floodOpacity="0" result="BackgroundImageFix" />
                <feColorMatrix
                  in="SourceAlpha"
                  type="matrix"
                  values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                  result="hardAlpha"
                />
                <feOffset />
                <feGaussianBlur stdDeviation="15" />
                <feComposite in2="hardAlpha" operator="out" />
                <feColorMatrix
                  type="matrix"
                  values="0 0 0 0 0.235294 0 0 0 0 0.933333 0 0 0 0 0.682353 0 0 0 1 0"
                />
                <feBlend
                  mode="normal"
                  in2="BackgroundImageFix"
                  result="effect1_dropShadow_840_1930"
                />
                <feBlend
                  mode="normal"
                  in="SourceGraphic"
                  in2="effect1_dropShadow_840_1930"
                  result="shape"
                />
              </filter>
            </defs>
          </svg>
        </div>
        <p className="relative text-xs">{text}</p>
      </div>
    </div>
  );
}
