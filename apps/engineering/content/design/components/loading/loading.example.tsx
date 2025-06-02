"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button, Loading } from "@unkey/ui";
import { useState } from "react";

export const LoadingExample = () => {
  const [loading, setLoading] = useState(false);

  const handleClick = () => {
    setLoading(true);
    setTimeout(() => setLoading(false), 3000);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-row items-start justify-between gap-2">
        <div className="flex flex-col items-center justify-start gap-2">
          <p>Default</p>
          <Button onClick={handleClick} size="lg">
            {loading ? <Loading /> : "Click me"}
          </Button>
        </div>
        <div className="flex flex-col items-center justify-start gap-2">
          <p>Duration</p>
          <Button onClick={handleClick} size="lg">
            {loading ? <Loading dur="1s" /> : "Click me"}
          </Button>
        </div>
        <div className="flex flex-col items-center justify-start gap-2">
          <p>Custom Size</p>
          <Button onClick={handleClick} size="2xlg" className="shrink-0 w-32">
            {loading ? <Loading width={24} height={24} /> : "Click me"}
          </Button>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
