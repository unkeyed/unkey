"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import type { Secret } from "@/lib/db";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff, Loader2, MoreHorizontal, Settings, VenetianMask } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

type Props = {
  secrets: Secret[];
};

export const Secrets: React.FC<Props> = ({ secrets }) => {
  if (secrets.length === 0) {
    return (
      <EmptyPlaceholder>
        <EmptyPlaceholder.Icon>
          <VenetianMask />
        </EmptyPlaceholder.Icon>
        <EmptyPlaceholder.Title>No secrets found</EmptyPlaceholder.Title>
      </EmptyPlaceholder>
    );
  }

  return (
    <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
      {secrets.map((s) => (
        <Row key={s.id} secret={s} />
      ))}
    </ul>
  );
};

const Row: React.FC<{ secret: Secret }> = ({ secret }) => {
  const [isEditMode, setIsEditMode] = useState(false);

  const formSchema = z.object({
    name: z.string(),
    value: z.string(),
    comment: z.string().optional(),
  });

  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    defaultValues: {
      name: secret.name,
      comment: secret.comment ?? "",
    },
  });

  const decrypt = trpc.secrets.decrypt.useMutation({
    onSuccess: ({ value }) => {
      form.setValue("value", value);
    },
  });
  useEffect(() => {
    if (isEditMode && !decrypt.data) {
      decrypt.mutate({ secretId: secret.id });
    }
  }, [isEditMode]);

  const update = trpc.secrets.update.useMutation({
    onSuccess() {
      toast.success("Secret updated");
      router.refresh();
    },
    onError(err) {
      toast.error("An error occured", {
        description: err.message,
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    update.mutate({
      secretId: secret.id,
      name: values.name,
      value: values.value,
      comment: values.comment === "" ? null : values.comment,
    });
  }

  return (
    <li
      key={secret.id}
      className={cn("px-4 py-2", {
        "bg-background-subtle": isEditMode,
      })}
    >
      <div className="grid items-center grid-cols-12 ">
        <div className="flex flex-col items-start col-span-5">
          <span className="text-sm text-content">{secret.name}</span>
          <pre className="text-xs text-content-subtle">{secret.id}</pre>
        </div>

        <div className="col-span-6">
          <Value secretId={secret.id} />
        </div>

        <div className="flex items-center justify-end col-span-1">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreHorizontal className="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-56">
              <DropdownMenuItem
                onClick={() => {
                  setIsEditMode(!isEditMode);
                }}
              >
                <Settings className="w-4 h-4 mr-2" />
                <span>Edit</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {isEditMode ? (
        <Form {...form}>
          <form className="flex flex-col gap-4" onSubmit={form.handleSubmit(onSubmit)}>
            <Separator className="mt-4" />

            <div className="flex items-start w-full gap-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="value"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Value</FormLabel>
                    <FormControl>
                      {decrypt.isLoading ? <Loading /> : <Input {...field} />}
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="comment"
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormLabel>Comment</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder="Comments can be used to explain what this secret is used for"
                    />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />

            <Separator className="" />

            <div className="flex items-center justify-end w-full gap-4">
              <Button
                type="button"
                disabled={update.isLoading}
                variant={update.isLoading ? "disabled" : "secondary"}
                onClick={() => {
                  setIsEditMode(false);
                }}
              >
                Cancel
              </Button>
              <Button
                disabled={!form.formState.isValid || update.isLoading}
                type="submit"
                variant={update.isLoading || !form.formState.isValid ? "disabled" : "primary"}
              >
                {update.isLoading ? <Loading className="w-4 h-4" /> : "Save"}
              </Button>
            </div>
          </form>
        </Form>
      ) : null}
    </li>
  );
};

const Value: React.FC<{ secretId: string }> = ({ secretId }) => {
  const decrypt = trpc.secrets.decrypt.useMutation({
    onError: (err) => {
      toast.error(err.message);
    },
  });

  if (decrypt.isSuccess && decrypt.data) {
    return (
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={() => decrypt.reset()}>
          <EyeOff className="w-4 h-4 " />
        </Button>
        <Badge
          variant="secondary"
          className="flex justify-between gap-2 font-mono font-medium ph-no-capture"
        >
          {decrypt.data.value}
          <CopyButton value={decrypt.data.value} />
        </Badge>
      </div>
    );
  }

  if (decrypt.isLoading) {
    return (
      <Button variant="ghost" size="icon">
        <Loader2 className="w-4 h-4 animate-spin text-content-subtle" />
      </Button>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <Button variant="ghost" size="icon" onClick={() => decrypt.mutate({ secretId })}>
        <Eye className="w-4 h-4 text-content-subtle" />
      </Button>
      <pre className="text-xs text-content-subtle">(encrypted)</pre>
    </div>
  );
};
