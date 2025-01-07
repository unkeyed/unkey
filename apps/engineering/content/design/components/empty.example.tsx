"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";
import { Empty } from "@unkey/ui";
import { BookOpen, ShieldBan } from "lucide-react";

export const EmptyExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Empty fill={true}>
        <Empty.Icon>
          <ShieldBan />
        </Empty.Icon>
        <Empty.Title>Example Title Text</Empty.Title>
        <Empty.Description>Example of Description Text.</Empty.Description>
        <Empty.Action>
          <Button className="bg-gray-2">
            <BookOpen /> Example action button with icon
          </Button>
        </Empty.Action>
      </Empty>
    </Row>
  </RenderComponentWithSnippet>
);
