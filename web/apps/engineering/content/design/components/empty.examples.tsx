import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { BookBookmark } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";

export function EmptyExample() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Empty>
    <Empty.Icon />
    <Empty.Title>Example Title Text</Empty.Title>
    <Empty.Description>Example of Description Text.</Empty.Description>
    <Empty.Actions>
    <Button>
        <BookBookmark /> 
        Submit
    </Button>
    </Empty.Actions>
</Empty>`}
    >
      <Row>
        <Empty>
          <Empty.Icon />
          <Empty.Title>Example Title Text</Empty.Title>
          <Empty.Description>Example of Description Text.</Empty.Description>
          <Empty.Actions>
            <Button>
              <BookBookmark />
              Submit
            </Button>
          </Empty.Actions>
        </Empty>
      </Row>
    </RenderComponentWithSnippet>
  );
}
