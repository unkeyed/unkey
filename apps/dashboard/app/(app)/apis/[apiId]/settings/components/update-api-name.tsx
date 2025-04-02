"use client";

import { Loading } from "@/components/dashboard/loading";
import { SettingCard } from "@/components/settings-card";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { InputPasswordEdit } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidateTag } from "../../../../../actions";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name is required and should be at least 3 characters"),
  apiId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: api.name,
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateName = trpc.api.updateName.useMutation({
    onSuccess() {
      toast.success("Your API name has been renamed!");
      revalidateTag(tags.api(api.id));
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.name === api.name || !values.name) {
      return toast.error("Please provide a valid name before saving.");
    }
    await updateName.mutateAsync(values);
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <SettingCard
        className=" py-[19px] mt-5"
        title={
          <div className="flex items-center justify-start gap-2.5">
            <InputPasswordEdit size="xl-medium" className="h-full text-brand-10" />
            <span className="text-sm font-medium text-accent-12">API Name</span>
          </div>
        }
        description={
          <div>
            Not customer-facing. Choose a name that is easy to <br /> recognize.
          </div>
        }
        border="top"
      >
        <div className="flex items-center justify-center w-full gap-2">
          <input type="hidden" name="workspaceId" value={api.workspaceId} />
          <input type="hidden" name="apiId" value={api.id} />
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <>
                <label htmlFor="api-name" className="hidden sr-only">
                  Name
                </label>
                <Input id="api-name" className="w-[257px] h-9" {...field} autoComplete="off" />
              </>
            )}
          />
          <Button
            size="lg"
            className="rounded-lg px-2.5"
            disabled={
              !form.formState.isValid ||
              form.formState.isSubmitting ||
              api.name === form.watch("name")
            }
            loading={form.formState.isSubmitting}
            type="submit"
          >
            {form.formState.isSubmitting ? <Loading /> : "Save"}
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};
