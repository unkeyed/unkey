import { useCallback, useMemo, useRef } from "react";
import { LayoutEngine } from "../../layout-engine";
import type { Point } from "../../types";
import { type DeploymentNode, NODE_SIZES } from "../nodes/types";
import { TreeConnectionLine } from "./tree-connection-line";
import { TreeElementNode } from "./tree-element-node";

type TreeLayoutProps = {
  data: DeploymentNode;
  nodeSpacing?: { x: number; y: number };
  onNodeClick?: (node: DeploymentNode) => void;
  renderNode: (node: DeploymentNode, parent?: DeploymentNode) => React.ReactNode;
  renderConnection?: (
    path: Point[],
    parent: DeploymentNode,
    child: DeploymentNode,
  ) => React.ReactNode;
};

/**
 * Vertical tree layout component (top to bottom).
 * Uses fixed node dimensions for immediate layout calculation.
 */
export function TreeLayout({
  data,
  nodeSpacing = { x: 50, y: 50 },
  renderNode,
  renderConnection,
  onNodeClick,
}: TreeLayoutProps) {
  const containerRef = useRef<SVGGElement>(null);

  const layoutEngine = useMemo(
    () =>
      new LayoutEngine<DeploymentNode>({
        spacing: nodeSpacing,
        direction: "vertical",
      }),
    [nodeSpacing],
  );

  const parentMap = useMemo(() => {
    const map = new Map<string, DeploymentNode>();
    const buildMap = (node: DeploymentNode) => {
      if ("children" in node && node.children) {
        for (const child of node.children) {
          map.set(child.id, node);
          buildMap(child);
        }
      }
    };
    buildMap(data);
    return map;
  }, [data]);

  const allNodes = useMemo(() => {
    const flatten = (node: DeploymentNode): DeploymentNode[] => {
      const result: DeploymentNode[] = [node];
      if ("children" in node && node.children) {
        for (const child of node.children) {
          result.push(...flatten(child));
        }
      }
      return result;
    };
    return flatten(data);
  }, [data]);

  const handleClick = useCallback(
    (e: React.MouseEvent<SVGGElement>) => {
      if (!onNodeClick) {
        return;
      }
      let target = e.target as HTMLElement | SVGElement;
      while (target && target !== e.currentTarget) {
        const nodeId = target.getAttribute("data-node-id");
        if (nodeId) {
          const node = allNodes.find((n) => n.id === nodeId);
          if (node) {
            onNodeClick(node);
          }
          break;
        }
        target = target.parentElement as HTMLElement | SVGElement;
      }
    },
    [onNodeClick, allNodes],
  );

  // Set dimensions based on node type
  for (const node of allNodes) {
    const size = NODE_SIZES[node.metadata.type];
    layoutEngine.setNodeDimension(node.id, size);
  }

  const layout = useMemo(() => {
    return layoutEngine.calculate(data);
  }, [data, layoutEngine]);

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: This is required for event bubbling
    <g ref={containerRef} onClick={handleClick}>
      {layout.connections.map((conn) =>
        renderConnection ? (
          renderConnection(conn.path, conn.parent, conn.child)
        ) : (
          <TreeConnectionLine key={`${conn.parent.id}-${conn.child.id}`} path={conn.path} />
        ),
      )}
      {layout.nodes.map((positioned) => {
        const parent = parentMap.get(positioned.node.id);
        return (
          <TreeElementNode
            key={positioned.node.id}
            id={positioned.node.id}
            position={positioned.position}
          >
            {renderNode(positioned.node, parent)}
          </TreeElementNode>
        );
      })}
    </g>
  );
}
