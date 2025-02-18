"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";

export const LoadingExample: React.FC = () => {
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    if (!loading) {
      return;
    }
    const id = setTimeout(() => setLoading(false), 5000);
    return () => {
      clearTimeout(id);
    };
  }, [loading]);
  return (
    <RenderComponentWithSnippet>
      <Row>
        <Button loading>Update Role</Button>
        <Button loading={loading} variant="primary" onClick={() => setLoading(true)}>
          Click Me
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
};
