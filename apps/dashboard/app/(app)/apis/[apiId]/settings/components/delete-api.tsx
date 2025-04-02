"use client";
import { Loading } from "@/components/dashboard/loading";
import { SettingCard } from "@/components/settings-card";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Lock, Trash } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useEffect, useState } from "react";
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

  useEffect(() => {
    if (!open) {
      form.reset();
    }
  }, [open]);

  const intent =
    keys > 0 ? `delete this api and ${keys} key${keys > 1 ? "s" : ""}` : "delete this api";

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

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

  const isValid = form.watch("intent") === intent && form.watch("name") === api.name;

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteApi.mutate({ apiId: api.id });
  }

  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <SettingCard
      className="py-5"
      title={
        <div className=" flex items-center justify-start gap-2.5">
          <Trash size="xl-medium" className="h-full text-error-9" />
          <span className="text-sm font-medium text-accent-12">Delete API</span>
          {api.deleteProtection && (
            <StatusBadge variant={"locked"} text={"Locked"} icon={<Lock size="sm-thin" />} />
          )}
        </div>
      }
      description={
        <div className="font-normal text-[13px]">
          Disabling this allows the API, along with all keys and data, <br />
          to be deleted by any team member.
        </div>
      }
      border="bottom"
    >
      <AlertDialog open={open} onOpenChange={handleDialogOpenChange}>
        <AlertDialogTrigger asChild>
          <div className="flex items-center justify-end w-full gap-2">
            <Button
              size="lg"
              disabled={!!api.deleteProtection}
              className="rounded-lg justify-end items-end px-3 text-error-9 font-medium text-[13px] leading-6"
              variant="outline"
              onClick={() => setOpen(!open)}
            >
              Delete API...
            </Button>
          </div>
        </AlertDialogTrigger>
        <AlertDialogContent className="w-[480px] rounded-2xl border border-grayA-4 bg-gray-1 shadow-lg m-0 p-0 overflow-hidden">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <AlertDialogTitle>
                <div className="flex flex-row justify-between w-full h-16 px-6 py-4 items-centerw-full">
                  <div className="flex font-medium leading-8 text-md whitespace-nowrap">
                    Delete API
                  </div>
                  <div className="flex justify-end w-full">
                    <AlertDialogCancel className="text-gray-11">X</AlertDialogCancel>
                  </div>
                </div>
              </AlertDialogTitle>
              <AlertDialogDescription>
                <div className="flex flex-col gap-2 bg-grayA-2">
                  <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem className="flex flex-col px-5 pt-6">
                        <AlertDialogDescription className="text-sm font-normal leading-6 text-gray-11">
                          Warning: Deleting this API will delete all keys and data associated with
                          it. This action cannot be undone. Any tracking, enforcement, and
                          historical insights tied to this API will be permanently lost.
                        </AlertDialogDescription>
                        <div className="pt-4 text-sm font-normal leading-6 text-gray-11">
                          Type <span className="font-medium text-gray-12">{api.name}</span> name to
                          confirm
                        </div>
                        <FormControl>
                          <Input {...field} autoComplete="off" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="intent"
                    render={({ field }) => (
                      <FormItem className="flex flex-col px-5 pt-2 pb-6">
                        <FormLabel className="pt-4 text-sm font-normal leading-6 text-gray-11">
                          To verify, type{" "}
                          <span className="py-0 my-0 font-medium text-gray-12">{intent}</span>{" "}
                          below:
                        </FormLabel>
                        <FormControl>
                          <Input {...field} autoComplete="off" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </AlertDialogDescription>
              <div className="flex flex-col gap-2 px-5 py-4 h-[112px] border-t border-grayA-4 overflow-hidden items-center justify-center">
                <Button
                  type="submit"
                  disabled={!isValid || deleteApi.isLoading}
                  loading={deleteApi.isLoading}
                  className={cn(
                    "rounded-lg text-white bg-error-9 font-medium text-[13px] leading-6 w-full border-grayA-3 h-10",
                    isValid ? "bg-error-9" : "disabled:bg-error-6",
                  )}
                >
                  {deleteApi.isLoading ? <Loading /> : "Delete API"}
                </Button>
                <div className="font-normal text-[12px] text-gray-9">
                  This action cannot be undone - proceed with caution
                </div>
              </div>
            </form>
          </Form>
        </AlertDialogContent>
      </AlertDialog>
    </SettingCard>
  );
};
