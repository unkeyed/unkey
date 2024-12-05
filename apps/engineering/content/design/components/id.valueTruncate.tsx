import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Id } from "@unkey/ui";

export const ValueTruncateExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} />
      <Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} truncate={6} />
      <Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} truncate={12} />
    </Row>
  </RenderComponentWithSnippet>
);
