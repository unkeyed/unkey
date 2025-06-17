"use client";

import { revalidateTag } from "@/app/actions";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { Clone } from "@unkey/icons";
import { Button, Input, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DeleteNamespaceDialog } from "../../_components/namespace-delete-dialog";
import { SettingsClientSkeleton } from "./skeleton";

type Props = {
  namespaceId: string;
};

export const SettingsClient = ({ namespaceId }: Props) => {
  const [isNamespaceNameDeleteModalOpen, setIsNamespaceNameDeleteModalOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const router = useRouter();

  const { data, isLoading } = trpc.ratelimit.namespace.queryDetails.useQuery({
    namespaceId,
    includeOverrides: false,
  });

  const [namespaceName, setNamespaceName] = useState(data?.namespace.name || "");

  // Update local state when data loads
  if (data?.namespace.name && namespaceName !== data.namespace.name && !isUpdating) {
    setNamespaceName(data.namespace.name);
  }

  const updateNameMutation = trpc.ratelimit.namespace.update.name.useMutation({
    onSuccess() {
      toast.success("Your namespace name has been renamed!");
      revalidateTag(tags.namespace(namespaceId));
      router.refresh();
      setIsUpdating(false);
    },
    onError(err) {
      toast.error("Failed to update namespace name", {
        description: err.message,
      });
      setIsUpdating(false);
    },
  });

  const handleUpdateName = async () => {
    if (!data?.namespace) {
      return;
    }

    if (namespaceName === data.namespace.name || !namespaceName) {
      return toast.error("Please provide a different name before saving.");
    }

    setIsUpdating(true);
    await updateNameMutation.mutateAsync({
      name: namespaceName,
      namespaceId: data.namespace.id,
    });
  };

  if (!data || isLoading) {
    return <SettingsClientSkeleton />;
  }

  const { namespace } = data;

  return (
    <>
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Ratelimit Settings
          </div>
          <div className="flex flex-col w-full gap-6">
            <div>
              <SettingCard
                title="Namespace name"
                description={
                  <div>
                    Used in API calls. Changing this may cause rate limit
                    <br /> requests to be rejected.
                  </div>
                }
                border="top"
                className="border-b-1"
                contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
              >
                <div className="flex flex-row justify-end items-center gap-x-2 mt-2 h-9">
                  <Input
                    placeholder="Namespace name"
                    value={namespaceName}
                    className="min-w-[16rem] items-end h-9"
                    onChange={(e) => setNamespaceName(e.target.value)}
                  />
                  <Button
                    className="h-full px-3.5 rounded-lg"
                    size="lg"
                    variant="primary"
                    onClick={handleUpdateName}
                    loading={isUpdating}
                    disabled={isUpdating || namespaceName === namespace.name || !namespaceName}
                  >
                    Save
                  </Button>
                </div>
              </SettingCard>
              <SettingCard
                title="Namespace ID"
                description="An identifier for the namespace, used in some API calls."
                border="bottom"
                contentWidth="w-full lg:w-[320px] h-full justify-end items-end"
              >
                <div className="flex flex-row justify-end items-center gap-x-2 mt-1 w-full h-9">
                  <Input
                    readOnly
                    defaultValue={namespace.id}
                    placeholder="Namespace name"
                    className="w-full focus:ring-0 focus:ring-offset-0 h-full"
                    rightIcon={
                      <button
                        type="button"
                        onClick={() => {
                          navigator.clipboard.writeText(namespace.id);
                          toast.success("Copied to clipboard", {
                            description: namespace.id,
                          });
                        }}
                      >
                        <Clone size="md-regular" />
                      </button>
                    }
                  />
                </div>
              </SettingCard>
            </div>

            <div className="w-full">
              <SettingCard
                title="Delete ratelimit"
                description={
                  <>
                    Deletes this namespace along with all associated
                    <br /> identifiers and data. This action cannot be undone.
                  </>
                }
                border="both"
                contentWidth="w-full lg:w-[320px] h-full justify-end items-end"
              >
                <div className="w-full flex justify-end lg:mt-3">
                  <Button
                    className="w-fit px-3.5 rounded-lg"
                    variant="outline"
                    color="danger"
                    size="lg"
                    onClick={() => setIsNamespaceNameDeleteModalOpen(true)}
                  >
                    Delete Namespace
                  </Button>
                </div>
              </SettingCard>
            </div>
          </div>
        </div>
      </div>
      <DeleteNamespaceDialog
        namespace={namespace}
        onOpenChange={setIsNamespaceNameDeleteModalOpen}
        isModalOpen={isNamespaceNameDeleteModalOpen}
      />
    </>
  );
};
