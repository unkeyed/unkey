"use client";
import React from "react";
import { cn } from "../lib/utils";

type AnimatedLoadingSpinnerProps = {
  className?: string;
  segmentTimeInMS?: number;
  size?: number;
};

export const AnimatedLoadingSpinner = ({
  className,
  segmentTimeInMS = 125,
  size = 18,
}: AnimatedLoadingSpinnerProps) => {
  const [segmentIndex, setSegmentIndex] = React.useState(0);

  // Each segment ID in the order they should light up
  const segments = [
    "segment-1", // Right top
    "segment-2", // Right
    "segment-3", // Right bottom
    "segment-4", // Bottom
    "segment-5", // Left bottom
    "segment-6", // Left
    "segment-7", // Left top
    "segment-8", // Top
  ];

  React.useEffect(() => {
    // Animate the segments in sequence
    const timer = setInterval(() => {
      setSegmentIndex((prevIndex) => (prevIndex + 1) % segments.length);
    }, segmentTimeInMS); // 125ms per segment = 1s for full rotation

    return () => clearInterval(timer);
  }, [segmentTimeInMS]);

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 18 18"
      className={cn("animate-spin-slow", className)}
      data-prefers-reduced-motion="respect-motion-preference"
    >
      <g>
        {segments.map((id, index) => {
          // Calculate opacity based on position relative to current index
          const distance = (segments.length + index - segmentIndex) % segments.length;
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

// Helper function to get path data for each segment
function getPathForSegment(index: number) {
  const paths = [
    "M13.162,3.82c-.148,0-.299-.044-.431-.136-.784-.552-1.662-.915-2.61-1.08-.407-.071-.681-.459-.61-.867,.071-.408,.459-.684,.868-.61,1.167,.203,2.248,.65,3.216,1.33,.339,.238,.42,.706,.182,1.045-.146,.208-.378,.319-.614,.319Z",
    "M16.136,8.5c-.357,0-.675-.257-.738-.622-.163-.942-.527-1.82-1.082-2.608-.238-.339-.157-.807,.182-1.045,.34-.239,.809-.156,1.045,.182,.683,.97,1.132,2.052,1.334,3.214,.07,.408-.203,.796-.611,.867-.043,.008-.086,.011-.129,.011Z",
    "M14.93,13.913c-.148,0-.299-.044-.431-.137-.339-.238-.42-.706-.182-1.045,.551-.784,.914-1.662,1.078-2.609,.071-.408,.466-.684,.867-.611,.408,.071,.682,.459,.611,.867-.203,1.167-.65,2.25-1.33,3.216-.146,.208-.378,.318-.614,.318Z",
    "M10.249,16.887c-.357,0-.675-.257-.738-.621-.07-.408,.202-.797,.61-.868,.945-.165,1.822-.529,2.608-1.082,.34-.238,.807-.156,1.045,.182,.238,.338,.157,.807-.182,1.045-.968,.682-2.05,1.13-3.214,1.333-.044,.008-.087,.011-.13,.011Z",
    "M7.751,16.885c-.043,0-.086-.003-.13-.011-1.167-.203-2.249-.651-3.216-1.33-.339-.238-.42-.706-.182-1.045,.236-.339,.702-.421,1.045-.183,.784,.551,1.662,.915,2.61,1.08,.408,.071,.681,.459,.61,.868-.063,.364-.381,.621-.738,.621Z",
    "M3.072,13.911c-.236,0-.469-.111-.614-.318-.683-.97-1.132-2.052-1.334-3.214-.07-.408,.203-.796,.611-.867,.403-.073,.796,.202,.867,.61,.163,.942,.527,1.82,1.082,2.608,.238,.339,.157,.807-.182,1.045-.131,.092-.282,.137-.431,.137Z",
    "M1.866,8.5c-.043,0-.086-.003-.129-.011-.408-.071-.682-.459-.611-.867,.203-1.167,.65-2.25,1.33-3.216,.236-.339,.703-.422,1.045-.182,.339,.238,.42,.706,.182,1.045-.551,.784-.914,1.662-1.078,2.609-.063,.365-.381,.622-.738,.622Z",
    "M4.84,3.821c-.236,0-.468-.111-.614-.318-.238-.338-.157-.807,.182-1.045,.968-.682,2.05-1.13,3.214-1.333,.41-.072,.797,.202,.868,.61,.07,.408-.202,.797-.61,.868-.945,.165-1.822,.529-2.608,1.082-.131,.092-.282,.137-.431,.137Z",
  ];

  return paths[index];
}
