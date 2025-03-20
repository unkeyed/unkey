import { RenderComponentWithSnippet } from "@/app/components/render";
import { FormTextarea } from "@unkey/ui";

export const DefaultFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Description"
        description="Provide a detailed description of your project"
        placeholder="e.g. A fellowship to destroy the One Ring..."
      />
    </RenderComponentWithSnippet>
  );
};

// Required field variant
export const RequiredFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Message"
        description="Share your thoughts with the council"
        required
        placeholder="Speak, friend, and enter your message here..."
      />
    </RenderComponentWithSnippet>
  );
};

// Required field with error variant
export const RequiredWithErrorFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Quest Description"
        required
        error="Your quest description is too short"
        placeholder="Describe your quest in detail..."
      />
    </RenderComponentWithSnippet>
  );
};

// Optional field variant
export const OptionalFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Additional Comments"
        description="Any other information you'd like to share"
        optional
        placeholder="Tell us anything else that might be relevant..."
      />
    </RenderComponentWithSnippet>
  );
};

// Success variant
export const SuccessFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Public Key"
        description="Your public key has been verified"
        variant="success"
        defaultValue="-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCridrK\n-----END PUBLIC KEY-----"
        placeholder="Enter your public key"
      />
    </RenderComponentWithSnippet>
  );
};

// Warning variant
export const WarningFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Notes"
        description="This content will be visible to all team members"
        variant="warning"
        placeholder="Enter your private notes here"
      />
    </RenderComponentWithSnippet>
  );
};

// Error variant
export const ErrorFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Code Snippet"
        error="Invalid syntax in your code"
        placeholder="function castRing() { ... }"
      />
    </RenderComponentWithSnippet>
  );
};

// Disabled variant
export const DisabledFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Terms and Conditions"
        description="Cannot be modified by regular users"
        disabled
        defaultValue="One does not simply walk into Mordor. Its black gates are guarded by more than just Orcs."
        placeholder="Terms and conditions text"
      />
    </RenderComponentWithSnippet>
  );
};

// With default value
export const DefaultValueFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Meeting Minutes"
        description="Notes from the Council of Elrond"
        defaultValue="The Ring must be destroyed. It must be taken deep into Mordor and cast back into the fiery chasm from whence it came."
        placeholder="Enter meeting notes"
      />
    </RenderComponentWithSnippet>
  );
};

// Readonly variant
export const ReadonlyFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Log Output"
        description="Copy this log for troubleshooting"
        readOnly
        defaultValue="[INFO] Fellowship initialized\n[INFO] Ring bearer assigned: Frodo Baggins\n[WARN] Detecting nearby Nazgûl\n[ERROR] Connection to Gondor lost"
        placeholder="Logs will appear here"
      />
    </RenderComponentWithSnippet>
  );
};

// Complex example with multiple props
export const ComplexFormTextareaVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <FormTextarea
        label="Custom Webhook Payload"
        description="Enter the JSON payload template for your webhook"
        required
        placeholder='{\n  "event_type": "ring_destroyed",\n  "data": {\n    "location": "Mount Doom",\n    "timestamp": "{{timestamp}}"\n  }\n}'
        className="max-w-lg font-mono"
        id="webhook-payload-input"
        rows={6}
      />
    </RenderComponentWithSnippet>
  );
};
