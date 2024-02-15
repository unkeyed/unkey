"use client";
import ReactFlow, {
  ConnectionLineType,
  Edge,
  Handle,
  Node,
  Position,
  NodeProps,
  useEdgesState,
  useNodesState,
  Controls,
  Background,
  BackgroundVariant,
} from "reactflow";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { Permission, Role } from "@unkey/db";
import dagre from "dagre";
import { KeySquare, MoreVertical, Settings2 } from "lucide-react";
import Link from "next/link";
import { PropsWithChildren, useCallback } from "react";
import "reactflow/dist/style.css";
import { RoleToggle } from "./role-toggle";

type Props = {
  roles: Array<{
    id: string;
    name: string;
    active: boolean;
    permissions: Array<{ permissionId: string }>;
  }>;
  permissions: Array<{ id: string; name: string; active: boolean }>;
  data: {
    id: string;
    name: string | null;
    keyAuthId: string;
    start: string;
    roles: {
      role: {
        id: string;
        permissions: {
          permission: {
            id: string;
          };
        }[];
      };
    }[];
  };
};

const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 300;
const nodeHeight = 36;
const getLayoutedElements = (nodes: Node[], edges: Edge[], direction = "LR") => {
  const isHorizontal = direction === "LR";
  dagreGraph.setGraph({ rankdir: direction });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  nodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.targetPosition = isHorizontal ? Position.Left : Position.Top;
    node.sourcePosition = isHorizontal ? Position.Right : Position.Bottom;

    // We are shifting the dagre node position (anchor=center center) to the top left
    // so it matches the React Flow node anchor point (top left).
    node.position = {
      x: nodeWithPosition.x - nodeWidth / 2,
      y: nodeWithPosition.y - nodeHeight / 2,
    };

    return node;
  });

  return { nodes, edges };
};

// will get updated automatically
const position = { x: 0, y: 0 };

export const Chart: React.FC<Props> = ({ data, roles, permissions }) => {
  const initialNodes: Node[] = [
    {
      id: data.id,
      position: { x: 0, y: 0 },
      data: { id: data.id, name: data.name, start: data.start, keyAuthid: data.keyAuthId },
      type: "key",
    },
  ];
  const initialEdges: Edge[] = [];

  for (const role of roles) {
    initialNodes.push({
      id: role.id,

      type: "role",
      position,
      data: {
        keyId: data.id,
        roleId: role.id,
        name: role.name,
        active: role.active,
      },
    });
    initialEdges.push({
      id: `${data.id}-${role.id}`,
      source: data.id,
      target: role.id,
      style: role.active ? { strokeWidth: 2, stroke: "#0239FC" } : undefined,
    });
    for (const permission of role.permissions) {
      initialEdges.push({
        id: `${role.id}-${permission.permissionId}`,
        source: role.id,
        target: permission.permissionId,
        style: role.active ? { strokeWidth: 2, stroke: "#0239FC" } : undefined,
      });
    }
  }
  for (const permission of permissions) {
    initialNodes.push({
      type: "permission",
      id: permission.id,
      position,
      data: {
        active: permission.active,
        name: permission.name,
        permissionId: permission.id,
      },
    });
  }

  const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
    initialNodes,
    initialEdges,
  );

  const [nodes, _setNodes, _onNodesChange] = useNodesState(layoutedNodes);
  const [edges, _setEdges, _onEdgesChange] = useEdgesState(layoutedEdges);
  // const onLayout = useCallback(
  //   (direction) => {
  //     const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
  //       nodes,
  //       edges,
  //       direction,
  //     );

  //     setNodes([...layoutedNodes]);
  //     setEdges([...layoutedEdges]);
  //   },
  //   [nodes, edges],
  // );

  //  [
  //   { id: "1", position: { x: 0, y: 0 }, data: { label: "1" } },
  //   { id: "2", position: { x: 0, y: 100 }, data: { label: "2" } },
  // ];

  return (
    <Card className="relative">
      <CardHeader className="absolute">
        <CardTitle>Permissions</CardTitle>
      </CardHeader>
      <div className="w-full h-[70vh]">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          connectionLineType={ConnectionLineType.SimpleBezier}
          nodeTypes={{ key: KeyNode, role: RoleNode, permission: PermissionNode }}
          proOptions={{
            hideAttribution: true,
          }}
        >
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>
    </Card>
  );
};

const KeyNode: React.FC<
  NodeProps<{ name?: string; id: string; start: string; keyAuthId: string }>
> = ({ data }) => {
  return (
    <NodeShell active>
      <div className="flex items-center gap-2 ml-2">
        <span className="text-sm truncate text-content text-ellipsis">{data.name ?? data.id}</span>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuGroup>
              <Link href={`/app/keys/${data.keyAuthId}/${data.id}`}>
                <DropdownMenuItem className="cursor-pointer">
                  <Settings2 className="w-4 h-4 mr-2" />
                  <span>Edit Key</span>
                </DropdownMenuItem>
              </Link>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <Handle type="source" position={Position.Right} className="!bg-content-subtle" />
    </NodeShell>
  );
};

const RoleNode: React.FC<
  NodeProps<{ name: string; active: boolean; roleId: string; keyId: string }>
> = ({ data }) => {
  return (
    <NodeShell active={data.active}>
      <div className="flex items-center gap-2 ml-2">
        <RoleToggle keyId={data.keyId} roleId={data.roleId} checked={data.active} />

        <span
          className={cn("font-mono text-content-subtle text-sm truncate text-ellipsis", {
            "text-content": data.active,
          })}
        >
          {data.name}
        </span>
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon">
            <MoreVertical className="w-4 h-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuGroup>
            <Link href={`/app/authorization/roles/${data.roleId}`}>
              <DropdownMenuItem className="cursor-pointer">
                <Settings2 className="w-4 h-4 mr-2" />
                <span>Edit Role</span>
              </DropdownMenuItem>
            </Link>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <Handle type="target" position={Position.Left} className="!bg-content-subtle" />
      <Handle type="source" position={Position.Right} className="!bg-content-subtle" />
    </NodeShell>
  );
};

const PermissionNode: React.FC<NodeProps<{ name: string; active: boolean; permissionId: string }>> =
  ({ data }) => {
    return (
      <NodeShell active={data.active}>
        <div className="flex items-center gap-2 ml-2 shrink">
          <span
            className={cn("font-mono text-content-subtle text-sm truncate text-ellipsis", {
              "text-content": data.active,
            })}
          >
            {data.name}
          </span>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuGroup>
              <Link href={`/app/authorization/permissions/${data.permissionId}`}>
                <DropdownMenuItem className="cursor-pointer">
                  <Settings2 className="w-4 h-4 mr-2" />
                  <span>Edit Permission</span>
                </DropdownMenuItem>
              </Link>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Handle type="target" position={Position.Left} className="!bg-content-subtle" />
      </NodeShell>
    );
  };

const NodeShell: React.FC<PropsWithChildren<{ active: boolean }>> = ({ active, children }) => (
  <div
    className={cn("p-2 border rounded-md bg-background w-[250px] overflow-hidden cursor-auto", {
      "border-[#0239FC] ": active,
    })}
  >
    {children}
  </div>
);
