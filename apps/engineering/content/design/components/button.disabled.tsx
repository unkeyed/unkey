"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";

export const DisabledExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button disabled>Update Role</Button>
      <Button disabled variant="primary">
        Update Role
      </Button>
      <Button disabled variant="destructive">
        Update Role
      </Button>
      <Button disabled variant="ghost">
        Update Role
      </Button>
    </Row>
  </RenderComponentWithSnippet>
);
