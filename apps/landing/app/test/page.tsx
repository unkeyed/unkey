"use client";
import { AnimatePresence, motion } from "framer-motion";
import { nanoid } from "nanoid";
import { useEffect, useState } from "react";

const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
export const generateRandomString = (length: number) => {
  let result = "";
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
};

// Function to wrap each letter in a span, coloring every eighth letter green
const renderStringWithColor = (str) => {
  return str.split("").map((char, index) => {
    const id = nanoid();
    return (
      <span
        key={id}
        style={{
          color: Math.random() < 0.12 ? "#3CEEAE" : Math.random() > 0.95 ? "white/40" : "white/40",
        }}
      >
        {char}
      </span>
    );
  });
};

export default function TestPage() {
  function Key() {
    return (
      <div className="relative -z-50 mr-10">
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
          <p className="relative right-4 text-[13px]">sk_TEwCE9AY9BFTq1XJdIO</p>
        </div>
      </div>
    );
  }
  const [randomString, setRandomString] = useState("");
  const [animateKey, setAnimateKey] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false); // Tracks if the key is currently animating

  // Function to trigger the animation
  const triggerAnimation = () => {
    setIsAnimating(true); // Start the animation
    setAnimateKey(!animateKey); // Toggle the animation state
  };

  useEffect(() => {
    // Initial random string generation
    setRandomString(generateRandomString(1500));
  }, []);

  const scrambleMultipleTimes = (times, interval) => {
    if (times <= 0) {
      setIsAnimating(false); // Stop the animation once scrambling is done
      return;
    }

    setTimeout(() => {
      setRandomString(generateRandomString(620)); // Update with new random string
      scrambleMultipleTimes(times - 1, interval); // Recurse with one less time
    }, interval);
  };

  return (
    <>
      <div>
        <button
          type="button"
          className="absolute top-[200px] left-[450px] text-white mr-10 border border-white px-2 rounded-md text-sm"
          onClick={() => {
            triggerAnimation();
            setTimeout(() => scrambleMultipleTimes(20, 10), 200);
          }}
        >
          Scramble
        </button>
      </div>
      <div className="h-[800px] w-full flex items-center justify-center">
        <AnimatePresence>
          {isAnimating ? (
            <motion.div
              initial={{ x: 0 }}
              animate={{ x: 2000 }}
              transition={{ duration: 5 }}
              onAnimationComplete={() => setIsAnimating(false)}
            >
              <Key />
            </motion.div>
          ) : (
            <Key />
          )}
        </AnimatePresence>
        <div className="bg-black border-white border-l h-[300px] w-[500px] overflow-hidden pt-8 pl-8 flex relative z-50">
          <p className="text-white/40 break-words whitespace-pre-wrap w-[180px] h-[240px] overflow-hidden font-mono">
            {renderStringWithColor(randomString)}
          </p>
        </div>
      </div>
    </>
  );
}
