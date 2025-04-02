"use client";
import { Loading } from "@/components/dashboard/loading";
import { SettingCard } from "@/components/settings-card";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { ShieldAlert, TriangleWarning2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "../actions";
import { StatusBadge } from "./status-badge";

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteProtection: React.FC<Props> = ({ api }) => {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) {
      form.reset();
    }
  }, [open]);
  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const updateDeleteProtection = trpc.api.updateDeleteProtection.useMutation({
    async onSuccess(_, { enabled }) {
      toast.message(
        `Delete protection for ${api.name} has been ${enabled ? "enabled" : "disabled"}`,
        {},
      );

      setOpen(false);
      await revalidate();

      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const isValid = form.watch("name") === api.name;

  async function onSubmit(_: z.infer<typeof formSchema>) {
    updateDeleteProtection.mutate({
      apiId: api.id,
      enabled: !api.deleteProtection,
    });
  }

  return (
    <SettingCard
      className="py-5 mt-5"
      title={
        <div className=" flex items-center justify-start gap-2.5">
          <ShieldAlert size="xl-medium" className="h-full text-warning-9" />
          <span className="text-sm font-medium text-accent-12">Delete Protection</span>{" "}
          <StatusBadge
            variant={api.deleteProtection ? "enabled" : "disabled"}
            text={api.deleteProtection ? "Enabled" : "Disabled"}
            icon={<TriangleWarning2 size="sm-thin" />}
          />
        </div>
      }
      description={
        <div className="font-normal text-[13px]">
          Disabling this allows the API, along with all keys and data, <br />
          to be deleted by any team member.
        </div>
      }
      border="top"
    >
      <AlertDialog open={open} onOpenChange={(o) => setOpen(o)}>
        <AlertDialogTrigger asChild>
          <div className="flex items-center justify-end w-full gap-2">
            {api.deleteProtection ? (
              <Button
                type="button"
                variant="outline"
                size="lg"
                className="rounded-lg text-warning-11 text-[13px] px-4"
              >
                Disable Delete Protection...
              </Button>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="lg"
                className="rounded-lg text-success-11 text-[13px]"
              >
                Enable Delete Protection...
              </Button>
            )}
          </div>
        </AlertDialogTrigger>
        <AlertDialogContent className="w-[480px] rounded-2xl border border-grayA-4 bg-gray-1 shadow-lg m-0 p-0 overflow-hidden">
          <Form {...form}>
            <form className="flex flex-col" onSubmit={form.handleSubmit(onSubmit)}>
              <div className="flex flex-row items-center justify-between w-full h-16 px-6 py-4">
                <div className="flex font-medium leading-8 text-md whitespace-nowrap">
                  {api.deleteProtection ? "Disable" : "Enable"} API Delete Protection
                </div>
                <div className="flex justify-end w-full">
                  <AlertDialogCancel className="text-gray-11">X</AlertDialogCancel>
                </div>
              </div>
              <div className="flex gap-2 bg-grayA-2 py-4 px-6 h-[192px]">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <div className="text-sm font-normal leading-6 text-gray-11">
                        Important: Enabling this prevents the API, along with all keys and data, to
                        be deleted by any non-admin team member. This setting can be disabled at any
                        time.
                      </div>
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
              </div>

              <div className="flex flex-col gap-2 px-5 py-4 h-[112px] border-t border-grayA-4 overflow-hidden items-center justify-center">
                <Button
                  type="submit"
                  disabled={!isValid || updateDeleteProtection.isLoading}
                  loading={updateDeleteProtection.isLoading}
                  className={cn(
                    "rounded-lg text-white bg-warning-9 font-medium text-[13px] leading-6 w-full border-grayA-3 h-10",
                    api.deleteProtection
                      ? isValid
                        ? "bg-warning-9"
                        : "disabled:bg-warning-6"
                      : isValid
                        ? "bg-success-9"
                        : "disabled:bg-success-6",
                  )}
                >
                  {updateDeleteProtection.isLoading ? (
                    <Loading />
                  ) : api.deleteProtection ? (
                    "Disable API Delete Protection"
                  ) : (
                    "Enable API Delete Protection"
                  )}
                </Button>
                <div className="font-normal text-[12px] text-gray-9">
                  This setting can be disabled at any time
                </div>
              </div>
            </form>
          </Form>
        </AlertDialogContent>
      </AlertDialog>
    </SettingCard>
  );
};
