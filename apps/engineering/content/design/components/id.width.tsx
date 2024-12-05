import { RenderComponentWithSnippet } from "@/app/components/render";
import { Id } from "@unkey/ui";

export const WidthExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <div className="w-full">
      <Id value={"api_zTLSnw1YTqUHRwJTgKWrg"} truncate={20} />
    </div>
  </RenderComponentWithSnippet>
);
