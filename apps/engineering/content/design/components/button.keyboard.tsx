"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button } from "@unkey/ui";

export const KeyboardExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Button
        variant="default"
        keyboard={{
          display: "X",
          trigger: (e) => e.key === "x",
          callback: () => {
            alert("hello keyboard warrior");
          },
        }}
      >
        Search
      </Button>
      <Button
        variant="primary"
        keyboard={{
          display: "S",
          trigger: (e) => e.key === "s",
          callback: () => {
            alert("hello keyboard warrior");
          },
        }}
      >
        Search
      </Button>
      <Button
        variant="ghost"
        keyboard={{
          display: "F",
          trigger: (e) => e.key === "f",
          callback: () => {
            alert("hello keyboard warrior");
          },
        }}
      >
        Search
      </Button>
      <Button
        variant="destructive"
        keyboard={{
          display: "T",
          trigger: (e) => e.key === "t",
          callback: () => {
            alert("hello keyboard warrior");
          },
        }}
      >
        Search
      </Button>
    </Row>
  </RenderComponentWithSnippet>
);
