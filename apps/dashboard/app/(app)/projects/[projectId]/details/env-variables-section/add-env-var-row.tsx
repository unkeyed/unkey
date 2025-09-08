import { Switch } from "@/components/ui/switch";
import { Button, Input } from "@unkey/ui";

type AddEnvVarRowProps = {
  value: { key: string; value: string; isSecret: boolean };
  onChange: (value: { key: string; value: string; isSecret: boolean }) => void;
  onSave: () => void;
  onCancel: () => void;
};

export function AddEnvVarRow({ value, onChange, onSave, onCancel }: AddEnvVarRowProps) {
  const handleSave = () => {
    if (!value.key.trim() || !value.value.trim()) {
      return;
    }
    onSave();
  };

  return (
    <div className="w-full flex px-4 py-3 bg-gray-2 border-b border-gray-4 last:border-b-0">
      <div className="w-fit flex gap-2 items-center">
        <Input
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          placeholder="Variable name"
          className="min-h-[32px] text-xs w-48"
          autoFocus
        />
        <span className="text-gray-9 text-xs px-1">=</span>
        <Input
          value={value.value}
          onChange={(e) => onChange({ ...value, value: e.target.value })}
          placeholder="Variable value"
          className="min-h-[32px] text-xs flex-1"
          type={value.isSecret ? "password" : "text"}
        />
      </div>
      <div className="flex items-center gap-2 ml-auto">
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-9">Secret</span>
          <Switch
            className="
                 h-4 w-8
                 data-[state=checked]:bg-success-9
                 data-[state=checked]:ring-2
                 data-[state=checked]:ring-successA-5
                 data-[state=unchecked]:bg-gray-3
                 data-[state=unchecked]:ring-2
                 data-[state=unchecked]:ring-grayA-3
                 [&>span]:h-3.5 [&>span]:w-3.5
               "
            thumbClassName="h-[14px] w-[14px] data-[state=unchecked]:bg-grayA-9 data-[state=checked]:bg-white"
            checked={value.isSecret}
            onCheckedChange={(checked) => onChange({ ...value, isSecret: checked })}
          />
        </div>
        <Button
          variant="outline"
          className="text-xs"
          onClick={handleSave}
          disabled={!value.key.trim() || !value.value.trim()}
        >
          Save
        </Button>
        <Button variant="outline" onClick={onCancel} className="text-xs">
          Cancel
        </Button>
      </div>
    </div>
  );
}
