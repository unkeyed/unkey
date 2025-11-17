import type { TreeNode, Point, PositionedNode } from "./types";

/**
 * Dimensions of a rendered node in pixels
 */
type NodeDimensions = {
  width: number;
  height: number;
};

/**
 * Configuration for tree layout calculation
 */
type LayoutConfig = {
  /** Space between adjacent nodes */
  spacing: { x: number; y: number };
  /** Tree growth direction */
  direction: "vertical" | "horizontal";
};

/**
 * Complete layout calculation result containing positioned nodes and their connections
 */
type LayoutResult<T extends TreeNode> = {
  nodes: PositionedNode<T>[];
  connections: Array<{
    from: Point;
    to: Point;
    parent: T;
    child: T;
    waypoints?: Point[];
  }>;
};

/**
 * Pure layout calculation engine for tree structures.
 * Requires all node dimensions before calculating positions.
 * Throws immediately on missing data or invalid state.
 */
export class LayoutEngine<T extends TreeNode> {
  private config: LayoutConfig;
  private dimensions: Map<string, NodeDimensions>;

  constructor(config: LayoutConfig) {
    this.config = config;
    this.dimensions = new Map();
  }

  /**
   * Register measured dimensions for a node.
   * Must be called for all nodes before calculate().
   */
  setNodeDimension(id: string, dimensions: NodeDimensions): void {
    this.dimensions.set(id, dimensions);
  }

  /**
   * Check if all nodes in the tree have registered dimensions.
   */
  hasAllDimensions(root: T): boolean {
    const allNodes = this.flattenTree(root);
    return allNodes.every((node) => this.dimensions.has(node.id));
  }

  /**
   * Calculate absolute positions for all nodes and their connections.
   * @throws Error if any node is missing dimensions
   */
  calculate(root: T): LayoutResult<T> {
    if (!this.hasAllDimensions(root)) {
      throw new Error(
        `Cannot calculate layout: missing dimensions for some nodes. Have ${this.dimensions.size} dimensions.`
      );
    }

    const positioned = this.layoutNode(root, 0, { x: 0, y: 0 });
    const connections = this.buildConnections(positioned);

    return { nodes: positioned, connections };
  }

  clear(): void {
    this.dimensions.clear();
  }

  /**
   * Flatten tree into linear array using depth-first traversal
   */
  private flattenTree(root: T): T[] {
    const result: T[] = [root];
    if (root.children) {
      root.children.forEach((child) => {
        result.push(...this.flattenTree(child as T));
      });
    }
    return result;
  }

  private getNodeDirection(node: T): "vertical" | "horizontal" {
    return node.direction ?? this.config.direction;
  }

  /**
   * Recursively layout a node and its children, respecting per-node direction.
   */
  private layoutNode(
    node: T,
    level: number,
    parentPosition: Point
  ): PositionedNode<T>[] {
    const positioned: PositionedNode<T>[] = [];

    const nodeDim = this.dimensions.get(node.id);
    if (!nodeDim) {
      throw new Error(`Missing dimensions for node ${node.id}`);
    }

    // Position this node
    const nodePosition: Point = level === 0 ? { x: 0, y: 0 } : parentPosition;

    positioned.push({
      node,
      position: nodePosition,
      level,
    });

    // Layout children if they exist
    if (node.children && node.children.length > 0) {
      const childDirection = this.getNodeDirection(node);

      if (childDirection === "vertical") {
        // Children spread horizontally below parent
        const subtreeWidths = node.children.map((child) =>
          this.calculateSubtreeWidth(child as T)
        );

        node.children.forEach((child, index) => {
          const childX = this.calculateChildXPosition(
            nodePosition.x,
            index,
            subtreeWidths
          );
          const childY =
            nodePosition.y + nodeDim.height / 2 + this.config.spacing.y;

          const childDim = this.dimensions.get(child.id);
          if (!childDim) {
            throw new Error(`Missing dimensions for child ${child.id}`);
          }

          const childPositioned = this.layoutNode(child as T, level + 1, {
            x: childX,
            y: childY + childDim.height / 2,
          });
          positioned.push(...childPositioned);
        });
      } else {
        // Children spread vertically to the right of parent (top to bottom)
        const subtreeHeights = node.children.map((child) =>
          this.calculateSubtreeHeight(child as T)
        );

        node.children.forEach((child, index) => {
          const childDim = this.dimensions.get(child.id);
          if (!childDim) {
            throw new Error(`Missing dimensions for child ${child.id}`);
          }

          const childX = nodePosition.x - nodeDim.width / 2;
          const childY = this.calculateChildYPosition(
            nodePosition.y + nodeDim.height / 2 + this.config.spacing.y,
            index,
            subtreeHeights
          );

          const childPositioned = this.layoutNode(child as T, level + 1, {
            x: childX + childDim.width / 2,
            y: childY,
          });
          positioned.push(...childPositioned);
        });
      }
    }

    return positioned;
  }

  /**
   * Calculate the horizontal width required for each node's entire subtree.
   * Width includes the node itself plus all descendants laid out horizontally.
   */
  private calculateSubtreeWidth(node: T): number {
    const nodeDim = this.dimensions.get(node.id);
    if (!nodeDim) {
      throw new Error(`Missing dimensions for node ${node.id}`);
    }

    if (!node.children || node.children.length === 0) {
      return nodeDim.width;
    }

    const childDirection = this.getNodeDirection(node);

    if (childDirection === "vertical") {
      // Children spread horizontally
      const childWidths = node.children.map((child) =>
        this.calculateSubtreeWidth(child as T)
      );
      const totalChildWidth = childWidths.reduce((sum, w) => sum + w, 0);
      const spacing = (node.children.length - 1) * this.config.spacing.x;
      return Math.max(nodeDim.width, totalChildWidth + spacing);
    }
    // Children spread vertically - width is node + deepest child, but allow overlap
    const childSubtreeWidths = node.children.map((child) =>
      this.calculateSubtreeWidth(child as T)
    );
    const maxChildWidth = Math.max(...childSubtreeWidths);
    // Use a percentage of the child width to allow controlled overlap
    const overlapFactor = 0.4; // Children's horizontal subtrees can overlap by 60%
    return (
      nodeDim.width + this.config.spacing.x + maxChildWidth * overlapFactor
    );
  }

  /**
   * Calculate the vertical height required for each node's entire subtree.
   * Height includes the node itself plus all descendants laid out vertically.
   */
  private calculateSubtreeHeight(node: T): number {
    const nodeDim = this.dimensions.get(node.id);
    if (!nodeDim) {
      throw new Error(`Missing dimensions for node ${node.id}`);
    }

    if (!node.children || node.children.length === 0) {
      return nodeDim.height;
    }

    const childDirection = this.getNodeDirection(node);

    if (childDirection === "horizontal") {
      // Children spread vertically (top to bottom)
      const childHeights = node.children.map((child) =>
        this.calculateSubtreeHeight(child as T)
      );
      const totalChildHeight = childHeights.reduce((sum, h) => sum + h, 0);
      const spacing = (node.children.length - 1) * this.config.spacing.y;
      return Math.max(nodeDim.height, totalChildHeight + spacing);
    }
    // Children spread horizontally - height is node + deepest child
    const maxChildHeight = Math.max(
      ...node.children.map((child) => this.calculateSubtreeHeight(child as T))
    );
    return nodeDim.height + this.config.spacing.y + maxChildHeight;
  }

  /**
   * Calculate X position for a child node based on subtree widths.
   * Uses each child's subtree width (not just node width) for spacing.
   */
  private calculateChildXPosition(
    parentX: number,
    childIndex: number,
    subtreeWidths: number[]
  ): number {
    const childCount = subtreeWidths.length;

    if (childCount === 1) {
      return parentX;
    }

    // Calculate total width needed for all subtrees + spacing
    const totalSubtreeWidth = subtreeWidths.reduce((sum, w) => sum + w, 0);
    const totalSpacing = (childCount - 1) * this.config.spacing.x;
    const totalWidth = totalSubtreeWidth + totalSpacing;

    // Start position (leftmost subtree's left edge)
    const startX = parentX - totalWidth / 2;

    // Calculate this child's X position (center of its subtree allocation)
    let x = startX;
    for (let i = 0; i < childIndex; i++) {
      x += subtreeWidths[i] + this.config.spacing.x;
    }
    x += subtreeWidths[childIndex] / 2;

    return x;
  }

  /**
   * Calculate Y position for a child node based on subtree heights.
   * Uses each child's subtree height (not just node height) for spacing.
   * Children stack top to bottom starting from parent's Y position.
   */
  private calculateChildYPosition(
    startY: number,
    childIndex: number,
    subtreeHeights: number[]
  ): number {
    let y = startY;

    // Add half of first child's height to get to its center
    y += subtreeHeights[0] / 2;

    // Add up heights and spacing for children before this one
    for (let i = 0; i < childIndex; i++) {
      y +=
        subtreeHeights[i] / 2 +
        this.config.spacing.y +
        subtreeHeights[i + 1] / 2;
    }

    return y;
  }

  /**
   * Build connection lines between parent and child nodes.
   * @throws Error if child node position not found
   */
  private buildConnections(positioned: PositionedNode<T>[]): Array<{
    from: Point;
    to: Point;
    parent: T;
    child: T;
    waypoints?: Point[];
  }> {
    const connections: Array<{
      from: Point;
      to: Point;
      parent: T;
      child: T;
      waypoints?: Point[];
    }> = [];

    positioned.forEach((pos) => {
      if (pos.node.children) {
        const parentDirection = this.getNodeDirection(pos.node);

        pos.node.children.forEach((child) => {
          const childPos = positioned.find((p) => p.node.id === child.id);
          if (!childPos) {
            throw new Error(
              `Cannot find positioned node for child ${child.id}`
            );
          }

          const parentDim = this.dimensions.get(pos.node.id);
          const childDim = this.dimensions.get(child.id);

          if (!parentDim || !childDim) {
            throw new Error(
              `Missing dimensions for connection: ${pos.node.id} -> ${child.id}`
            );
          }

          const parentEdges = getNodeEdges(pos.position, parentDim);
          const childEdges = getNodeEdges(childPos.position, childDim);

          if (parentDirection === "horizontal") {
            // Vertical trunk with horizontal branches layout:
            // Parent ───┐
            //           │ (trunk goes down)
            //           ├──→ Child1
            //           ├──→ Child2
            //           └──→ Child3

            // Calculate X positions
            const trunkOffsetFromParent = 100; // How far left the trunk extends from parent
            const trunkX = parentEdges.left - trunkOffsetFromParent; // Absolute X position of the vertical trunk line

            connections.push({
              from: { x: parentEdges.left, y: pos.position.y }, // Start at parent's left edge center
              to: { x: childEdges.left, y: childPos.position.y }, // End at child's left edge center
              parent: pos.node,
              child: childPos.node as T,
              waypoints: [
                { x: trunkX, y: pos.position.y }, // Go left to trunk, stay at parent's Y (horizontal segment)
                { x: trunkX, y: childPos.position.y }, // Go down trunk to child's Y level (vertical segment, creates rounded corner)
                // Final segment from last waypoint to 'to' goes right to child (horizontal, creates rounded corner)
              ],
            });
          } else {
            // Vertical Z-shaped path with rounded corners:
            // Parent
            //   │ (go down)
            //   └──╮ (turn right with curve)
            //      │ (go across)
            //   ╭──┘ (turn down with curve)
            //   │ (go down)
            // Child

            // Calculate Y positions
            const verticalGap = childEdges.top - parentEdges.bottom; // Total vertical distance between parent bottom and child top
            const midY = parentEdges.bottom + verticalGap * 0.5; // Midpoint Y where horizontal segment occurs (halfway between parent and child)

            connections.push({
              from: {
                x: pos.position.x, // Start at parent's center X
                y: parentEdges.bottom, // Start at parent's bottom edge
              },
              to: {
                x: childPos.position.x, // End at child's center X
                y: childEdges.top, // End at child's top edge
              },
              parent: pos.node,
              child: childPos.node as T,
              waypoints: [
                { x: pos.position.x, y: midY }, // Go down to midpoint, stay at parent's X (vertical segment)
                { x: childPos.position.x, y: midY }, // Go horizontally to child's X at midpoint (horizontal segment, creates rounded corner)
                // Final segment from last waypoint to 'to' goes down to child (vertical, creates rounded corner)
              ],
            });
          }
        });
      }
    });

    return connections;
  }
}

// Add this helper function at the top of the file or in a shared utils
function getNodeEdges(
  position: Point,
  dimensions: { width: number; height: number }
) {
  return {
    left: position.x - dimensions.width / 2,
    right: position.x + dimensions.width / 2,
    top: position.y - dimensions.height / 2,
    bottom: position.y + dimensions.height / 2,
  };
}
