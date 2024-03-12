"use client";
import { BorderBeam } from "@/components/border-beam";
import FsLightbox from "fslightbox-react";
import Image from "next/image";
import { useState } from "react";

export function YoutubeEmbed() {
  const [toggler, setToggler] = useState(false);

  return (
    <div className="rounded-[38px] bg-white/5 border border-gray-800/40 z-10 mt-16 xl:mt-0 ">
      <div className="m-[10px] rounded-[28px] flex items-center justify-center">
        <div className="flex items-center justify-center">
          <button
            type="button"
            className="relative rounded-[28px]"
            onClick={() => setToggler(!toggler)}
          >
            <Image src="/images/hero.png" alt="Youtube" width={600} height={340} />
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
    </div>
  );
}
