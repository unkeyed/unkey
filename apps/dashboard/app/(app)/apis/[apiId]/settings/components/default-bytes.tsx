"use client";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyAuthId: z.string(),
  defaultBytes: z
    .number()
    .min(8, "Key must be between 8 and 255 bytes long")
    .max(255, "Key must be between 8 and 255 bytes long")
    .optional(),
});

type Props = {
  keyAuth: {
    id: string;
    defaultBytes: number | undefined | null;
  };
};

export const DefaultBytes: React.FC<Props> = ({ keyAuth }) => {
  const router = useRouter();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      defaultBytes: keyAuth.defaultBytes ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultBytes = trpc.api.setDefaultBytes.useMutation({
    onSuccess() {
      toast.success("Default Byte length for this API is updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultBytes === keyAuth.defaultBytes || !values.defaultBytes) {
      return toast.error(
        "Please provide a different byte-size than already existing one as default",
      );
    }
    await setDefaultBytes.mutateAsync(values);
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Default Bytes"
        description={
          <div className="max-w-[380px]">
            Sets the default byte size for keys under this API. Must be between 8 and 255.
          </div>
        }
        border="top"
        className="border-b-1"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <input type="hidden" name="keyAuthId" value={keyAuth.id} />

          <Controller
            control={control}
            name="defaultBytes"
            render={({ field }) => (
              <FormInput
                {...field}
                className="w-[16.5rem]"
                autoComplete="off"
                type="text"
                onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
              />
            )}
          />

          <Button
            size="lg"
            variant="primary"
            disabled={!isValid || isSubmitting || !isDirty}
            type="submit"
            loading={isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};
