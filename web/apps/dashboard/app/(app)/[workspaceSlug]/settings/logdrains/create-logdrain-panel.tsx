"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Item,
  ItemContent,
  ItemDescription,
  ItemTitle,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { DESTINATIONS, DrainMedia } from "./drain-destinations";
import { DestinationFields, NameField, StartFromField } from "./drain-fields";
import {
  type DrainFormValues,
  type DrainKind,
  createDrainSchema,
  emptyDrainForm,
} from "./drain-schema";
import { DrainStepCard } from "./drain-step-card";
import { toHeaderRecord } from "./header-fields";

export function CreateLogdrainPanel({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const utils = trpc.useUtils();
  const [kind, setKind] = useState<DrainKind | null>(null);
  const [confirmChange, setConfirmChange] = useState(false);

  const form = useForm<DrainFormValues>({
    resolver: zodResolver(createDrainSchema),
    defaultValues: emptyDrainForm,
    mode: "onChange",
  });

  const { isDirty } = form.formState;

  const create = trpc.logdrain.create.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      toast.success("Log drain created");
      onClose();
    },
    onError: (error) => toast.error(error.message),
  });

  const startOver = () => {
    form.reset(emptyDrainForm);
    setKind(null);
  };

  const requestChange = () => {
    if (isDirty) {
      setConfirmChange(true);
      return;
    }
    startOver();
  };

  const chooseKind = (next: DrainKind) => {
    form.reset({ ...emptyDrainForm, kind: next });
    setKind(next);
  };

  const submit = form.handleSubmit((values) => {
    const destination =
      values.kind === "http"
        ? {
            kind: "http" as const,
            config: {
              url: values.url.trim(),
              format: values.format,
              headers: toHeaderRecord(values.headers),
            },
          }
        : {
            kind: "axiom" as const,
            config: { dataset: values.dataset.trim(), token: values.token },
          };

    create.mutate({
      name: values.name.trim(),
      stream: "audit_logs",
      startFrom: values.startFrom,
      ...destination,
    });
  });

  const chosen = DESTINATIONS.find((option) => option.kind === kind);

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} onExitComplete={startOver} widthClassName="w-160">
      <SlidePanelHeader className="items-center">
        <SlidePanelTitle>New Log Drain</SlidePanelTitle>
        <SlidePanelCloseButton />
      </SlidePanelHeader>

      <SlidePanelContent>
        <FormProvider {...form}>
          <form onSubmit={submit} className="h-full overflow-y-auto px-6 pb-4 pt-1">
            <div className="flex flex-col gap-4">
              <DrainStepCard
                state={kind ? "settled" : "active"}
                step={1}
                title={chosen?.title ?? "Select a destination"}
                icon={chosen?.icon}
                onReset={requestChange}
              >
                <div className="grid gap-3">
                  {DESTINATIONS.map((destination) => (
                    <Item
                      key={destination.kind}
                      variant="outline"
                      render={
                        <button type="button" onClick={() => chooseKind(destination.kind)}>
                          <DrainMedia kind={destination.kind} />
                          <ItemContent>
                            <ItemTitle>{destination.title}</ItemTitle>
                            <ItemDescription>{destination.description}</ItemDescription>
                          </ItemContent>
                        </button>
                      }
                    />
                  ))}
                </div>
              </DrainStepCard>

              <DrainStepCard
                state={kind ? "active" : "waiting"}
                step={2}
                title="Configure"
                footer={
                  <div className="flex items-center justify-between gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      size="md"
                      disabled={create.isLoading}
                      onClick={requestChange}
                    >
                      Back
                    </Button>
                    <Button type="submit" variant="primary" size="md" loading={create.isLoading}>
                      {create.isLoading ? "Creating…" : "Create Log Drain"}
                    </Button>
                  </div>
                }
              >
                <div className="flex flex-col gap-6">
                  <NameField />
                  <DestinationFields tokenRequired />
                  <StartFromField />
                </div>
              </DrainStepCard>
            </div>
          </form>
        </FormProvider>
      </SlidePanelContent>

      <AlertDialog open={confirmChange} onOpenChange={setConfirmChange}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Switch destination?</AlertDialogTitle>
            <AlertDialogDescription>
              Switching destination clears everything you entered in this form.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction onClick={startOver}>Switch destination</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SlidePanel>
  );
}
