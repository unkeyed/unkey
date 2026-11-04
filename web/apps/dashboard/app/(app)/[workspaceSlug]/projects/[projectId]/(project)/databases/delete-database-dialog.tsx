"use client";

import { IconTriangleWarningOutline12 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  Button,
  DialogContainer,
  FormField,
  Input,
} from "@unkey/ui";
import { useState } from "react";
import type { Database } from "./database-model";

type Props = {
  db: Database;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onDelete: (db: Database) => void;
};

export function DeleteDatabaseDialog({ db, isOpen, onOpenChange, onDelete }: Props) {
  const [typed, setTyped] = useState("");
  const confirmed = typed === db.name;

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        setTyped("");
        onOpenChange(open);
      }}
      title="Delete database"
      footer={
        <Button
          type="submit"
          form="delete-database-form"
          variant="primary"
          color="danger"
          size="md"
          className="w-full"
          disabled={!confirmed}
        >
          Delete database
        </Button>
      }
    >
      <AlertBanner variant="error">
        <IconTriangleWarningOutline12 aria-hidden="true" />
        <AlertBannerDescription>
          Deleting <span className="font-medium">{db.name}</span> removes all its data, branches and
          backups. This cannot be undone.
        </AlertBannerDescription>
      </AlertBanner>
      <form
        id="delete-database-form"
        onSubmit={(e) => {
          e.preventDefault();
          if (confirmed) {
            onDelete(db);
          }
        }}
      >
        <FormField
          className="mt-4"
          label={
            <span>
              Type <span className="font-medium text-gray-12">{db.name}</span> to confirm
            </span>
          }
        >
          {(control) => (
            <Input
              id={control.id}
              aria-describedby={control.describedBy}
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
          )}
        </FormField>
      </form>
    </DialogContainer>
  );
}
