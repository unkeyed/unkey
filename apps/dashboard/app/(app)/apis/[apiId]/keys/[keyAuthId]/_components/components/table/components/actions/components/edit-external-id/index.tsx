import { DialogContainer } from "@/components/dialog-container";
import { Combobox } from "@/components/ui/combobox";

import { CopyButton } from "@/components/dashboard/copy-button";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { User } from "@unkey/icons";
import { Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { useState } from "react";
import type { ActionComponentProps } from "../../keys-table-action.popover";
import { KeyInfo } from "../key-info";
import { useFetchIdentities } from "./hooks/use-fetch-identities";

type EditExternalIdProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export const EditExternalId = ({
  keyDetails,
  isOpen,
  onClose,
}: EditExternalIdProps): JSX.Element => {
  const [value, setValue] = useState<string>("");
  const { identities } = useFetchIdentities();

  return (
    <DialogContainer
      isOpen={isOpen}
      subTitle="Update the ID of the user/workspace in your system for key attribution"
      onOpenChange={() => {
        onClose();
      }}
      title="Edit external ID"
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form="edit-remaining-uses-form"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
          >
            Update external ID
          </Button>
          <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
        </div>
      }
    >
      <KeyInfo keyDetails={keyDetails} />
      <div className="py-1 my-2">
        <div className="h-[1px] bg-grayA-3 w-full" />
      </div>
      <Combobox
        options={identities.map((identity) => ({
          label: (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="flex w-full text-accent-8 text-xs gap-1.5 py-0.5 items-center">
                    <div className="flex items-center justify-center gap-2">
                      <div className="border rounded-full flex items-center justify-center border-grayA-6 size-5">
                        <User size="sm-regular" className="text-grayA-11" />
                      </div>
                      <span className="text-accent-12 font-medium text-xs w-[120px] truncate">
                        {identity.id}
                      </span>
                    </div>
                    <span className="w-[200px] truncate text-accent-8">{identity.externalId}</span>
                  </div>
                </TooltipTrigger>
                <TooltipContent
                  side="right"
                  align="start"
                  sideOffset={17}
                  className="drop-shadow-2xl border border-gray-4 overflow-hidden rounded-[10px] p-0 bg-white dark:bg-black w-[320px] h-[300px]"
                >
                  <div className="flex flex-col h-full">
                    {/* Header */}
                    <div className="px-4 py-2 border-b border-grayA-4 text-gray-10 text-xs font-medium bg-grayA-2">
                      Metadata
                    </div>
                    {/* Content */}
                    <div className="px-2 py-2 flex-1 overflow-y-auto">
                      <div className="w-full bg-grayA-1 dark:bg-grayA-2 border rounded-lg border-grayA-5 overflow-hidden h-full">
                        <div className="flex items-start justify-between w-full gap-2 h-full">
                          {/* JSON Content */}
                          <div className="overflow-x-auto w-full min-w-0 p-3 h-full">
                            <pre className="whitespace-pre-wrap break-all text-xs text-gray-12 font-mono h-full overflow-y-auto">
                              {JSON.stringify(identity.meta, null, 2)}
                            </pre>
                          </div>
                          {/* Copy Button */}
                          <div className="p-2 flex-shrink-0">
                            <Button
                              variant="outline"
                              size="icon"
                              className="bg-white dark:bg-grayA-3 hover:bg-grayA-3 dark:hover:bg-grayA-4 shadow-sm"
                            >
                              <div className="flex items-center justify-center">
                                <CopyButton value={JSON.stringify(identity.meta, null, 2)} />
                              </div>
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ),
          value: identity.externalId,
        }))}
        value={value}
        onValueChange={setValue}
        placeholder={
          <div className="flex w-full text-grayA-8 text-xs gap-1.5 items-center">
            Select external ID
          </div>
        }
        searchPlaceholder="Search external ID..."
        emptyMessage="No external ID found."
        variant="default"
      />
    </DialogContainer>
  );
};
