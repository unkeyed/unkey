"use client";

import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, Input } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z
    .string()
    // biome-ignore lint/suspicious/noSelfCompare: <explanation>
    .refine((v) => v === v, "Please confirm the namespace name"),
});

type FormValues = z.infer<typeof formSchema>;
type DeleteNamespaceProps = {
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  namespace: {
    id: string;
    name: string;
  };
};

export const DeleteNamespaceDialog = ({
  isModalOpen,
  onOpenChange,
  namespace,
}: DeleteNamespaceProps) => {
  const router = useRouter();
  const { register, handleSubmit, watch } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });
  const isValid = watch("name") === namespace.name;

  const onSubmit = async () => {
    collection.ratelimitNamespaces.delete(namespace.id);
    router.push("/ratelimits");

    //await deleteNamespace.mutateAsync({ namespaceId: namespace.id });
  };
  return (
    <DialogContainer
      isOpen={isModalOpen}
      onOpenChange={onOpenChange}
      title="Delete Namespace"
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form="delete-namespace-form" // Connect to form ID
            variant="primary"
            color="danger"
            size="xlg"
            disabled={!isValid}
            className="w-full rounded-lg"
          >
            Delete Namespace
          </Button>
          <div className="text-gray-9 text-xs">
            This action cannot be undone – proceed with caution
          </div>
        </div>
      }
    >
      <p className="text-gray-11 text-[13px]">
        <span className="font-medium">Warning: </span>
        Deleting this namespace while it is in use may cause your current requests to fail. You will
        lose access to analytical data.
      </p>

      <form id="delete-namespace-form" onSubmit={handleSubmit(onSubmit)}>
        <div className="space-y-1">
          <p className="text-gray-11 text-[13px]">
            Type <span className="text-gray-12 font-medium">{namespace.name}</span> to confirm
          </p>

          <Input {...register("name")} placeholder={`Enter "${namespace.name}" to confirm`} />
        </div>
      </form>
    </DialogContainer>
  );
};
