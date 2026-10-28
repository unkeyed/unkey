"use client";

import { LazyMotion } from "framer-motion";
import type { ReactNode } from "react";

const features = import("./motion-features").then((mod) => mod.domMax);
const loadFeatures = () => features;

export function MotionProvider({ children }: { children: ReactNode }) {
  return (
    <LazyMotion features={loadFeatures} strict>
      {children}
    </LazyMotion>
  );
}
