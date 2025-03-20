"use client";
import { useForm } from "@tanstack/react-form";
import { useStore } from "@tanstack/react-store";
import { Button, FormInput, FormTextarea } from "@unkey/ui";
import { useState } from "react";
import create, { type ServerResponse } from "../server/action";
import { formOpts } from "../validator";

const EMAIL_REGEX = /^([A-Z0-9_+-]+\.?)*[A-Z0-9_+-]@([A-Z0-9][A-Z0-9-]*\.)+[A-Z]{2,}$/i;

export const ContactForm = () => {
  const [loading, setLoading] = useState(false);
  const initialServerState = {
    status: "success",
    submitted: false,
  } as const;

  const [serverState, setServerState] = useState<ServerResponse>(initialServerState);

  const form = useForm({
    ...formOpts,
  });

  const formErrors = useStore(form.store, (formState) => formState.errors ?? []);

  return (
    <div className="p-8 md:p-8 rounded-lg relative bg-black border-0 overflow-hidden before:absolute before:inset-0 before:p-[1px] before:rounded-lg before:bg-linear-to-r before:from-orange-500 before:via-purple-500 before:to-blue-500 before:-z-10 shadow-[0_0_50px_rgba(249,115,22,0.25)]">
      <div className="h-full">
        <div className="">
          <h2 className="text-2xl font-semibold">Apply now</h2>
          <p className="py-4">Complete the fields below and we'll be in touch.</p>
        </div>

        <form
          action={async (formData) => {
            const result = await create(formData);
            setServerState(result);
            if (result.status === "success") {
              form.reset();
            }
            setLoading(false);
          }}
          onSubmit={() => {
            form.handleSubmit();
            setLoading(true);
          }}
          className="space-y-6"
        >
          {/* Name field */}
          <div className="space-y-2">
            <form.Field
              name="Full Name"
              validators={{
                onChangeAsyncDebounceMs: 500,
                onChangeAsync: async ({ value }) => {
                  if (!value) {
                    return "A name is required";
                  }
                  if (value.length < 3) {
                    return "Name must be at least 3 characters";
                  }
                  return undefined;
                },
              }}
              children={(field) => {
                return (
                  <FormInput
                    className="dark"
                    label={field.name}
                    placeholder="Bruce Wayne"
                    id={field.name}
                    name={field.name}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    error={field.state.meta.errors.join(",")}
                    onChange={(e) => field.handleChange(e.target.value)}
                    required
                  />
                );
              }}
            />
          </div>

          {/* Email field */}
          <div className="space-y-2 relative">
            <form.Field
              name="Email"
              validators={{
                onChangeAsyncDebounceMs: 500,
                onChangeAsync: async ({ value }) => {
                  if (!value) {
                    return "Email is required";
                  }
                  if (!EMAIL_REGEX.test(value)) {
                    return "Email must be at valid";
                  }
                },
              }}
              children={(field) => {
                return (
                  <FormInput
                    className="dark"
                    type="email"
                    placeholder="bruce.wayne@gotham.com"
                    label={field.name}
                    id={field.name}
                    name={field.name}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    error={field.state.meta.errors.join(",")}
                    onChange={(e) => field.handleChange(e.target.value)}
                    required
                  />
                );
              }}
            />
          </div>

          {/* YC batch field */}
          <div className="space-y-2 relative">
            <form.Field
              name="YC Batch"
              validators={{
                onChangeAsyncDebounceMs: 500,
                onChangeAsync: async ({ value }) => {
                  if (!value) {
                    return "A YC batch is required";
                  }
                  if (value.length < 3) {
                    return "YC batch must be at least 3 characters";
                  }
                },
              }}
              children={(field) => {
                return (
                  <FormInput
                    className="dark"
                    placeholder="YCW2025"
                    id={field.name}
                    label={field.name}
                    name={field.name}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    error={field.state.meta.errors.join(",")}
                    onChange={(e) => field.handleChange(e.target.value)}
                    required
                  />
                );
              }}
            />
          </div>

          {/* Workspace ID field */}
          <div className="space-y-2 relative">
            <form.Field
              name="Workspace ID"
              validators={{
                onBlur: ({ value }) =>
                  value.startsWith("ws_") && value.length < 19
                    ? "Workspace ID must start with 'ws_' and be at least 19 characters long"
                    : undefined,
              }}
              children={(field) => {
                return (
                  <FormInput
                    className="dark"
                    placeholder="ws_123"
                    id={field.name}
                    name={field.name}
                    label={field.name}
                    value={field.state.value}
                    error={field.state.meta.errors.join(",")}
                    onBlur={field.handleBlur}
                    onChange={(e) => field.handleChange(e.target.value)}
                  />
                );
              }}
            />
          </div>

          {/* Migration support field */}
          <div className="space-y-2 relative">
            <form.Field
              name="Migrating From"
              children={(field) => {
                return (
                  <FormInput
                    className="dark"
                    placeholder="we are coming from Apigee"
                    id={field.name}
                    name={field.name}
                    label={field.name}
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                  />
                );
              }}
            />
          </div>

          <div className="space-y-2 relative">
            <form.Field
              name="More Info"
              children={(field) => {
                return (
                  <FormTextarea
                    className="dark"
                    id={field.name}
                    label={field.name}
                    name={field.name}
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                  />
                );
              }}
            />
          </div>

          {/* Submit button */}
          <div className="space-y-2 justify-end align-bottom">
            <form.Subscribe
              selector={(state) => [state.canSubmit, loading, state.isPristine, state.isSubmitting]}
              children={([canSubmit, loading, isPristine, isSubmitting]) => (
                <Button
                  size="lg"
                  className="dark w-full bg-orange-500 hover:bg-orange-400 inset-x-0 bottom-0 "
                  type="submit"
                  loading={loading}
                  disabled={!canSubmit || loading || isPristine || isSubmitting}
                >
                  Submit
                </Button>
              )}
            />
          </div>
          {serverState?.status === "success" && serverState.submitted ? (
            <div className="text-green-500 p-4 rounded-md bg-green-500/10 mb-4">
              <p>Thank you for your submission! We'll be in touch soon.</p>
            </div>
          ) : null}

          {serverState?.status === "error" && serverState.errors?.length > 0 && (
            <div className="text-red-500 p-4 rounded-md bg-red-500/10 mb-4">
              {serverState?.status === "error" &&
                serverState.errors?.map((error) => (
                  <p key={`server-error-${error}`} className="mb-1">
                    {error}
                  </p>
                ))}
              {formErrors.map((error) => (
                <p key={`form-error-${error}`} className="mb-1">
                  {error}
                </p>
              ))}
            </div>
          )}
        </form>
      </div>
    </div>
  );
};
