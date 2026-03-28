import { BookBookmark } from "@unkey/icons";
import { buttonVariants } from "../../../buttons/button";
import { Empty } from "../../../empty";

export function EmptyApiRequests() {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-100 flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>Key Verification Logs</Empty.Title>
        <Empty.Description className="text-left">
          No key verification data to show. Once requests are made with API keys, you'll see a
          summary of successful and failed verification attempts.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-center md:justify-start">
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ size: "md" })}
          >
            <BookBookmark />
            Documentation
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
