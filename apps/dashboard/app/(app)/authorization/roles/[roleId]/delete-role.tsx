"use client";
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
  role: {
    id: string;
    name: string;
  };
};

export const DeleteRole = ({ trigger, role }: Props) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === role.name, "Please confirm the role's name"),
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

  const isValid = watch("name") === role.name;

  const deleteRole = trpc.rbac.deleteRole.useMutation({
    onSuccess() {
      toast.success("Role deleted successfully", {
        description: "The role has been permanently removed",
      });
      router.push("/authorization/roles");
    },
    onError(err) {
      toast.error("Failed to delete role", {
        description: err.message,
      });
    },
  });

  const onSubmit = async () => {
    try {
      await deleteRole.mutateAsync({ roleId: role.id });
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
        title="Delete Role"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-role-form"
              variant="primary"
              color="danger"
              size="xlg"
              disabled={!isValid || deleteRole.isLoading || isSubmitting}
              loading={deleteRole.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              Delete Role
            </Button>
            <div className="text-gray-9 text-xs">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <p className="text-gray-11 text-[13px]">
          <span className="font-medium">Warning: </span>
          This role will be deleted, and keys with this role will be disconnected from all
          permissions granted by this role. This action is not reversible.
        </p>

        <form id="delete-role-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="space-y-1">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium">{role.name}</span> to confirm
            </p>
            <Input
              {...register("name")}
              placeholder={`Enter "${role.name}" to confirm`}
              autoComplete="off"
            />
          </div>
        </form>
      </DialogContainer>
    </>
  );
};
