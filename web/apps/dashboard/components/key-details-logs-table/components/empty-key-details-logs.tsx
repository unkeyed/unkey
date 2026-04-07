import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

export const EmptyKeyDetailsLogs = () => {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-100 flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>Key Verification Logs</Empty.Title>
        <Empty.Description className="text-left">
          No verification logs found for this key. When this API key is used, details about each
          verification attempt will appear here.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-center md:justify-start">
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button size="md">
              <BookBookmark />
              Documentation
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
