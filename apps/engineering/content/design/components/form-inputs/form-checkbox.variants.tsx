import { RenderComponentWithSnippet } from "@/app/components/render";
import { FormCheckbox } from "@unkey/ui";

export const DefaultFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I agree to receive marketing emails"
  description="We'll send you occasional updates about our products"
/>`}
    >
      <FormCheckbox
        label="I agree to receive marketing emails"
        description="We'll send you occasional updates about our products"
      />
    </RenderComponentWithSnippet>
  );
};

// Required field variant
export const RequiredFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I agree to the Terms of Service"
  description="You must accept our terms to continue"
  required
/>`}
    >
      <FormCheckbox
        label="I agree to the Terms of Service"
        description="You must accept our terms to continue"
        required
      />
    </RenderComponentWithSnippet>
  );
};

// Required field with error variant
export const RequiredWithErrorFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I agree to the Terms of Service"
  required
  error="You must accept the terms to continue"
/>`}
    >
      <FormCheckbox
        label="I agree to the Terms of Service"
        required
        error="You must accept the terms to continue"
      />
    </RenderComponentWithSnippet>
  );
};

// Optional field variant
export const OptionalFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Subscribe to our newsletter"
  description="Get the latest updates directly to your inbox"
  optional
/>`}
    >
      <FormCheckbox
        label="Subscribe to our newsletter"
        description="Get the latest updates directly to your inbox"
        optional
      />
    </RenderComponentWithSnippet>
  );
};

// Success variant
export const SuccessFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Two-factor authentication enabled"
  description="Your account is now more secure"
  variant="primary"
  color="success"
  defaultChecked
/>`}
    >
      <FormCheckbox
        label="Two-factor authentication enabled"
        description="Your account is now more secure"
        variant="primary"
        color="success"
        defaultChecked
      />
    </RenderComponentWithSnippet>
  );
};

// Warning variant
export const WarningFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Share usage data with developers"
  description="This includes anonymous activity information"
  variant="primary"
  color="warning"
/>`}
    >
      <FormCheckbox
        label="Share usage data with developers"
        description="This includes anonymous activity information"
        variant="primary"
        color="warning"
      />
    </RenderComponentWithSnippet>
  );
};

// Error variant
export const ErrorFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Delete my account"
  error="This action cannot be undone"
  variant="primary"
  color="danger"
/>`}
    >
      <FormCheckbox
        label="Delete my account"
        error="This action cannot be undone"
        variant="primary"
        color="danger"
      />
    </RenderComponentWithSnippet>
  );
};

// Disabled variant
export const DisabledFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Admin privileges"
  description="Only available to authorized personnel"
  disabled
/>`}
    >
      <FormCheckbox
        label="Admin privileges"
        description="Only available to authorized personnel"
        disabled
      />
    </RenderComponentWithSnippet>
  );
};

// Checked variant
export const CheckedFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Remember my preferences"
  description="Save your settings for future visits"
  defaultChecked
/>`}
    >
      <FormCheckbox
        label="Remember my preferences"
        description="Save your settings for future visits"
        defaultChecked
      />
    </RenderComponentWithSnippet>
  );
};

// Outline variant
export const OutlineFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Send me weekly reports"
  description="A summary of your account activity"
  variant="outline"
/>`}
    >
      <FormCheckbox
        label="Send me weekly reports"
        description="A summary of your account activity"
        variant="outline"
      />
    </RenderComponentWithSnippet>
  );
};

// Ghost variant
export const GhostFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="Show advanced settings"
  description="Display additional configuration options"
  variant="ghost"
/>`}
    >
      <FormCheckbox
        label="Show advanced settings"
        description="Display additional configuration options"
        variant="ghost"
      />
    </RenderComponentWithSnippet>
  );
};

// Different size variant
export const XLargeFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I confirm that all information is correct"
  description="Please verify before submission"
  size="xlg"
/>`}
    >
      <FormCheckbox
        label="I confirm that all information is correct"
        description="Please verify before submission"
        size="xlg"
      />
    </RenderComponentWithSnippet>
  );
};

export const LargeFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I confirm that all information is correct"
  description="Please verify before submission"
  size="lg"
/>`}
    >
      <FormCheckbox
        label="I confirm that all information is correct"
        description="Please verify before submission"
        size="lg"
      />
    </RenderComponentWithSnippet>
  );
};

export const MediumFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I confirm that all information is correct"
  description="Please verify before submission"
  size="md"
/>`}
    >
      <FormCheckbox
        label="I confirm that all information is correct"
        description="Please verify before submission"
        size="md"
      />
    </RenderComponentWithSnippet>
  );
};

export const SmallFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<FormCheckbox
  label="I confirm that all information is correct"
  description="Please verify before submission"
  size="sm"
/>`}
    >
      <FormCheckbox
        label="I confirm that all information is correct"
        description="Please verify before submission"
        size="sm"
      />
    </RenderComponentWithSnippet>
  );
};

// Complex example with multiple props
export const ComplexFormCheckboxVariant = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <FormCheckbox
    label="Accept all cookies"
    description="Enable all cookie types for the best experience"
    variant="primary"
    size="md"
    className="border-b pb-2"
  />
  <div className="pl-6 space-y-2">
    <FormCheckbox
      label="Essential cookies"
      description="Required for the website to function"
      variant="outline"
      size="sm"
      defaultChecked
      disabled
    />
    <FormCheckbox
      label="Performance cookies"
      description="Help us improve site performance and usability"
      variant="outline"
      size="sm"
    />
    <FormCheckbox
      label="Functional cookies"
      description="Enable advanced features and personalization"
      variant="outline"
      size="sm"
    />
    <FormCheckbox
      label="Marketing cookies"
      description="Allow us to provide relevant advertisements"
      variant="outline"
      size="sm"
    />
  </div>
</div>`}
    >
      <div className="space-y-4">
        <FormCheckbox
          label="Accept all cookies"
          description="Enable all cookie types for the best experience"
          variant="primary"
          size="md"
          className="border-b pb-2"
        />
        <div className="pl-6 space-y-2">
          <FormCheckbox
            label="Essential cookies"
            description="Required for the website to function"
            variant="outline"
            size="sm"
            defaultChecked
            disabled
          />
          <FormCheckbox
            label="Performance cookies"
            description="Help us improve site performance and usability"
            variant="outline"
            size="sm"
          />
          <FormCheckbox
            label="Functional cookies"
            description="Enable advanced features and personalization"
            variant="outline"
            size="sm"
          />
          <FormCheckbox
            label="Marketing cookies"
            description="Allow us to provide relevant advertisements"
            variant="outline"
            size="sm"
          />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
