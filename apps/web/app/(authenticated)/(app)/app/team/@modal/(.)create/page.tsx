"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { usePathname, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useOrganizationList } from "@clerk/nextjs";
import { Modal } from "@/components/modal";
import { useEffect, useState } from "react";
const formSchema = z.object({
  name: z.string().min(3).max(50),
  slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
});

export default function TeamCreationModal() {
  const [modalOpen, setModalOpen] = useState(true);
  const pathname = usePathname();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const { createOrganization, isLoaded, setActive } = useOrganizationList();

  const { toast } = useToast();
  const router = useRouter();
  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess() {
      router.push("/app/stripe");
    },
    onError(err) {
      console.error(err);
      toast({
        title: "Error",
        description: err.message,
        variant: "destructive",
      });
    },
  });

  useEffect(() => {
    if (pathname === "/app/team/create" && modalOpen) {
      setModalOpen(false);
    }
  }, [pathname, modalOpen]);

  useEffect(() => {
    if (modalOpen) {
      form.reset();
      form.clearErrors();
    }
  }, [modalOpen]);

  if (!isLoaded) {
    return null;
  }

  const createClerkOrg = async ({
    values,
  }: {
    values: { name: string; slug: string };
  }) => {
    await createOrganization({
      ...values,
    })
      .then((res) => {
        setActive({ organization: res.id });
        createWorkspace.mutate({ ...values, tenantId: res.id });
      })
      .catch((err) => {
        console.error(err);
        toast({
          title: "Error",
          description: err.message,
          variant: "destructive",
        });
      });
  };

  return (
    <Modal isOpen={modalOpen} setIsOpen={setModalOpen} onRequestClose={() => router.back()}>
      <h1 className=" text-2xl font-semibold mb-4 leading-none tracking-tight">
        Create your Team workspace
      </h1>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) => createClerkOrg({ values }))}
          className="flex flex-col space-y-4 max-w-2xl"
        >
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>What should your team be called?</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="slug"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Slug</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  This will be used in urls etc. Only alphanumeric, dashes, underscores and periods
                  are allowed
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="mt-8 space-y-2">
            <Button
              disabled={createWorkspace.isLoading || !form.formState.isValid}
              type="submit"
              className="w-full"
            >
              {createWorkspace.isLoading ? <Loading /> : "Create"}
            </Button>
            <Button
              variant="outline"
              type="button"
              className="w-full"
              onClick={() => router.back()}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Form>
    </Modal>
  );
}
