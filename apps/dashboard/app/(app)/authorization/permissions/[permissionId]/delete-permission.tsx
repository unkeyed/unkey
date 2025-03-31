"use client";
import { revalidate } from "@/app/actions";
import { DialogContainer } from "@/components/dialog-container";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
  permission: {
    id: string;
    name: string;
  };
};

export const DeletePermission: React.FC<Props> = ({ trigger, permission }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === permission.name, "Please confirm the permission's name"),
  });

  type FormValues = z.infer<typeof formSchema>;

  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
    },
  });

  const isValid = watch("name") === permission.name;

  const deletePermission = trpc.rbac.deletePermission.useMutation({
    onSuccess() {
      toast.success("Permission deleted successfully", {
        description: "The permission has been permanently removed",
      });
      revalidate("/authorization/permissions");
      router.push("/authorization/permissions");
    },
    onError(err) {
      toast.error("Failed to delete permission", {
        description: err.message,
      });
    },
  });

  const onSubmit = async () => {
    try {
      await deletePermission.mutateAsync({ permissionId: permission.id });
    } catch (error) {
      console.error("Delete error:", error);
    }
  };

  const handleOpenChange = (newState: boolean) => {
    setOpen(newState);
    if (!newState) {
      reset();
    }
  };

  return (
    <>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
      <div onClick={() => handleOpenChange(true)}>{trigger}</div>

      <DialogContainer
        isOpen={open}
        onOpenChange={handleOpenChange}
        title="Delete Permission"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-permission-form"
              variant="primary"
              color="danger"
              size="xlg"
              disabled={!isValid || deletePermission.isLoading || isSubmitting}
              loading={deletePermission.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              Delete Permission
            </Button>
            <div className="text-gray-9 text-xs">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <p className="text-gray-11 text-[13px]">
          <span className="font-medium">Warning: </span>
          This action is not reversible. The permission will be removed from all roles and keys that
          currently use it.
        </p>

        <form id="delete-permission-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="space-y-1">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium break-all">{permission.name}</span> to
              confirm
            </p>
            <Input
              {...register("name")}
              placeholder={`Enter "${permission.name}" to confirm`}
              autoComplete="off"
            />
          </div>
        </form>
      </DialogContainer>
    </>
  );
};
