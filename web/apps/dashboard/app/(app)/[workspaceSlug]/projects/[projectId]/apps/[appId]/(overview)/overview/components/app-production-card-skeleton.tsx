import {
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconLayers3Outline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
  IconShieldKeyOutline18,
} from "@unkey/icons";
import { Card, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import { Canvas, Group, HFlow, Node, NodeHeader, RowItem, Rows } from "./canvas/nodes";

function Bar({ className }: { className: string }) {
  return <Skeleton className={cn("inline-block h-3 rounded align-middle", className)} />;
}

export function AppProductionCardSkeleton() {
  return (
    <Card className="flex flex-col">
      <Canvas label="App canvas loading">
        <Group icon={<IconEarthOutline18 />} label="Domains">
          <Node>
            <NodeHeader icon={<IconEarthOutline18 />} name="Domains" />
            <Rows>
              <RowItem
                icon={<IconEarthOutline18 />}
                label="Unkey"
                value={<Bar className="w-32" />}
              />
              <RowItem
                icon={<IconLink4Outline18 />}
                label="Custom"
                value={<Bar className="w-28" />}
              />
            </Rows>
          </Node>
        </Group>
        <HFlow empty />
        <Group icon={<IconCubeOutline18 />} label="App">
          <Node>
            <NodeHeader
              icon={<Skeleton className="size-3.5 rounded-full" />}
              name={<Bar className="w-16" />}
              right={<Bar className="w-10" />}
            />
            <Rows>
              <RowItem
                icon={<IconCodeBranchOutline18 />}
                label="Branch"
                value={<Bar className="w-20" />}
              />
              <RowItem
                icon={<IconCodeCommitOutline18 />}
                label="Commit"
                value={<Bar className="w-36" />}
              />
            </Rows>
            <Rows>
              <RowItem
                icon={<IconEarthOutline18 />}
                label="Regions"
                value={<Bar className="w-10" />}
              />
              <RowItem
                icon={<IconLayers3Outline18 />}
                label="Instances"
                value={<Bar className="w-4" />}
              />
              <RowItem
                icon={<IconMicrochipOutline18 />}
                label="Resources"
                value={<Bar className="w-20" />}
              />
            </Rows>
          </Node>
        </Group>
        <HFlow empty />
        <Group icon={<IconGridOutline18 />} label="Services">
          <Node>
            <NodeHeader icon={<IconShieldKeyOutline18 />} name="Policies" />
            <Rows>
              <Bar className="w-3/4" />
              <Bar className="w-2/3" />
              <Bar className="w-1/2" />
            </Rows>
          </Node>
        </Group>
      </Canvas>
    </Card>
  );
}
