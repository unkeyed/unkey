"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";

export const LoadingExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button loading>Update Role</Button>
      <Button loading variant="primary">
        Update Role
      </Button>
      <Button loading variant="destructive">
        Update Role
      </Button>
      <Button loading variant="ghost">
        Update Role
      </Button>
    </Row>
  </RenderComponentWithSnippet>
);
