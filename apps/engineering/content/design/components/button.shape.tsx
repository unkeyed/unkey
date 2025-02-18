"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";
import { Settings2 } from "lucide-react";

export const ShapeExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button shape="square">
        <Settings2 />
      </Button>
    </Row>
  </RenderComponentWithSnippet>
);
