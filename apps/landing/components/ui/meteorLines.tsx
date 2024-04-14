"use client";
import clsx from "clsx";
import { useEffect, useState } from "react";
interface MeteorsProps {
  number?: number;
  xPos?: number;
  speed?: number;
  delay?: number;
  direction?: "left" | "right";
  className?: string;
}
// function getRandomFromSet(set: number[]) {
//   return set[Math.floor(Math.random() * set.length)];
// }
const MeteorLines = ({
  number = 20,
  xPos = 60,
  speed = 10,
  delay = 0,
  direction = "left",
  className,
}: MeteorsProps) => {
  const [meteorStyles, setMeteorStyles] = useState<Array<React.CSSProperties>>([]);
  const [windowWidth, _setWindowWidth] = useState<number>(0);
  useEffect(() => {
    const width = window.innerWidth;
    const pos = direction === "left" ? xPos : width - (xPos + 75);

    const styles = [...new Array(number)].map(() => ({
      top: -50,
      left: pos,

      animationDelay: delay ? `${delay}s` : `${Math.random() * 1 + 0.2}s`,
      animationDuration: speed ? `${speed}s` : `${Math.floor(Math.random() * 10 + 2)}s`,
    }));

    setMeteorStyles(styles);
  }, [number, windowWidth]);

  return (
    <>
      {[...meteorStyles].map((style, idx) => (
        // Meteor Head
        <span
          key={idx.toString()}
          className={clsx(
            className,
            "-z-20 pointer-events-none absolute left-0 top-0 h-[.75px] w-20 rotate-[90deg] opacity-0 animate-meteor rounded-[9999px] bg-gradient-to-r from-white/90 to-transparent shadow-[0_0_0_1px_#ffffff10]",
          )}
          style={style}
        >
          {/* Meteor Tail */}
          <div className="-z-20 pointer-events-none absolute top-1/2 h-[.75px] w-[500px] -translate-y-1/2 bg-gradient-to-r from-white/10 to-transparent" />
        </span>
      ))}
    </>
  );
};

const MeteorLinesAngular = ({
  number = 20,
  xPos = 60,
  speed = 10,
  delay = 0,
  className,
}: MeteorsProps) => {
  const [meteorStyles, setMeteorStyles] = useState<Array<React.CSSProperties>>([]);
  const [windowWidth, setWindowWidth] = useState<number>(0);

  useEffect(() => {
    const width = window.innerWidth;
    setWindowWidth(width);
    const pos = width / 2 + xPos;
    const styles = [...new Array(number)].map(() => ({
      top: -100,
      left: pos,
      animationDelay: delay ? `${delay}s` : `${Math.random() * 1 + 0.2}s`,
      animationDuration: speed ? `${speed}s` : `${Math.floor(Math.random() * 10 + 2)}s`,
    }));
    setMeteorStyles(styles);
  }, [number, windowWidth]);
  return (
    <>
      {[...meteorStyles].map((style, idx) => (
        // Meteor Head
        <span
          key={idx.toString()}
          className={clsx(
            className,
            "pointer-events-none absolute left-1/2 top-0 h-[.75px] w-20 rotate-[300deg] animate-meteorAngle rounded-[9999px] bg-gradient-to-r from-white/90 to-transparent shadow-[0_0_0_1px_#ffffff10]",
          )}
          style={style}
        >
          {/* Meteor Tail */}
          <div className="-z-20 pointer-events-none absolute top-0 h-[.75px] w-[500px] -translate-y-1/2 bg-gradient-to-r from-white/10 to-transparent" />
        </span>
      ))}
    </>
  );
};

export { MeteorLines, MeteorLinesAngular };
