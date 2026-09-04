"use client";

import { type HTMLMotionProps, motion, useReducedMotion } from "framer-motion";

const viewport = { once: true, margin: "0px 0px -200px" };

interface FadeInProps extends HTMLMotionProps<"div"> {
  children?: React.ReactNode;
}

export function FadeIn(props: FadeInProps) {
  const shouldReduceMotion = useReducedMotion();

  return (
    <motion.div
      variants={{
        hidden: { opacity: 0, y: shouldReduceMotion ? 0 : 24 },
        visible: { opacity: 1, y: 0 },
      }}
      transition={{ duration: 0.5 }}
      initial="hidden"
      whileInView="visible"
      viewport={viewport}
      {...props}
    />
  );
}
