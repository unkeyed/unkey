"use client";
import { DialogContainer } from "@/components/dialog-container";
import { Form, FormDescription, FormField, FormItem, FormLabel } from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatNumber } from "@/lib/fmt";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  productId: z.string(),
});

type FormValues = z.infer<typeof formSchema>;
type Props = {
  currentProductId: string;
  products: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quota: {
      requestsPerMonth: number;
    };
  }>;
};

export const ChangePlanButton = ({ currentProductId, products }: Props) => {
  const router = useRouter();
  const [isOpen, setOpen] = useState(false);

  const [isRedirecting, setRedirecting] = useState(false);

  const form = useForm<FormValues>({
    mode: "onChange",
    resolver: zodResolver(formSchema),
    defaultValues: {
      productId: currentProductId,
    },
  });

  const selectedProductId = form.watch("productId");
  const isValid = selectedProductId !== currentProductId;

  const selected = products.find((p) => p.id === selectedProductId)!;

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setOpen}
        title="Change Plan"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="change-plan" // Connect to form ID
              variant="primary"
              size="xlg"
              disabled={!isValid}
              loading={isRedirecting}
              className="w-full rounded-lg"
            >
              Change Plan
            </Button>

            <div className="text-gray-9 text-xs">
              Subscription changes take effect immediately with prorated billing adjustments.
            </div>
          </div>
        }
      >
        <Form {...form}>
          <form
            id="change-plan"
            onSubmit={form.handleSubmit(({ productId }) => {
              setRedirecting(true);
              router.push(`/settings/billing/stripe?action=change_plan&product_id=${productId}`);
            })}
          >
            <FormField
              control={form.control}
              name="productId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Choose new plan</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value || "none"}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {products.map((p) => (
                        <SelectItem value={p.id} disabled={p.id === currentProductId}>
                          {p.name}
                          {p.id === currentProductId ? (
                            <span className="ml-1 text-xs">(Current)</span>
                          ) : null}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    The {selected.name} plan includes{" "}
                    {formatNumber(selected?.quota.requestsPerMonth)} Requests for $
                    {Intl.NumberFormat(undefined, {
                      currency: "USD",
                    }).format(selected?.dollar)}{" "}
                    per month.
                  </FormDescription>
                </FormItem>
              )}
            />
          </form>
        </Form>
      </DialogContainer>
      <Button variant="outline" size="lg" onClick={() => setOpen(true)}>
        Change Plan
      </Button>
    </>
  );
};
