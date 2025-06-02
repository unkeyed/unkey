"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button, DefaultDialogContentArea, DefaultDialogFooter, DialogContainer } from "@unkey/ui";

export const Default = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-row justify-center gap-8">
        <div className="flex gap-2">
          <span>Basic usage:</span>
          <DialogContainer isOpen={true} onOpenChange={() => {}} title="Dialog Title">
            <DefaultDialogContentArea>
              <p>Dialog Content</p>
            </DefaultDialogContentArea>
            <DefaultDialogFooter>
              <Button>Close</Button>
            </DefaultDialogFooter>
          </DialogContainer>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
