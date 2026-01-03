import { useEffect, useState } from "react";
import { getPathForSegment } from "./utils";

// Define style ID to check for duplicates
const STYLE_ID = "animated-loading-spinner-styles";

// Add styles only once when module is loaded
if (typeof document !== "undefined" && !document.getElementById(STYLE_ID)) {
  const style = document.createElement("style");
  style.id = STYLE_ID;
  style.textContent = `
    @media (prefers-reduced-motion: reduce) {
      [data-prefers-reduced-motion="respect-motion-preference"] {
        animation: none !important;
        transition: none !important;
      }
    }
    
    @keyframes spin-slow {
      from {
        transform: rotate(0deg);
      }
      to {
        transform: rotate(360deg);
      }
    }
    
    .animate-spin-slow {
      animation: spin-slow 1.5s linear infinite;
    }
  `;
  document.head.appendChild(style);
}

const SEGMENTS = [
  "segment-1", // Right top
  "segment-2", // Right
  "segment-3", // Right bottom
  "segment-4", // Bottom
  "segment-5", // Left bottom
  "segment-6", // Left
  "segment-7", // Left top
  "segment-8", // Top
];

export const AnimatedLoadingSpinner = () => {
  const [segmentIndex, setSegmentIndex] = useState(0);

  useEffect(() => {
    // Animate the segments in sequence
    const timer = setInterval(() => {
      setSegmentIndex((prevIndex) => (prevIndex + 1) % SEGMENTS.length);
    }, 125); // 125ms per segment = 1s for full rotation

    return () => clearInterval(timer);
  }, []);

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="18"
      height="18"
      viewBox="0 0 18 18"
      className="animate-spin-slow"
      data-prefers-reduced-motion="respect-motion-preference"
    >
      <g>
        {SEGMENTS.map((id, index) => {
          const distance = (SEGMENTS.length + index - segmentIndex) % SEGMENTS.length;
          const opacity = distance <= 4 ? 1 - distance * 0.2 : 0.1;
          return (
            <path
              key={id}
              id={id}
              style={{
                fill: "currentColor",
                opacity: opacity,
                transition: "opacity 0.12s ease-in-out",
              }}
              d={getPathForSegment(index)}
            />
          );
        })}
        <path
          d="M9,6.5c-1.379,0-2.5,1.121-2.5,2.5s1.121,2.5,2.5,2.5,2.5-1.121,2.5-2.5-1.121-2.5-2.5-2.5Z"
          style={{ fill: "currentColor", opacity: 0.6 }}
        />
      </g>
    </svg>
  );
};
