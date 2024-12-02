"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Id } from "@unkey/ui";

export const CopyExample: React.FC = () => (
    <RenderComponentWithSnippet>
        <Row>
            <Id value={"Hover and click to copy text"} />
            <Id value={"Hover and click to copy text"} truncate={12} />
            <input className="w-fit bg-base-12 border border-black border-2 text-black rounded-lg font-normal h-8 px-3 py-1 text-sm" placeholder="Paste here:"/>
        </Row>
    </RenderComponentWithSnippet>
);
