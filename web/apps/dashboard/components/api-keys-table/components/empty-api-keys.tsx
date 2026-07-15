import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

export const EmptyApiKeys = () => {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-100 flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No API Keys Found</Empty.Title>
        <Empty.Description className="text-left">
          There are no API keys associated with this service yet. Create your first API key to get
          started.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-start">
          <Button
            size="md"
            render={
              // biome-ignore lint/a11y/useAnchorContent: link content is supplied by the Button's children via Base UI's render prop
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            <BookBookmark />
            Learn about Keys
          </Button>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
