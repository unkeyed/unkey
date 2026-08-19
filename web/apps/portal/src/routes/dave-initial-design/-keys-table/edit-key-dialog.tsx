import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
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
import type { Key } from "~/routes/dave-initial-design/-seed";
import { ExpirationPicker } from "./expiration-picker";
import { type KeyFormValues, keyFormSchema } from "./key-form-schema";

export type EditKeyValues = {
  name: string | null;
  expires: number | null;
};

type EditKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyToEdit: Key | null;
  onSave: (id: string, values: EditKeyValues) => void;
};

export function EditKeyDialog({ open, onOpenChange, keyToEdit, onSave }: EditKeyDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {keyToEdit ? (
        <EditKeyDialogBody
          keyToEdit={keyToEdit}
          onSave={onSave}
          onClose={() => onOpenChange(false)}
        />
      ) : null}
    </Dialog>
  );
}

type EditKeyDialogBodyProps = {
  keyToEdit: Key;
  onSave: (id: string, values: EditKeyValues) => void;
  onClose: () => void;
};

function toFormValues(k: Key): KeyFormValues {
  return {
    name: k.name ?? "",
    expiration: k.expires ? new Date(k.expires) : undefined,
  };
}

function EditKeyDialogBody({ keyToEdit, onSave, onClose }: EditKeyDialogBodyProps) {
  const form = useForm<KeyFormValues>({
    resolver: zodResolver(keyFormSchema),
    mode: "onSubmit",
    defaultValues: toFormValues(keyToEdit),
  });

  useEffect(() => {
    form.reset(toFormValues(keyToEdit));
  }, [keyToEdit, form]);

  const nameError = form.formState.errors.name;

  const handleValidSubmit = (data: KeyFormValues) => {
    onSave(keyToEdit.id, {
      name: data.name === "" ? null : data.name,
      expires: data.expiration?.getTime() ?? null,
    });
    onClose();
  };

  return (
    <DialogContent>
      <form onSubmit={form.handleSubmit(handleValidSubmit)} noValidate>
        <DialogHeader className="border-b-0 pb-2">
          <DialogTitle>Edit key</DialogTitle>
          <DialogDescription>Update this key's name and expiration.</DialogDescription>
        </DialogHeader>

        <DialogBody className="px-5 pt-2 pb-5">
          <FieldGroup>
            <Field data-invalid={!!nameError}>
              <FieldLabel htmlFor="edit-key-name">Name</FieldLabel>
              <Input
                id="edit-key-name"
                placeholder="e.g. Production"
                autoFocus
                aria-invalid={!!nameError}
                {...form.register("name")}
              />
              {nameError ? <FieldError errors={[nameError]} /> : null}
            </Field>

            <Controller
              control={form.control}
              name="expiration"
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor="edit-key-expiration">Expiration</FieldLabel>
                  <ExpirationPicker
                    id="edit-key-expiration"
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
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit">Save changes</Button>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}
