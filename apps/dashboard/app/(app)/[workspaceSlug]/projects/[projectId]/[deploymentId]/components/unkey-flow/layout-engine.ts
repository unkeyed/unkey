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
          const childX =
            nodePosition.x + nodeDim.width / 2 + this.config.spacing.x;
          const childY = this.calculateChildYPosition(
            nodePosition.y,
            index,
            subtreeHeights
          );

          const childDim = this.dimensions.get(child.id);
          if (!childDim) {
            throw new Error(`Missing dimensions for child ${child.id}`);
          }

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
    // Children spread vertically - width is node + deepest child
    const maxChildWidth = Math.max(
      ...node.children.map((child) => this.calculateSubtreeWidth(child as T))
    );
    return nodeDim.width + this.config.spacing.x + maxChildWidth;
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
    parentY: number,
    childIndex: number,
    subtreeHeights: number[]
  ): number {
    const childCount = subtreeHeights.length;

    if (childCount === 1) {
      return parentY;
    }

    // Calculate total height needed for all subtrees + spacing
    const totalSubtreeHeight = subtreeHeights.reduce((sum, h) => sum + h, 0);
    const totalSpacing = (childCount - 1) * this.config.spacing.y;
    const totalHeight = totalSubtreeHeight + totalSpacing;

    // Start position (topmost subtree's top edge, centered on parent)
    const startY = parentY - totalHeight / 2;

    // Calculate this child's Y position (center of its subtree allocation)
    let y = startY;
    for (let i = 0; i < childIndex; i++) {
      y += subtreeHeights[i] + this.config.spacing.y;
    }
    y += subtreeHeights[childIndex] / 2;

    return y;
  }

  /**
   * Build connection lines between parent and child nodes.
   * @throws Error if child node position not found
   */
  private buildConnections(
    positioned: PositionedNode<T>[]
  ): Array<{ from: Point; to: Point; parent: T; child: T }> {
    const connections: Array<{
      from: Point;
      to: Point;
      parent: T;
      child: T;
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

          if (parentDirection === "horizontal") {
            // Horizontal: right edge of parent to left edge of child
            connections.push({
              from: {
                x: pos.position.x + parentDim.width / 2,
                y: pos.position.y,
              },
              to: {
                x: childPos.position.x - childDim.width / 2,
                y: childPos.position.y,
              },
              parent: pos.node,
              child: childPos.node as T,
            });
          } else {
            // Vertical: bottom edge of parent to top edge of child
            connections.push({
              from: {
                x: pos.position.x,
                y: pos.position.y + parentDim.height / 2,
              },
              to: {
                x: childPos.position.x,
                y: childPos.position.y - childDim.height / 2,
              },
              parent: pos.node,
              child: childPos.node as T,
            });
          }
        });
      }
    });

    return connections;
  }
}
