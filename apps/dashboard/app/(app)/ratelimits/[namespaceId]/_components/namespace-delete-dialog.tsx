"use client";

import { revalidateTag } from "@/app/actions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { FormField } from "./form-field";

const intent = "delete namespace";

const formSchema = z.object({
  // biome-ignore lint/suspicious/noSelfCompare: <explanation>
  name: z.string().refine((v) => v === v, "Please confirm the namespace name"),
  intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  namespaceId: validation.unkeyId,
});

type FormValues = z.infer<typeof formSchema>;
type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  namespace: {
    id: string;
    workspaceId: string;
    name: string;
  };
}>;

export const DeleteNamespaceDialog = ({ isModalOpen, onOpenChange, namespace }: Props) => {
  const router = useRouter();
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });

  const isValid = watch("intent") === intent && watch("name") === namespace.name;

  const deleteNamespace = trpc.ratelimit.namespace.delete.useMutation({
    onSuccess() {
      toast.success("Namespace Deleted", {
        description: "Your namespace and all its overridden identifiers have been deleted.",
      });
      revalidateTag(tags.namespace(namespace.id));
      router.push("/ratelimits");
      onOpenChange(false);
    },
    onError(err) {
      toast.error("Failed to delete namespace", {
        description: err.message,
      });
    },
  });

  const onSubmit = async () => {
    await deleteNamespace.mutateAsync({ namespaceId: namespace.id });
  };

  return (
    <Dialog open={isModalOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className="bg-gray-1 dark:bg-black drop-shadow-2xl border-error-7 rounded-lg p-0 gap-0"
        onOpenAutoFocus={(e) => {
          e.preventDefault();
        }}
      >
        <DialogHeader className="border-b border-gray-4">
          <DialogTitle className="px-6 py-4 text-gray-12 font-medium text-base">
            Delete Namespace
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-4 py-4 px-6 bg-accent-2">
            <Alert variant="alert" className="bg-error-2 text-error-11">
              <AlertTitle className="text-error-11">Warning</AlertTitle>
              <AlertDescription>
                This namespace will be deleted, along with all of its identifiers and data. This
                action cannot be undone.
              </AlertDescription>
            </Alert>

            <FormField
              label="Namespace Name Confirmation"
              tooltip="Enter the namespace name to confirm deletion"
              error={errors.name?.message}
            >
              <Input
                {...register("name")}
                placeholder={`Enter "${namespace.name}" to confirm`}
                className="border border-gray-4 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md"
              />
            </FormField>

            <FormField
              label="Intent Confirmation"
              tooltip="Type 'delete namespace' to confirm"
              error={errors.intent?.message}
            >
              <Input
                {...register("intent")}
                placeholder='Type "delete namespace" to confirm'
                className="border border-gray-4 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md"
              />
            </FormField>

            <input type="hidden" {...register("namespaceId")} defaultValue={namespace.id} />
          </div>

          <DialogFooter className="px-6 py-4 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <div className="flex w-full gap-4 justify-end">
                <Button
                  type="button"
                  onClick={() => onOpenChange(false)}
                  disabled={deleteNamespace.isLoading || isSubmitting}
                  className="h-10"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="destructive"
                  disabled={!isValid || deleteNamespace.isLoading || isSubmitting}
                  loading={deleteNamespace.isLoading || isSubmitting}
                  className="h-10"
                >
                  Delete Namespace
                </Button>
              </div>
              <div className="text-gray-9 text-xs">This action cannot be undone</div>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export const DeleteNamespace = ({
  namespace,
}: {
  namespace: Props["namespace"];
}) => {
  return (
    <Card className="relative border-2 border-error-7">
      <CardHeader>
        <CardTitle>Delete</CardTitle>
        <CardDescription>
          This namespace will be deleted, along with all of its identifiers and data. This action
          cannot be undone.
        </CardDescription>
      </CardHeader>

      <CardFooter className="z-10 justify-end">
        <DeleteNamespaceDialog namespace={namespace} isModalOpen={false} onOpenChange={() => {}} />
        <Button type="button" variant="destructive">
          Delete namespace
        </Button>
      </CardFooter>
    </Card>
  );
};
