import { RenderComponentWithSnippet } from "@/app/components/render";
import { FormInput } from "@unkey/ui";

export const DefaultFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Username"
        description="Choose a unique username for your account"
        placeholder="e.g. gandalf_grey"
      />
    </RenderComponentWithSnippet>
  );
};

// Required field variant
export const RequiredFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Email Address"
        description="We'll send your confirmation email here"
        required
        placeholder="frodo@shire.me"
      />
    </RenderComponentWithSnippet>
  );
};

// Required field with error variant
export const RequiredWithErrorFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Username"
        required
        error="This username is already taken"
        placeholder="e.g. aragorn_king"
      />
    </RenderComponentWithSnippet>
  );
};

// Optional field variant
export const OptionalFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Phone Number"
        description="We'll only use this for important account notifications"
        optional
        placeholder="+1 (555) 123-4567"
      />
    </RenderComponentWithSnippet>
  );
};

// Success variant
export const SuccessFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="API Key"
        description="Your API key has been verified"
        variant="success"
        defaultValue="sk_live_middleearth123"
        placeholder="Enter your API key"
      />
    </RenderComponentWithSnippet>
  );
};

// Warning variant
export const WarningFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Password"
        description="Your password is about to expire"
        variant="warning"
        type="password"
        placeholder="Enter your password"
      />
    </RenderComponentWithSnippet>
  );
};

// Error variant
export const ErrorFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Repository Name"
        error="A repository with this name already exists"
        placeholder="my-awesome-project"
      />
    </RenderComponentWithSnippet>
  );
};

// Disabled variant
export const DisabledFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Organization ID"
        description="Contact admin to change organization ID"
        disabled
        defaultValue="org_fellowship123"
        placeholder="Organization ID"
      />
    </RenderComponentWithSnippet>
  );
};

// With default value
export const DefaultValueFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Project Name"
        description="Name of your new project"
        defaultValue="The Fellowship Project"
        placeholder="Enter project name"
      />
    </RenderComponentWithSnippet>
  );
};

// Readonly variant
export const ReadonlyFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Generated Token"
        description="Copy this token for your records"
        readOnly
        defaultValue="tkn_1ring2rulethemall"
        placeholder="Your token will appear here"
      />
    </RenderComponentWithSnippet>
  );
};

// Complex example with multiple props
export const ComplexFormInputVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormInput
        label="Webhook URL"
        description="Enter the URL where we'll send event notifications"
        required
        placeholder="https://api.yourdomain.com/webhooks"
        className="max-w-lg"
        id="webhook-url-input"
      />
    </RenderComponentWithSnippet>
  );
};
