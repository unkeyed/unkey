"use client";

import { AnimatePresence, motion } from "framer-motion";
import { nanoid } from "nanoid";
import { useEffect, useState } from "react";

import { StarsSvg } from "@/components/svg/stars";

const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
export const generateRandomString = (length: number) => {
  let result = "";
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
};

function Key({ className, text }: { className?: string; text: string }) {
  return (
    <div className={className}>
      <div className="inline-flex items-center overflow-hidden ml-4 h-[36px] text-white font-mono text-sm ratelimits-key-gradient border-[.75px] border-[#ffffff]/20 rounded-xl">
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
        <p className="relative right-4 text-[13px]">{text}</p>
      </div>
    </div>
  );
}

export function HashedKeys() {
  const [animating, setAnimating] = useState(false);
  const [randomString, setRandomString] = useState("");
  const [fadeOut, setFadeOut] = useState(false);

  useEffect(() => {
    // Initial random string generation
    setRandomString(generateRandomString(1500));
  }, []);

  const renderStringWithColor = (str: string) => {
    return str.split("").map((char) => {
      const id = nanoid();
      return (
        <span
          key={id}
          style={
            animating
              ? {
                  color: Math.random() < 0.12 ? "#3CEEAE" : "white/40",
                }
              : fadeOut
                ? { color: "#ffffff33" }
                : { color: "white/40" }
          }
        >
          {char}
        </span>
      );
    });
  };

  const scrambleMultipleTimes = (times: number, initialInterval: number, increaseFactor = 1.2) => {
    if (times <= 0) {
      setFadeOut(true);
      setAnimating(false);
      return;
    }

    setTimeout(() => {
      setRandomString(generateRandomString(620));
      const nextInterval = initialInterval * increaseFactor ** (50 - times);
      scrambleMultipleTimes(times - 1, nextInterval, increaseFactor);
    }, initialInterval);
  };

  return (
    <div className="w-full relative flex items-center mb-[100px]">
      <StarsSvg className="absolute" />
      <AnimatePresence>
        <motion.div
          initial={{ x: 0 }}
          whileInView={{ x: 400 }}
          transition={{ duration: 6, delay: 1, type: "inertia", velocity: 290 }}
          onViewportEnter={() => {
            setTimeout(() => {
              setAnimating(true);
              scrambleMultipleTimes(100, 10, 0.1);
            }, 1120);
          }}
        >
          <Key text="sk_TEwCE9AY9BFTq1XJdIO" />
        </motion.div>
      </AnimatePresence>
      <div className="line h-[300px] w-[0.75px] bg-gradient-to-b from-black to-black via-white relative z-50" />
      <div className="bg-[#080808] hk-radial rounded-lg w-[200px] h-[300px] overflow-hidden pt-8 pl-4 flex relative z-50 select-none">
        <p className="text-white/40 break-words whitespace-pre-wrap w-[160px] h-[240px] overflow-hidden font-mono text-sm">
          {renderStringWithColor(randomString)}
        </p>
      </div>
    </div>
  );
}
