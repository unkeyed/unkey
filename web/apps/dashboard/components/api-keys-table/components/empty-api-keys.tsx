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
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button size="md">
              <BookBookmark />
              Learn about Keys
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
