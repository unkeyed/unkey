"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Button, CopyButton, Empty, Input, SettingCard, toast } from "@unkey/ui";
import { useEffect, useState } from "react";
import { CreateNamespaceButton } from "../../../_components/create-namespace-button";
import { DeleteNamespaceDialog } from "../../_components/namespace-delete-dialog";
import { SettingsClientSkeleton } from "./skeleton";

type Props = {
  namespaceId: string;
};

export const SettingsClient = ({ namespaceId }: Props) => {
  const [isNamespaceNameDeleteModalOpen, setIsNamespaceNameDeleteModalOpen] = useState(false);

  const { data, isLoading } = useLiveQuery((q) =>
    q
      .from({ namespace: collection.ratelimitNamespaces })
      .where(({ namespace }) => eq(namespace.id, namespaceId)),
  );

  const namespace = data.at(0);

  const [namespaceName, setNamespaceName] = useState<string | null>(null);
  useEffect(() => {
    if (namespaceName === null && namespace) {
      setNamespaceName(namespace.name);
    }
  }, [namespace, namespaceName]);

  if (isLoading) {
    return <SettingsClientSkeleton />;
  }

  if (!namespace) {
    return (
      <Empty>
        <Empty.Icon />
        <Empty.Title>404</Empty.Title>
        <Empty.Description>This namespace does not exist</Empty.Description>
        <Empty.Actions>
          <CreateNamespaceButton />
        </Empty.Actions>
      </Empty>
    );
  }

  const handleUpdateName = async () => {
    if (!namespace) {
      return;
    }

    if (namespaceName === namespace.name || !namespaceName) {
      return toast.error("Please provide a different name before saving.");
    }
    let error = "";
    collection.ratelimitNamespaces.forEach((ns) => {
      if (ns.id !== namespaceId && namespaceName === ns.name) {
        error = "Another namespace already has this name";
        return;
      }
    });
    if (error) {
      return toast.error(error);
    }

    collection.ratelimitNamespaces.update(namespace.id, (draft) => {
      draft.name = namespaceName;
    });
  };

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
                className="border-b"
                contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
              >
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    handleUpdateName();
                  }}
                  className="flex flex-row justify-end items-center gap-x-2 mt-2 h-9"
                >
                  <Input
                    placeholder="Namespace name"
                    value={namespaceName ?? ""}
                    className="min-w-[16rem] items-end h-9"
                    onChange={(e) => setNamespaceName(e.target.value)}
                  />
                  <Button
                    type="submit"
                    className="h-full px-3.5 rounded-lg"
                    size="lg"
                    variant="primary"
                    disabled={namespaceName === namespace.name || !namespaceName}
                  >
                    Save
                  </Button>
                </form>
              </SettingCard>
              <SettingCard
                title="Namespace ID"
                description="An identifier for the namespace, used in some API calls."
                border="bottom"
                contentWidth="w-full lg:w-[320px] h-full justify-end items-end"
              >
                <div className="flex flex-row justify-end items-center pl-2 pr-2 py-2 w-full border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black rounded-lg min-w-[327px]">
                  <pre className="flex-1 text-xs text-left overflow-x-auto">
                    <code>{namespace.id}</code>
                  </pre>
                  <CopyButton value={namespace.id} variant="ghost" size="sm" />
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
