"use client";

import { motion, useMotionTemplate, useScroll, useTransform } from "framer-motion";
import Image from "next/image";
import { useRef } from "react";

const MotionImage = motion(Image);

// rome-ignore lint/suspicious/noExplicitAny: it's tailwindui's code
export function GrayscaleTransitionImage(props: any) {
  // rome-ignore lint/suspicious/noExplicitAny: it's tailwindui's code
  const ref = useRef() as any;
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ["start 65%", "end 35%"],
  });
  const grayscale = useTransform(scrollYProgress, [0, 0.5, 1], [1, 0, 1]);
  const filter = useMotionTemplate`grayscale(${grayscale})`;

  return (
    <div ref={ref} className="relative group">
      <MotionImage style={{ filter }} {...props} />
      <div
        className="absolute top-0 left-0 w-full transition duration-300 opacity-0 pointer-events-none group-hover:opacity-100"
        aria-hidden="true"
      >
        <Image {...props} />
      </div>
    </div>
  );
}
