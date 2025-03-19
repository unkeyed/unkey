import { formOptions } from "@tanstack/react-form/nextjs";

export const formOpts = formOptions({
  defaultValues: {
    "Full Name": "",
    Email: "",
    "YC Batch": "",
    "Workspace ID": "",
    "Migrating From": "",
    "More Info": "",
  },
});
