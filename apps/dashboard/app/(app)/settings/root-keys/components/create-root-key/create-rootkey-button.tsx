"use client";
import { SecretKey } from "@/app/(app)/apis/[apiId]/_components/create-key/components/secret-key";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ArrowRight, Check, CircleInfo, Key2, Plus } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";
import {
  Button,
  Code,
  CopyButton,
  Dialog,
  DialogContent,
  FormInput,
  InfoTooltip,
  VisibleButton,
  toast,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useCallback, useMemo, useRef, useState } from "react";
import { PermissionBadgeList } from "./components/permission-badge-list";
import { PermissionSheet } from "./components/permission-sheet";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const DEFAULT_LIMIT = 12;

type Props = {
  className?: string;
} & React.ComponentProps<typeof Button>;

export const CreateRootKeyButton = ({ className, ...props }: Props) => {
  const UNNAMED_KEY = "Unnamed";
  const trpcUtils = trpc.useUtils();
  const [name, setName] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<
    "close" | "create-another" | "go-to-details" | null
  >(null);
  const dividerRef = useRef<HTMLDivElement>(null);
  const [selectedPermissions, setSelectedPermissions] = useState<UnkeyPermission[]>([]);

  const router = useRouter();

  const handleCloseAttempt = (action: "close" | "create-another" | "go-to-details" = "close") => {
    setPendingAction(action);
    setIsConfirmOpen(true);
  };

  const handleConfirmClose = () => {
    if (!pendingAction) {
      console.error("No pending action when confirming close");
      return;
    }

    setIsConfirmOpen(false);

    try {
      // Always close the dialog first
      setIsOpen(false);
      key.reset();
      setSelectedPermissions([]);
      setName("");

      // Then execute the specific action
      switch (pendingAction) {
        case "create-another":
          // Reset form for creating another key
          setIsOpen(true);
          break;

        case "go-to-details":
          router.push(`/settings/root-keys/${key.data?.keyId}`);
          break;

        default:
          // Dialog already closed, nothing more to do

          router.push("/settings/root-keys");
          break;
      }
    } catch (error) {
      console.error("Error executing pending action:", error);
      toast.error("Action Failed", {
        description: "An unexpected error occurred. Please try again.",
      });
    } finally {
      setPendingAction(null);
    }
  };

  const handleDialogOpenChange = (open: boolean) => {
    if (!open) {
      handleCloseAttempt("close");
    }
  };
  const {
    data: apisData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit: DEFAULT_LIMIT },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  const allApis = useMemo(() => {
    if (!apisData?.pages) {
      return [];
    }
    return apisData.pages.flatMap((page) => {
      return page.apiList.map((api) => ({
        id: api.id,
        name: api.name,
      }));
    });
  }, [apisData]);

  const key = trpc.rootKey.create.useMutation({
    onSuccess() {
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error(err);
      toast.error(err.message);
    },
  });

  function fetchMoreApis() {
    if (hasNextPage) {
      fetchNextPage();
    }
  }
  const snippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.createKey' \\
    -H 'Authorization: Bearer ${key.data?.key}' \\
    -H 'Content-Type: application/json' \\
    -d '{
      "prefix": "hello",
      "apiId": "<API_ID>"
    }'`;

  const split = key.data?.key?.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  const handlePermissionChange = useCallback((permissions: UnkeyPermission[]) => {
    setSelectedPermissions(permissions);
  }, []);

  return (
    <>
      <Button
        {...props}
        title="New root key"
        onClick={() => setIsOpen(true)}
        variant="primary"
        size="md"
        className={cn("rounded-lg", className)}
      >
        <Plus />
        New root key
      </Button>
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create new root key"
        contentClassName="p-0 mb-0 gap-0"
        className="max-w-[460px]"
        subTitle="Define a new root key and assign permissions"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              variant="primary"
              size="xlg"
              className="w-full rounded-lg"
              disabled={selectedPermissions.length === 0}
              onClick={() => {
                key.mutate({
                  name: name && name.length > 0 ? name : undefined,
                  permissions: selectedPermissions,
                });
              }}
            >
              Create root key
            </Button>
            <div className="text-gray-9 text-xs">This root key will be created immediately</div>
          </div>
        }
      >
        <div className="flex flex-col p-6 gap-4">
          <div className="flex flex-col">
            <FormInput
              name="name"
              label="Name"
              description="Give your key a name, this is not customer facing."
              placeholder="e.g. Vercel Production"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label className="text-[13px] font-regular text-gray-10">Permissions</Label>
            <PermissionSheet
              selectedPermissions={selectedPermissions}
              apis={allApis}
              onChange={handlePermissionChange}
              loadMore={fetchMoreApis}
              hasNextPage={hasNextPage}
              isFetchingNextPage={isFetchingNextPage}
            >
              <Button type="button" variant="outline" size="md" className="w-fit rounded-lg pl-3">
                Select Permissions...
              </Button>
            </PermissionSheet>
          </div>
        </div>
        <ScrollArea className="w-full overflow-y-auto pt-0 mb-4">
          <div className="flex flex-col px-6 py-0 gap-3">
            <PermissionBadgeList
              selectedPermissions={selectedPermissions}
              apiId={"workspace"}
              title="Selected from"
              name="Workspace"
              expandCount={3}
              removePermission={(permission) =>
                handlePermissionChange(selectedPermissions.filter((p) => p !== permission))
              }
            />
            {allApis.map((api) => (
              <PermissionBadgeList
                key={api.id}
                selectedPermissions={selectedPermissions}
                apiId={api.id}
                title="from"
                name={api.name}
                expandCount={3}
                removePermission={(permission) =>
                  handlePermissionChange(selectedPermissions.filter((p) => p !== permission))
                }
              />
            ))}
          </div>
        </ScrollArea>
      </DynamicDialogContainer>
      <Dialog open={!!key.data?.key} onOpenChange={handleDialogOpenChange}>
        <DialogContent
          className="drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0 min-w-[760px] max-h-[90vh] overflow-y-auto"
          showCloseWarning
          onAttemptClose={() => handleCloseAttempt("close")}
        >
          <>
            <div className="bg-grayA-2 py-10 flex flex-col items-center justify-center w-full px-[120px]">
              <div className="py-4 mt-[30px]">
                <div className="flex gap-4">
                  <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
                  <div className="border border-grayA-4 rounded-[10px] size-14" />
                  <div className="border border-grayA-4 rounded-[10px] size-14 flex items-center justify-center relative">
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 top-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 top-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 bottom-0" />
                    <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 bottom-0" />
                    <Key2 size="2xl-thin" />
                    <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                      <Check size="sm-bold" />
                    </div>
                  </div>
                  <div className="border border-grayA-4 rounded-[10px] size-14" />
                  <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
                </div>
              </div>
              <div className="mt-5 flex flex-col gap-2 items-center">
                <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
                  Root Key Created
                </div>
                <div
                  className="text-gray-10 text-[13px] leading-[24px] text-center"
                  ref={dividerRef}
                >
                  You've successfully generated a new Root key.
                </div>
              </div>
              <div className="p-1 w-full my-8">
                <div className="h-[1px] bg-grayA-3 w-full" />
              </div>
              <div className="flex flex-col gap-2 items-start w-full">
                <div className="text-gray-12 text-sm font-semibold">Key Details</div>
                <div className="bg-white dark:bg-black border rounded-xl border-grayA-5 px-6 w-full">
                  <div className="flex gap-6 items-center">
                    <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded ">
                      <Key2 size="sm-regular" />
                    </div>
                    <div className="flex flex-col gap-1 py-6">
                      <div className="text-accent-12 text-xs font-mono">{key.data?.keyId}</div>
                      <InfoTooltip
                        content={name ?? UNNAMED_KEY}
                        position={{ side: "bottom", align: "center" }}
                        asChild
                        disabled={!name}
                        variant="inverted"
                      >
                        <div className="text-accent-9 text-xs max-w-[160px] truncate">
                          {name ?? UNNAMED_KEY}
                        </div>
                      </InfoTooltip>
                    </div>
                    <Button
                      variant="outline"
                      className="ml-auto font-medium text-[13px] text-gray-12"
                      onClick={() => handleCloseAttempt("go-to-details")}
                    >
                      See key details <ArrowRight size="sm-regular" />
                    </Button>
                  </div>
                </div>
              </div>
              <div className="flex flex-col gap-2 items-start w-full mt-6">
                <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
                <SecretKey
                  value={key.data?.key ?? ""}
                  title="API Key"
                  className="bg-white dark:bg-black "
                />
                <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
                  <CircleInfo className="text-accent-9" size="sm-regular" />
                  <span>
                    Copy and save this key secret as it won't be shown again.{" "}
                    <a
                      href="https://www.unkey.com/docs/security/recovering-keys"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-info-11 hover:underline"
                    >
                      Learn more
                    </a>
                  </span>
                </div>
              </div>
              <div className="flex flex-col gap-2 items-start w-full mt-8">
                <div className="text-gray-12 text-sm font-semibold">Try It Out</div>
                <Code
                  visibleButton={
                    <VisibleButton
                      isVisible={showKeyInSnippet}
                      setIsVisible={setShowKeyInSnippet}
                    />
                  }
                  copyButton={<CopyButton value={snippet} />}
                >
                  {showKeyInSnippet ? snippet : snippet.replace(key.data?.key ?? "", maskedKey)}
                </Code>
              </div>
              <div className="mt-6">
                <div className="mt-4 text-center text-gray-10 text-xs leading-6">
                  All set! You can now create another key or explore the docs to learn more
                </div>
              </div>
            </div>
            <ConfirmPopover
              isOpen={isConfirmOpen}
              onOpenChange={setIsConfirmOpen}
              onConfirm={handleConfirmClose}
              triggerRef={dividerRef}
              title="You won't see this secret key again!"
              description="Make sure to copy your secret key before closing. It cannot be retrieved later."
              confirmButtonText="Close anyway"
              cancelButtonText="Dismiss"
              variant="warning"
              popoverProps={{
                side: "right",
                align: "end",
                sideOffset: 5,
                alignOffset: 30,
                onOpenAutoFocus: (e) => e.preventDefault(),
              }}
            />
          </>
        </DialogContent>
      </Dialog>
    </>
  );
};
