"use client";

import { revalidateTag } from "@/app/actions";
import { SettingCard } from "@/components/settings-card";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { Clone } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DeleteNamespaceDialog } from "../../_components/namespace-delete-dialog";
import { Separator } from "@/components/ui/separator";

type Props = {
  namespace: {
    id: string;
    workspaceId: string;
    name: string;
  };
};
export const SettingsClient = ({ namespace }: Props) => {
  const [isNamespaceNameDeleteModalOpen, setIsNamespaceNameDeleteModalOpen] = useState(false);
  const [namespaceName, setNamespaceName] = useState(namespace.name);
  const [isUpdating, setIsUpdating] = useState(false);
  const router = useRouter();

  const updateNameMutation = trpc.ratelimit.namespace.update.name.useMutation({
    onSuccess() {
      toast.success("Your namespace name has been renamed!");
      revalidateTag(tags.namespace(namespace.id));
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
    if (namespaceName === namespace.name || !namespaceName) {
      return toast.error("Please provide a different name before saving.");
    }

    setIsUpdating(true);
    await updateNameMutation.mutateAsync({
      name: namespaceName,
      namespaceId: namespace.id,
    });
  };

  return (
    <>
      <div className="py-3 w-full flex items-center justify-center ">
        <div className="w-[760px] flex flex-col justify-center items-center gap-5">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Ratelimit Settings
          </div>

          <div className="w-full">
            <SettingCard
              title="Namespace name"
              description={
                <div>
                  Used in API calls. Changing this may cause rate limit
                  <br /> requests to be rejected.
                </div>
              }
              border="top"
            >
              <div className="flex gap-2 items-center justify-center w-full">
                <Input
                  placeholder="Namespace name"
                  className="h-9"
                  value={namespaceName}
                  onChange={(e) => setNamespaceName(e.target.value)}
                />
                <Button
                  size="lg"
                  className="rounded-lg"
                  onClick={handleUpdateName}
                  loading={isUpdating}
                  disabled={isUpdating || namespaceName === namespace.name || !namespaceName}
                >
                  Save
                </Button>
              </div>
            </SettingCard>
            <Separator className="bg-gray-4" orientation="horizontal" />
            <SettingCard
              title="Namespace ID"
              description="An identifier for the namespace, used in some API calls."
              border="bottom"
            >
              <Input
                readOnly
                disabled
                defaultValue={namespace.id}
                placeholder="Namespace name"
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
            </SettingCard>
          </div>

          <SettingCard
            title="Delete ratelimit"
            description={
              <>
                Deletes this namespace along with all associated
                <br /> identifiers and data. This action cannot be undone.
              </>
            }
            border="both"
          >
            <div className="w-full flex justify-end">
              <Button
                className="w-fit rounded-lg"
                variant="outline"
                color="danger"
                size="lg"
                onClick={() => setIsNamespaceNameDeleteModalOpen(true)}
              >
                Delete Namespace...
              </Button>
            </div>
          </SettingCard>
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
