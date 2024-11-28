"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";

export const VariantExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button>Update Role</Button>
      <Button variant="primary">Update Role</Button>
      <Button variant="destructive">Delete Role</Button>
      <Button variant="ghost">Filter</Button>
    </Row>
  </RenderComponentWithSnippet>
);
