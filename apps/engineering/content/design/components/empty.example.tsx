import { RenderComponentWithSnippet } from "@/app/components/render";
import { BookBookmark, ShieldCheck } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { Empty } from "@unkey/ui";

export const EmptyExample: React.FC = () => (
  <RenderComponentWithSnippet>
    <Empty fill>
      <Empty.Icon>
        <ShieldCheck />
      </Empty.Icon>
      <Empty.Title>Example Title Text</Empty.Title>
      <Empty.Description>Example of Description Text.</Empty.Description>
      <Empty.Actions>
        <Button className="bg-gray-2">
          <BookBookmark /> Example action button with icon
        </Button>
      </Empty.Actions>
    </Empty>
  </RenderComponentWithSnippet>
);
