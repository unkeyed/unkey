"use client";

import { motion, useReducedMotion } from "framer-motion";
import type React from "react";

function FadeInStagger({ ...props }) {
  return (
    <motion.div
      initial="hidden"
      whileInView="visible"
      viewport={{ once: false, margin: "0px 0px 0px 0px" }}
      transition={{ staggerChildren: 0.15 }}
      {...props}
    />
  );
}
export const Wordmark: React.FC<{ className?: string }> = ({ className }) => {
  const shouldReduceMotion = useReducedMotion();

  const variants = {
    hidden: { opacity: 0, y: shouldReduceMotion ? 0 : 64 },
    visible: { opacity: 1, y: 0 },
  };
  const transition = {
    duration: 0.05,
    ease: "easeOut",
    type: "spring",
    stiffness: 200,
    damping: 50,
  };

  return (
    <FadeInStagger className={className}>
      <svg
        width="1376"
        height="248"
        viewBox="0 0 1376 248"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <motion.path
          variants={variants}
          transition={transition}
          d="M0 177.333C0 259.667 51.533 307.619 141.942 307.619C232.803 307.619 284.336 259.667 284.336 177.333V0H235.515V174.167C235.515 235.238 208.392 260.572 141.942 260.572C75.4913 260.572 48.3687 235.238 48.3687 174.167V0H0V177.333Z"
          fill="url(#paint4_linear_830_5782)"
        />
        <motion.path
          variants={variants}
          transition={transition}
          d="M376.418 303.095H327.598V78.2618H372.35V147.929H375.514C382.295 109.928 412.13 73.738 469.087 73.738C531.469 73.738 562.208 115.809 562.208 167.833V303.095H513.388V180.952C513.388 138.881 494.402 117.619 447.841 117.619C398.568 117.619 376.418 142.952 376.418 191.81V303.095Z"
          fill="url(#paint3_linear_830_5782)"
        />
        <motion.path
          variants={variants}
          transition={transition}
          d="M605.71 303.095H654.531V211.262H718.721L785.172 303.095H842.581L758.501 186.381L843.034 78.262H786.076L718.721 167.381H654.531V0H605.71V303.095Z"
          fill="url(#paint0_linear_830_5782)"
        />
        <motion.path
          variants={variants}
          transition={transition}
          fillRule="evenodd"
          clipRule="evenodd"
          d="M980.059 307.619C906.376 307.619 858.007 266 858.007 190.905C858.007 120.786 905.924 73.738 979.155 73.738C1048.77 73.738 1096.23 112.19 1096.23 180.5C1096.23 188.643 1095.78 194.976 1094.43 201.762H903.664C905.472 245.19 926.718 268.262 978.703 268.262C1025.72 268.262 1045.15 252.881 1045.15 226.19V222.571H1093.97V226.643C1093.97 274.595 1046.96 307.619 980.059 307.619ZM978.251 112.19C928.526 112.19 906.828 134.357 904.116 174.619H1050.13V173.714C1050.13 132.095 1026.17 112.19 978.251 112.19Z"
          fill="url(#paint2_linear_830_5782)"
        />
        <motion.path
          variants={variants}
          transition={transition}
          d="M1136.87 380H1168.96C1215.07 380 1240.39 367.786 1258.92 327.524L1376 78.2617H1322.21L1269.32 197.238L1248.07 251.524H1244.46L1222.31 197.69L1164.9 78.2617H1110.2L1220.95 303.095L1215.52 314.857C1208.74 330.238 1200.61 335.667 1180.72 335.667H1136.87V380Z"
          fill="url(#paint1_linear_830_5782)"
        />
        <defs>
          <linearGradient
            id="paint0_linear_830_5782"
            x1="-243.049"
            y1="-228.123"
            x2="-150.186"
            y2="402.501"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="white" stopOpacity="0.4" />
            <stop offset="0.693236" stopColor="white" stopOpacity="0.1" />
          </linearGradient>
          <linearGradient
            id="paint1_linear_830_5782"
            x1="-243.049"
            y1="-228.123"
            x2="-150.186"
            y2="402.501"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="white" stopOpacity="0.4" />
            <stop offset="0.693236" stopColor="white" stopOpacity="0.1" />
          </linearGradient>
          <linearGradient
            id="paint2_linear_830_5782"
            x1="-243.049"
            y1="-228.123"
            x2="-150.186"
            y2="402.501"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="white" stopOpacity="0.4" />
            <stop offset="0.693236" stopColor="white" stopOpacity="0.1" />
          </linearGradient>
          <linearGradient
            id="paint3_linear_830_5782"
            x1="-243.049"
            y1="-228.123"
            x2="-150.186"
            y2="402.501"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="white" stopOpacity="0.4" />
            <stop offset="0.693236" stopColor="white" stopOpacity="0.1" />
          </linearGradient>
          <linearGradient
            id="paint4_linear_830_5782"
            x1="-243.049"
            y1="-228.123"
            x2="-150.186"
            y2="402.501"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="white" stopOpacity="0.4" />
            <stop offset="0.693236" stopColor="white" stopOpacity="0.1" />
          </linearGradient>
        </defs>
      </svg>
    </FadeInStagger>
  );
};
