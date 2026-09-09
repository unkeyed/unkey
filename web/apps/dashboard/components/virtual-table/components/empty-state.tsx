import { Button, Empty } from "@unkey/ui";
import { IconBookBookmarkOutline18 } from "nucleo-ui-outline-18";

export const EmptyState = ({ content }: { content?: React.ReactNode }) => (
  <div className="flex-1 flex items-center justify-center">
    {content || (
      <div className="w-full flex justify-center items-center h-full">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>Nothing here yet</Empty.Title>
          <Empty.Description className="text-left">
            Ready to get started? Check our documentation for a step-by-step guide.
          </Empty.Description>
          <Empty.Actions className="mt-4 justify-start">
            <a href="https://www.unkey.com/docs" target="_blank" rel="noopener noreferrer">
              <Button>
                <IconBookBookmarkOutline18 />
                Documentation
              </Button>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    )}
  </div>
);
