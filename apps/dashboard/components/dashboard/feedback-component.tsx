"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CircleInfo } from "@unkey/icons";
import {
  Button,
  DialogContainer,
  FormTextarea,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { parseAsBoolean, useQueryState } from "nuqs";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

export const useFeedback = () => {
  return useQueryState("feedback", {
    ...parseAsBoolean,
    clearOnDefault: true,
  });
};

const feedbackSchema = z.object({
  severity: z.enum(["p0", "p1", "p2", "p3"]),
  issueType: z.enum(["bug", "feature", "security", "payment", "question"]),
  message: z.string().trim().min(20, "Feedback must contain at least 20 characters"),
});

type FormValues = z.infer<typeof feedbackSchema>;

export const Feedback: React.FC = () => {
  const [open, setOpen] = useFeedback();

  const {
    handleSubmit,
    control,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(feedbackSchema),
    defaultValues: {
      severity: "p2",
      issueType: "bug",
      message: "",
    },
  });

  const create = trpc.plain.createIssue.useMutation({
    onSuccess: () => {
      setOpen(false);
      toast.success("Your issue has been created, we'll get back to you as soon as possible");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const onSubmitForm = async (values: FormValues) => {
    try {
      await create.mutateAsync(values);
    } catch (error) {
      console.error("Form submission error:", error);
    }
  };

  return (
    <DialogContainer
      isOpen={Boolean(open)}
      onOpenChange={setOpen}
      title="Report an issue"
      subTitle="What went wrong or how can we improve?"
      footer={
        <div className="flex flex-col items-center justify-center w-full gap-2">
          <Button
            type="submit"
            form="feedback-form"
            variant="primary"
            size="xlg"
            disabled={isSubmitting || create.isLoading}
            loading={isSubmitting || create.isLoading}
            className="w-full rounded-lg"
          >
            Send
          </Button>
          <div className="text-xs text-gray-9">
            We'll try to get back to you as soon as possible
          </div>
        </div>
      }
    >
      <form
        id="feedback-form"
        onSubmit={handleSubmit(onSubmitForm)}
        className="flex flex-col gap-4"
      >
        <div className="grid grid-cols-2 gap-4">
          <Controller
            control={control}
            name="issueType"
            render={({ field }) => (
              <div className="space-y-1.5">
                <div className="text-gray-11 text-[13px] flex items-center">Area</div>
                <Select onValueChange={field.onChange} value={field.value}>
                  <SelectTrigger className="h-9">
                    <SelectValue placeholder="What area is this" />
                  </SelectTrigger>
                  <SelectContent className="border-none rounded-md">
                    <SelectItem value="bug">Bug</SelectItem>
                    <SelectItem value="feature">Feature Request</SelectItem>
                    <SelectItem value="security">Security</SelectItem>
                    <SelectItem value="payment">Payments</SelectItem>
                    <SelectItem value="question">General Question</SelectItem>
                  </SelectContent>
                </Select>
                {errors.issueType && (
                  <div className="text-error-11 text-xs">{errors.issueType.message}</div>
                )}
                <output className="text-gray-9 flex gap-2 items-center text-[13px]">
                  <CircleInfo iconsize="md-medium" aria-hidden="true" />
                  <span>Select the appropriate category</span>
                </output>
              </div>
            )}
          />

          <Controller
            control={control}
            name="severity"
            render={({ field }) => (
              <div className="space-y-1.5">
                <div className="text-gray-11 text-[13px] flex items-center">Severity</div>
                <Select onValueChange={field.onChange} value={field.value}>
                  <SelectTrigger className="h-9">
                    <SelectValue placeholder="Select a severity" />
                  </SelectTrigger>
                  <SelectContent className="border-none rounded-md">
                    <SelectItem value="p0">Urgent</SelectItem>
                    <SelectItem value="p1">High</SelectItem>
                    <SelectItem value="p2">Normal</SelectItem>
                    <SelectItem value="p3">Low</SelectItem>
                  </SelectContent>
                </Select>
                {errors.severity && (
                  <div className="text-error-11 text-xs">{errors.severity.message}</div>
                )}
                <output className="text-gray-9 flex gap-2 items-center text-[13px]">
                  <CircleInfo iconsize="md-medium" aria-hidden="true" />
                  <span>How urgent is this issue?</span>
                </output>
              </div>
            )}
          />
        </div>

        <Controller
          control={control}
          name="message"
          render={({ field }) => (
            <FormTextarea
              label="What can we do for you?"
              description="Please include all information relevant to your issue."
              placeholder="Please describe your issue in detail..."
              error={errors.message?.message}
              className="min-h-24 w-full"
              {...field}
            />
          )}
        />
      </form>
    </DialogContainer>
  );
};
