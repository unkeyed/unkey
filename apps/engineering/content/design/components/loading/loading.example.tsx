"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Loading } from "@unkey/ui";

export const LoadingExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Loading />
    </RenderComponentWithSnippet>
  );
};

export const LoadingWithDurationExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Loading dur={"1s"} />
    </RenderComponentWithSnippet>
  );
};

export const LoadingCustomExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Loading width={100} height={100} />
    </RenderComponentWithSnippet>
  );
};
