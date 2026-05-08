import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, type UseFormReturn, useForm } from "react-hook-form";
import { Button } from "~/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "~/components/ui/field";
import { Input } from "~/components/ui/input";
import { makeKey } from "~/lib/random-key";
import type { Key } from "~/routes/dave-initial-design/-seed";
import { ExpirationPicker, formatDate } from "./expiration-picker";
import { type KeyFormValues, keyFormSchema } from "./key-form-schema";
import { DiscardSecretConfirm, SecretRevealCard, useSecretCloseGate } from "./secret-reveal-card";

type CreateKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (key: Key) => void;
};

export function CreateKeyDialog({ open, onOpenChange, onCreate }: CreateKeyDialogProps) {
  const [createdKey, setCreatedKey] = useState<{ key: Key; plaintext: string } | null>(null);
  const [hasCopied, setHasCopied] = useState(false);

  const form = useForm<KeyFormValues>({
    resolver: zodResolver(keyFormSchema),
    mode: "onSubmit",
    defaultValues: { name: "", expiration: undefined },
  });

  const close = () => {
    if (createdKey) {
      onCreate(createdKey.key);
    }
    setCreatedKey(null);
    setHasCopied(false);
    form.reset({ name: "", expiration: undefined });
    onOpenChange(false);
  };

  const { tryClose, discardConfirm } = useSecretCloseGate({
    hasSecret: createdKey !== null,
    hasCopied,
    onClose: close,
  });

  const handleSubmit = (data: KeyFormValues) => {
    setCreatedKey(makeKey({ name: data.name || null, expiration: data.expiration }));
  };

  const expirationCopy = createdKey?.key.expires
    ? `Expires ${formatDate(new Date(createdKey.key.expires))}`
    : "No expiration";

  return (
    <>
      <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : tryClose())}>
        <DialogContent>
          {createdKey === null ? (
            <FormCard form={form} onCancel={tryClose} onSubmit={handleSubmit} />
          ) : (
            <SecretRevealCard
              title="Save your key"
              description="Copy your secret key now. It cannot be retrieved later."
              plaintext={createdKey.plaintext}
              trailer={expirationCopy}
              onCopied={() => setHasCopied(true)}
              onDone={tryClose}
            />
          )}
        </DialogContent>
      </Dialog>

      <DiscardSecretConfirm {...discardConfirm} />
    </>
  );
}

type FormCardProps = {
  form: UseFormReturn<KeyFormValues>;
  onCancel: () => void;
  onSubmit: (data: KeyFormValues) => void;
};

function FormCard({ form, onCancel, onSubmit }: FormCardProps) {
  const nameError = form.formState.errors.name;

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <DialogHeader>
        <DialogTitle>Create key</DialogTitle>
        <DialogDescription className="sr-only">
          Configure a name and expiration for the new API key.
        </DialogDescription>
      </DialogHeader>

      <DialogBody className="p-5">
        <FieldGroup>
          <Field data-invalid={!!nameError}>
            <FieldLabel htmlFor="create-key-name">Name</FieldLabel>
            <Input
              id="create-key-name"
              placeholder="e.g. Production"
              autoFocus
              aria-invalid={!!nameError}
              {...form.register("name")}
            />
            {nameError ? (
              <FieldError errors={[nameError]} />
            ) : (
              <FieldDescription>Helps you identify this key.</FieldDescription>
            )}
          </Field>

          <Controller
            control={form.control}
            name="expiration"
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldLabel htmlFor="create-key-expiration">Expiration</FieldLabel>
                <ExpirationPicker
                  id="create-key-expiration"
                  value={field.value}
                  invalid={fieldState.invalid}
                  onChange={(d) => {
                    field.onChange(d);
                    field.onBlur();
                  }}
                />
                {fieldState.error ? (
                  <FieldError errors={[fieldState.error]} />
                ) : (
                  <FieldDescription>Leave blank for no expiration.</FieldDescription>
                )}
              </Field>
            )}
          />
        </FieldGroup>
      </DialogBody>

      <DialogFooter>
        <Button type="button" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Create key</Button>
      </DialogFooter>
    </form>
  );
}
