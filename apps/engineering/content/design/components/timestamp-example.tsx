import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { TimestampInfo } from "@unkey/ui";

export const TimestampExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Row>
      <TimestampInfo value={1745352664}/>
    </Row>
  </RenderComponentWithSnippet>
);

