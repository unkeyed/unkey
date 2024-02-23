"use client";
import clsx from "clsx";
import { useEffect, useState } from "react";
interface MeteorsProps {
  number?: number;
  xPos?: number;
  direction?: "left" | "right";
}
// function getRandomFromSet(set: number[]) {
//   return set[Math.floor(Math.random() * set.length)];
// }
export const MeteorLines = ({ number = 20, xPos = 60, direction = "left" }: MeteorsProps) => {
  const [meteorStyles, setMeteorStyles] = useState<Array<React.CSSProperties>>([]);
  useEffect(() => {
    const width = window.innerWidth;
    const pos = direction === "left" ? xPos : width - (xPos + 75);

    const styles = [...new Array(number)].map(() => ({
      top: -50,
      left: pos,
      animationDelay: `${Math.random() * 1 + 0.2}s`,
      animationDuration: `${Math.floor(Math.random() * 20 + 2)}s`,
    }));
    setMeteorStyles(styles);
  }, [number]);
  return (
    <>
      {[...meteorStyles].map((style, idx) => (
        // Meteor Head
        <span
          key={idx.toString()}
          className={clsx(
            "-z-20 pointer-events-none absolute left-0 top-0 h-0.5 w-20 rotate-[90deg] animate-meteor rounded-[9999px] bg-gradient-to-r from-white to-transparent shadow-[0_0_0_1px_#ffffff10]",
          )}
          style={style}
        >
          {/* Meteor Tail */}
          <div className="-z-20 pointer-events-none absolute top-1/2 h-[1px] w-[500px] -translate-y-1/2 bg-gradient-to-r from-white/10 to-transparent" />
        </span>
      ))}
    </>
  );
};
export default MeteorLines;
