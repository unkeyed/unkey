import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

export const EmptyRatelimitLogs = () => (
  <div className="w-full flex justify-center items-center h-full">
    <Empty className="w-[400px] flex items-start">
      <Empty.Icon className="w-auto" />
      <Empty.Title>Logs</Empty.Title>
      <Empty.Description className="text-left">
        No ratelimit logs yet. Once API requests start coming in, you'll see a detailed view of your
        rate limits, including passed and blocked requests, across your API endpoints.
      </Empty.Description>
      <Empty.Actions className="mt-4 justify-start">
        <a href="https://www.unkey.com/docs/introduction" target="_blank" rel="noopener noreferrer">
          <Button size="md">
            <BookBookmark />
            Documentation
          </Button>
        </a>
      </Empty.Actions>
    </Empty>
  </div>
);
