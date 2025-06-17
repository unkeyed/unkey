"use client";
import { toast } from "@/components/ui/toaster";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Lock } from "@unkey/icons";
import { Button, DialogContainer, Input, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "../actions";
import { StatusBadge } from "./status-badge";

type Props = {
  keys: number;
  api: {
    id: string;
    workspaceId: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteApi: React.FC<Props> = ({ api, keys }) => {
  const [open, setOpen] = useState(false);

  const intent =
    keys > 0 ? `delete this api and ${keys} key${keys > 1 ? "s" : ""}` : "delete this api";

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  type FormValues = z.infer<typeof formSchema>;

  const {
    register,
    handleSubmit,
    watch,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });
  const isValid = watch("name") === api.name && watch("intent") === intent;
  const router = useRouter();

  const deleteApi = trpc.api.delete.useMutation({
    async onSuccess() {
      toast.message("API Deleted", {
        description: `Your API and ${formatNumber(keys)} keys have been deleted.`,
      });

      await revalidate();

      router.push("/apis");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteApi.mutate({ apiId: api.id });
  }

  return (
    <div>
      <SettingCard
        title={
          <div className="inline-flex gap-2">
            <span>Delete API </span>
            {api.deleteProtection && (
              <StatusBadge variant={"locked"} text={"Locked"} icon={<Lock size="sm-thin" />} />
            )}
          </div>
        }
        description={
          api.deleteProtection ? (
            <div className="font-normal text-[13px]">
              Permanently deletes this API, including all keys and data. This action is locked by
              the <span className="font-medium text-accent-12">Delete Protection</span> feature.
            </div>
          ) : (
            <div className="font-normal text-[13px] max-w-[380px]">
              <div className="font-normal text-[13px] max-w-[380px]">
                <div className="font-normal text-[13px] max-w-[380px]">
                  Permanently deletes this API, including all keys and data. This action cannot be
                  undone.
                </div>
              </div>
            </div>
          )
        }
        border="bottom"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="w-full flex justify-end">
          <Button
            variant="outline"
            color="danger"
            size="lg"
            disabled={api.deleteProtection === true}
            onClick={() => setOpen(true)}
          >
            Delete API
          </Button>
        </div>
      </SettingCard>
      <DialogContainer
        isOpen={open}
        onOpenChange={setOpen}
        title="Delete API"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-api-form" // Connect to form ID
              variant="primary"
              color="danger"
              size="xlg"
              disabled={api.deleteProtection || !isValid || deleteApi.isLoading || isSubmitting}
              loading={deleteApi.isLoading || isSubmitting}
              className="w-full"
            >
              Delete API
            </Button>
            <div className="text-gray-9 text-xs">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <p className="text-gray-11 text-[13px]">
          <span className="font-medium">Warning: </span>
          Deleting this API will delete all keys and data associated with it. This action cannot be
          undone. Any tracking, enforcement, and historical insights tied to this API will be
          permanently lost.
        </p>
        <form id="delete-api-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="space-y-1">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium">{api.name}</span> to confirm
            </p>
            <Input {...register("name")} placeholder={`Enter "${api.name}" to confirm`} />
          </div>
          <div className="space-y-1 mt-6">
            <p className="text-gray-11 text-[13px]">
              To verify, type <span className="text-gray-12 font-medium">{intent}</span> to confirm
            </p>
            <Input {...register("intent")} placeholder={`Enter "${intent}" to confirm`} />
          </div>
        </form>
      </DialogContainer>
    </div>
  );
};
