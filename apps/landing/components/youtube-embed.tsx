"use client";
import { BorderBeam } from "@/components/border-beam";
import { motion } from "framer-motion";
import FsLightbox from "fslightbox-react";
import { useState } from "react";

export function YoutubeEmbed() {
  const [toggler, setToggler] = useState(false);

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 1, ease: "easeOut" }}
      className="rounded-[38px] bg-white/5 border border-gray-800 z-10 mt-16 xl:mt-0 "
    >
      <div className="m-[10px] rounded-[28px] flex items-center justify-center">
        <div>
          <button
            type="button"
            className="relative rounded-[28px]"
            onClick={() => setToggler(!toggler)}
          >
            <BorderBeam size={400} colorFrom="#72FFF9" />
            <img src="/images/hero.png" alt="Youtube" />
            <div className="absolute top-[calc(50%-80px/2)] left-[calc(50%-112px/2)] bg-[#ffffff/30] h-[80px] w-[112px] bg-yt-button-gradient hover:bg-[#111111] transition-all rounded-[20px] flex items-center justify-center">
              <BorderBeam />
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="48"
                height="48"
                viewBox="0 0 48 48"
                fill="none"
              >
                <path d="M16 38V10L40 24L16 38Z" fill="white" />
              </svg>
            </div>
          </button>
          <FsLightbox
            toggler={toggler}
            sources={[
              <div className="h-[600px] w-[1200px]">
                <iframe
                  width="100%"
                  height="100%"
                  src="https://www.youtube.com/embed/-gvpo4SWgG8?si=1kmwJVtQ5IaZrxD7"
                  title="YouTube video player"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                  allowFullScreen
                />
              </div>,
            ]}
          />
        </div>
      </div>
    </motion.div>
  );
}
