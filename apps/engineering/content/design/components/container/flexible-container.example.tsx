"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { FlexibleContainer } from "@unkey/ui";

export const DefaultExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col">
          <FlexibleContainer className="bg-accent-6">
            <div className="min-w-full min-h-full bg-accent-9">
              Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien
              vitae pellentesque sem placerat.
            </div>
          </FlexibleContainer>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

export const SmallExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col">
          <FlexibleContainer className="bg-accent-6" padding="small">
            <div className="min-w-full min-h-full bg-accent-9">
              Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien
              vitae pellentesque sem placerat.
            </div>
          </FlexibleContainer>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

export const MediumExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col">
          <FlexibleContainer className="bg-accent-6" padding="medium">
            <div className="min-w-full min-h-full bg-accent-9">
              Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien
              vitae pellentesque sem placerat.
            </div>
          </FlexibleContainer>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

export const LargeExample = () => {
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col">
          <FlexibleContainer className="bg-accent-6" padding="large">
            <div className="min-w-full min-h-full bg-accent-9">
              Lorem ipsum dolor sit amet consectetur adipiscing elit. Quisque faucibus ex sapien
              vitae pellentesque sem placerat.
            </div>
          </FlexibleContainer>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

export const CustomExample = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-row gap-2 w-full h-full">
        <div className="flex flex-col gap-2 w-full">
          <FlexibleContainer
            padding="small"
            justify="start"
            align="start"
            className="border bg-accent-6 flex-row"
          >
            <div className="w-fit h-44 bg-accent-9 p-2">Justify Start Align Start</div>
          </FlexibleContainer>
          <FlexibleContainer
            padding="small"
            justify="center"
            align="center"
            className="border bg-accent-6 flex-row"
          >
            <div className="w-fit h-44 bg-accent-9 p-2">Justify Center Align Center</div>
          </FlexibleContainer>
          <FlexibleContainer
            padding="small"
            justify="end"
            align="end"
            className="border bg-accent-6 flex-col"
          >
            <div className="w-fit h-44 bg-accent-9 p-2">Justify End Align End</div>
          </FlexibleContainer>
        </div>
        {/* Needed to get height to work properly in RenderComponentWithSnippet */}
        <div className="flex flex-col gap-2">
          <div className="w-fit bg-accent-9 p-2">Outside Container</div>
          <div className="w-fit bg-accent-9 p-2">Outside Container</div>
          <div className="w-fit bg-accent-9 p-2">Outside Container</div>
          <div className="w-fit bg-accent-9 p-2">Outside Container</div>
          <div className="w-fit bg-accent-9 p-2">Outside Container</div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
