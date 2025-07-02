import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";

import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { InfoIcon } from "lucide-react";
export const OnHoverExample: React.FC = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<Row>
  <Tooltip>
    <TooltipTrigger>
      <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
        Bottom{" "}
        <span>
          <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
        </span>
      </p>
    </TooltipTrigger>
    <TooltipContent className="h-8" side="bottom">
      Content
    </TooltipContent>
  </Tooltip>
  <Tooltip>
    <TooltipTrigger>
      <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
        Top{" "}
        <span>
          <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
        </span>
      </p>
    </TooltipTrigger>
    <TooltipContent className="h-8" side="top">
      Content
    </TooltipContent>
  </Tooltip>
  <Tooltip>
    <TooltipTrigger>
      <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
        Right{" "}
        <span>
          <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
        </span>
      </p>
    </TooltipTrigger>
    <TooltipContent className="h-8" side="right">
      Content
    </TooltipContent>
  </Tooltip>
  <Tooltip>
    <TooltipTrigger>
      <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
        Left{" "}
        <span>
          <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
        </span>
      </p>
    </TooltipTrigger>
    <TooltipContent className="h-8" side="left">
      Content
    </TooltipContent>
  </Tooltip>
</Row>`}
  >
    <Row>
      <Tooltip>
        <TooltipTrigger>
          <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
            Bottom{" "}
            <span>
              <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
            </span>
          </p>
        </TooltipTrigger>
        <TooltipContent className="h-8" side="bottom">
          Content
        </TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger>
          <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
            Top{" "}
            <span>
              <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
            </span>
          </p>
        </TooltipTrigger>
        <TooltipContent className="h-8" side="top">
          Content
        </TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger>
          <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
            Right{" "}
            <span>
              <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
            </span>
          </p>
        </TooltipTrigger>
        <TooltipContent className="h-8" side="right">
          Content
        </TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger>
          <p className="inline-flex gap-4 border border-gray-2 px-3 py-1 rounded-lg ">
            Left{" "}
            <span>
              <InfoIcon size={14} className="text-gray-10 self-auto h-full" />
            </span>
          </p>
        </TooltipTrigger>
        <TooltipContent className="h-8" side="left">
          Content
        </TooltipContent>
      </Tooltip>
    </Row>
  </RenderComponentWithSnippet>
);
