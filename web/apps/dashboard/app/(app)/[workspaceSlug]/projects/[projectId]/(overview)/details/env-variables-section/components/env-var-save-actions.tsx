import { Button } from "@unkey/ui";

export const EnvVarSaveActions = ({
  save,
  cancel,
  isSubmitting,
}: {
  isSubmitting: boolean;
  save: {
    disabled: boolean;
  };
  cancel: {
    disabled: boolean;
    onClick: () => void;
  };
}) => {
  return (
    <>
      <Button
        type="submit"
        variant="primary"
        className="text-xs"
        disabled={save.disabled}
        loading={isSubmitting}
      >
        Save
      </Button>
      <Button
        type="button"
        variant="ghost"
        disabled={cancel.disabled}
        onClick={cancel.onClick}
        className="text-xs"
      >
        Cancel
      </Button>
    </>
  );
};
