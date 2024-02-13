"use client";
import FsLightbox from "fslightbox-react";
import { useState } from "react";

export function YoutubeEmbed() {
  const [toggler, setToggler] = useState(false);

  return (
    <>
      <button type="button" onClick={() => setToggler(!toggler)}>
        <img src="/images/hero.png" alt="Youtube" />
      </button>
      <FsLightbox
        toggler={toggler}
        sources={[
          <div className="h-[1000px] w-[1200px]">
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
    </>
  );
}
