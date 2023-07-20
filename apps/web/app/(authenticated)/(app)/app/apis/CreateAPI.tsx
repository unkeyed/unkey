"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { Loading } from "@/components/loading";
import { Modal } from "@/components/modal";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(2).max(50),
});
type Props = {};

export const CreateApiButton = () => {
  const { toast } = useToast();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const create = trpc.api.create.useMutation({
    onSuccess(res) {
      toast({
        title: "API created",
        description: "Your API has been created",
      });
      setModalOpen(false);
      router.refresh();
      router.push(`/app/${res.id}`);
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
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }
  const router = useRouter();

  const [modalOpen, setModalOpen] = useState(false);
  return (
    <>
      <Modal
        isOpen={modalOpen}
        setIsOpen={setModalOpen}
        trigger={() => (
          <Button className=" gap-2">
            <Plus size={18} />
            Create New API
          </Button>
        )}
      >
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="my-api" {...field} />
                  </FormControl>
                  <FormDescription>
                    This is just a human readable name for you and not visible
                    to anyone else
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className=" pt-4 justify-end">
              <Button
                disabled={create.isLoading || !form.formState.isValid}
                className="mt-4 w-1/4"
                type="submit"
              >
                {create.isLoading ? <Loading /> : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </Modal>
    </>
  );
};
