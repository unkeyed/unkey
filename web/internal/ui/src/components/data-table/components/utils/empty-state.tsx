import { BookBookmark } from "@unkey/icons";
import type React from "react";
import { Button } from "../../../buttons/button";
import { Empty } from "../../../empty";

export interface EmptyStateProps {
  content?: React.ReactNode;
}

/**
 * Empty state component for tables with no data
 */
export const EmptyState = ({ content }: EmptyStateProps) => {
  return (
    <div className="flex-1 flex items-center justify-center">
      {content ?? (
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Nothing here yet</Empty.Title>
            <Empty.Description className="text-left">
              Ready to get started? Check our documentation for a step-by-step guide.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <Button
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
                Documentation
              </Button>
            </Empty.Actions>
          </Empty>
        </div>
      )}
    </div>
  );
};
