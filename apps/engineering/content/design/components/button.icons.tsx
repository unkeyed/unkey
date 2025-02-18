"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";
import { ArrowLeft, ArrowRight, Settings2, Trash } from "lucide-react";

export const IconsExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button>
        <Settings2 />
        Filter
      </Button>
      <Button variant="primary">
        {" "}
        <ArrowLeft /> Update Role
        <ArrowRight />
      </Button>
      <Button variant="destructive">
        Update Role
        <Trash />
      </Button>
      <Button shape="square">
        <Settings2 />
      </Button>
    </Row>
  </RenderComponentWithSnippet>
);
