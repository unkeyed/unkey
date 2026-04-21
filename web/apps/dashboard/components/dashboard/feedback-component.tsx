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
import { parseAsBoolean, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

const ISSUE_TYPES = ["bug", "feature", "security", "payment", "question", "feedback"] as const;
type IssueType = (typeof ISSUE_TYPES)[number];

export const useFeedback = () => {
  const [{ feedback: open, feedbackType }, setState] = useQueryStates({
    feedback: parseAsBoolean.withOptions({ clearOnDefault: true }),
    feedbackType: parseAsStringLiteral(ISSUE_TYPES).withOptions({ clearOnDefault: true }),
  });

  const openFeedback = useCallback(
    (next: boolean, type?: IssueType) =>
      setState(
        next
          ? { feedback: true, feedbackType: type ?? null }
          : { feedback: null, feedbackType: null },
      ),
    [setState],
  );

  return { open, openFeedback, feedbackType };
};

const feedbackSchema = z.object({
  severity: z.enum(["p0", "p1", "p2", "p3"]),
  issueType: z.enum(ISSUE_TYPES),
  message: z.string().trim().min(20, "Feedback must contain at least 20 characters"),
});

type FormValues = z.infer<typeof feedbackSchema>;

const DEFAULT_FORM_VALUES: FormValues = {
  severity: "p2",
  issueType: "bug",
  message: "",
};

export const Feedback: React.FC = () => {
  const { open, openFeedback, feedbackType } = useFeedback();
  const [internalOpen, setInternalOpen] = useState(false);

  const {
    handleSubmit,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(feedbackSchema),
    defaultValues: DEFAULT_FORM_VALUES,
  });

  useEffect(() => {
    if (open) {
      reset({ ...DEFAULT_FORM_VALUES, issueType: feedbackType ?? DEFAULT_FORM_VALUES.issueType });
      setInternalOpen(true);
      return;
    }
    setInternalOpen(false);
  }, [open, feedbackType, reset]);

  const create = trpc.plain.createIssue.useMutation({
    onSuccess: () => {
      openFeedback(false);
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

  const handleClose = () => {
    openFeedback(false);
  };

  return (
    <DialogContainer
      isOpen={internalOpen}
      onOpenChange={() => {}} // Prevent automatic closing
      showCloseWarning={true}
      onAttemptClose={handleClose}
      modal={true}
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
        onClick={(e) => e.stopPropagation()}
        onMouseDown={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        onKeyUp={(e) => e.stopPropagation()}
      >
        <div className="grid grid-cols-2 gap-4">
          <Controller
            control={control}
            name="issueType"
            render={({ field }) => (
              <div className="flex flex-col gap-1.5">
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
                    <SelectItem value="feedback">Feedback</SelectItem>
                  </SelectContent>
                </Select>
                {errors.issueType && (
                  <div className="text-error-11 text-xs">{errors.issueType.message}</div>
                )}
                <output className="text-gray-9 flex gap-2 items-center text-[13px]">
                  <CircleInfo iconSize="md-medium" aria-hidden="true" />
                  <span>Select the appropriate category</span>
                </output>
              </div>
            )}
          />

          <Controller
            control={control}
            name="severity"
            render={({ field }) => (
              <div className="flex flex-col gap-1.5">
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
                  <CircleInfo iconSize="md-medium" aria-hidden="true" />
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
