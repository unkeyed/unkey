import { BookBookmark } from "@unkey/icons";
import { Empty, buttonVariants } from "@unkey/ui";

export const EmptyAuditLogs = () => {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-100 flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No Audit Logs Found</Empty.Title>
        <Empty.Description className="text-left">
          There are no audit logs matching your filters. Adjust your search criteria or check back
          later.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-start">
          <a
            href="https://www.unkey.com/docs/audit-log/introduction"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "outline" })}
          >
            <BookBookmark />
            Learn about Audit Logs
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
