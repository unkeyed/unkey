import type { EnvVar } from "@/lib/trpc/routers/deploy/project/envs/getEnvs";
import { Eye, EyeSlash, PenWriting3, Trash } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { useEffect, useState } from "react";

type EnvVarRowProps = {
  envVar: EnvVar;
  isEditing: boolean;
  onEdit: () => void;
  onSave: (updates: Partial<EnvVar>) => void;
  onDelete: () => void;
  onCancel: () => void;
};

export function EnvVarRow({
  envVar,
  isEditing,
  onEdit,
  onSave,
  onDelete,
  onCancel,
}: EnvVarRowProps) {
  const [editKey, setEditKey] = useState(envVar.key);
  const [editValue, setEditValue] = useState(envVar.value);
  const [isValueVisible, setIsValueVisible] = useState(false);

  // Make value visible when entering edit mode
  useEffect(() => {
    if (isEditing) {
      setIsValueVisible(true);
    }
  }, [isEditing]);

  const handleSave = () => {
    if (!editKey.trim() || !editValue.trim()) {
      return;
    }
    onSave({ key: editKey.trim(), value: editValue.trim() });
  };

  const handleCancel = () => {
    setEditKey(envVar.key);
    setEditValue(envVar.value);
    setIsValueVisible(false);
    onCancel();
  };

  if (isEditing) {
    return (
      <div className="w-full flex px-4 py-3 bg-gray-2 h-12">
        <div className="w-fit flex gap-2 items-center font-mono">
          <Input
            value={editKey}
            onChange={(e) => setEditKey(e.target.value)}
            placeholder="Variable name"
            className="min-h-[32px] text-xs w-48 "
            autoFocus
          />
          <span className="text-gray-9 text-xs px-1">=</span>
          <Input
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            placeholder="Variable value"
            className="min-h-[32px] text-xs flex-1"
            type="text"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <Button
            variant="outline"
            className="text-xs"
            onClick={handleSave}
            disabled={!editKey.trim() || !editValue.trim()}
          >
            Save
          </Button>
          <Button variant="outline" onClick={handleCancel} className="text-xs">
            Cancel
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full px-4 py-3 flex items-center hover:bg-gray-2 transition-colors border-b border-gray-4 last:border-b-0 h-12">
      <div className="flex items-center flex-1 min-w-0">
        <div className="text-gray-12 font-medium text-xs font-mono w-48 truncate">{envVar.key}</div>
        <span className="text-gray-9 text-xs px-2">=</span>
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div className="text-gray-10 text-xs font-mono truncate flex-1">
            {envVar.isSecret && !isValueVisible ? "••••••••••••••••" : envVar.value}
          </div>
          {envVar.isSecret && (
            <Button
              size="icon"
              variant="outline"
              onClick={() => setIsValueVisible(!isValueVisible)}
              className="size-7 text-gray-9 hover:text-gray-11 shrink-0"
            >
              {isValueVisible ? (
                <EyeSlash className="!size-[14px]" size="sm-medium" />
              ) : (
                <Eye className="!size-[14px]" size="sm-medium" />
              )}
            </Button>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2 shrink-0 ml-2">
        <Button size="icon" variant="outline" onClick={onEdit} className="size-7 text-gray-9">
          <PenWriting3 className="!size-[14px]" size="sm-medium" />
        </Button>
        <Button size="icon" variant="outline" onClick={onDelete} className="size-7 text-gray-9">
          <Trash className="!size-[14px]" size="sm-medium" />
        </Button>
      </div>
    </div>
  );
}
