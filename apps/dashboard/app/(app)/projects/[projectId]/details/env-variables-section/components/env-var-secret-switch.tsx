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
      <span className="text-xs text-gray-9">Secret</span>
      <Switch
        className="h-4 w-8 data-[state=checked]:bg-success-9 data-[state=checked]:ring-2 data-[state=checked]:ring-successA-5 data-[state=unchecked]:bg-gray-3 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-3 [&>span]:h-3.5 [&>span]:w-3.5"
        thumbClassName="h-[14px] w-[14px] data-[state=unchecked]:bg-grayA-9 data-[state=checked]:bg-white"
        checked={isSecret}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
      />
    </div>
  );
};
