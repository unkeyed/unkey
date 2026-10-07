import { GuardedSlidePanel } from "@/components/guarded-slide-panel";
import { IconChevronDownOutline18 } from "@unkey/icons";
import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { type FormEvent, useEffect, useId, useState } from "react";
import { EnvironmentLabel } from "../../../../components/environment-label";
import { useProjectData } from "../../../data-provider";
import { useAddEnvVars } from "../../hooks/use-add-env-vars";
import { useVariableRows } from "../variables-table/use-variable-rows";
import { VariablesTable } from "../variables-table/variables-table";

const ALL_ENVIRONMENTS = "__all__";

type AddEnvVarExpandableProps = {
  projectId: string;
  appId: string;
  isOpen: boolean;
  onClose: () => void;
};

export function AddEnvVarExpandable({
  projectId,
  appId,
  isOpen,
  onClose,
}: AddEnvVarExpandableProps) {
  const { environments } = useProjectData();
  const draft = useVariableRows();
  const [environmentId, setEnvironmentId] = useState(ALL_ENVIRONMENTS);
  const selectedEnvironment = environments.find((env) => env.id === environmentId);
  const environmentSelectId = useId();
  const targetEnvironmentIds =
    environmentId === ALL_ENVIRONMENTS ? environments.map((e) => e.id) : [environmentId];

  const isDirty =
    environmentId !== ALL_ENVIRONMENTS ||
    draft.rows.some((row) => row.key !== "" || row.value !== "" || row.sensitive);
  const resetAndClose = () => {
    draft.reset();
    setEnvironmentId(ALL_ENVIRONMENTS);
    onClose();
  };
  const add = useAddEnvVars({ projectId, appId, draft, onAdded: resetAndClose });

  useEffect(
    function purgeLegacyPersistedDraft() {
      // Earlier versions persisted this draft (possibly containing secrets) to
      // sessionStorage, which survives page reloads. Remove any leftovers.
      sessionStorage.removeItem(`env-vars-add-${appId}`);
    },
    [appId],
  );

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    add.submit(targetEnvironmentIds);
  };

  return (
    <GuardedSlidePanel
      dirty={isDirty}
      isOpen={isOpen}
      onClose={resetAndClose}
      widthClassName="w-240"
    >
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>Add environment variables</SlidePanelTitle>
          <SlidePanelDescription>
            Paste or drop a .env file into the table to add many at once.
          </SlidePanelDescription>
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>

      <SlidePanelContent>
        <form onSubmit={onSubmit} className="flex h-full flex-col">
          <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 pt-4 pb-6">
            <VariablesTable draft={draft} />
            <div className="flex flex-col gap-1.5">
              <label htmlFor={environmentSelectId} className="text-sm font-medium text-gray-12">
                Environment
              </label>
              <Select
                value={environmentId}
                onValueChange={(value) => {
                  if (value !== null) {
                    setEnvironmentId(value);
                  }
                }}
                items={[
                  { value: ALL_ENVIRONMENTS, label: "All environments" },
                  ...environments.map((env) => ({ value: env.id, label: env.slug })),
                ]}
              >
                <SelectTrigger
                  id={environmentSelectId}
                  wrapperClassName="w-60"
                  rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
                >
                  <SelectValue placeholder="Select environment">
                    {selectedEnvironment ? (
                      <EnvironmentLabel
                        environment={selectedEnvironment}
                        className="text-gray-12"
                      />
                    ) : (
                      "All environments"
                    )}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent className="z-60">
                  <SelectItem value={ALL_ENVIRONMENTS}>All environments</SelectItem>
                  {environments.map((env) => (
                    <SelectItem key={env.id} value={env.id}>
                      <EnvironmentLabel environment={env} className="text-gray-12" />
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <SlidePanelFooter className="flex items-center justify-end gap-4 bg-raised py-5">
            <Button
              type="submit"
              variant="primary"
              size="md"
              className="px-3"
              loading={add.isPending}
              disabled={add.isPending || targetEnvironmentIds.length === 0}
            >
              Save
            </Button>
          </SlidePanelFooter>
        </form>
      </SlidePanelContent>
    </GuardedSlidePanel>
  );
}
