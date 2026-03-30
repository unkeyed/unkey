import { Switch } from "@/components/ui/switch";

export const EnvVarSecretSwitch = ({
  isSecret,
  onCheckedChange,
  disabled,
}: {
  isSecret: boolean;
  onCheckedChange?(checked: boolean): void;
  disabled: boolean;
}) => {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-gray-10">Secret</span>
      <Switch
        className="data-[state=checked]:bg-success-9 data-[state=checked]:ring-2 data-[state=checked]:ring-successA-5 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-3"
        checked={isSecret}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
      />
    </div>
  );
};
