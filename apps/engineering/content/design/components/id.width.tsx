import { RenderComponentWithSnippet } from "@/app/components/render";
import { Id } from "@unkey/ui";

export const WidthExample: React.FC = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} truncate={20} />`}
  >
    <Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} truncate={20} />
  </RenderComponentWithSnippet>
);
