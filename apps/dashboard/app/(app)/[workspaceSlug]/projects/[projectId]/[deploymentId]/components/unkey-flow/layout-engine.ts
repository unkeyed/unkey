import type { Point, PositionedNode, TreeNode } from "./types";

/**
 * Dimensions of a rendered node in pixels
 */
type NodeDimensions = {
  width: number;
  height: number;
};

/**
 * Connection line between parent and child nodes
 */
type Connection<T extends TreeNode> = {
  parent: T;
  child: T;
  path: Point[];
};

/**
 * Configuration for tree layout calculation
 */
type LayoutConfig = {
  /** Space between adjacent nodes */
  spacing: {
    x: number;
    y: number;
  };

  /** Tree growth direction */
  direction: "vertical" | "horizontal";

  /** Layout tuning for different orientations */
  layout?: {
    /** Horizontal indent from parent to children (horizontal layout only) */
    horizontalIndent?: number; // default: 60

    /** Vertical offset from calculated child position (horizontal layout only) */
    verticalOffset?: number; // default: -25

    /** How tightly subtrees pack together (0-1, lower = more overlap) */
    subtreeOverlap?: number; // default: 0.4

    /** Sibling spacing multiplier for horizontal layout (e.g., 0.833 makes siblings closer) */
    horizontalSiblingSpacing?: number; // default: 0.833 (equivalent to current / 1.2)
  };

  /** Connection line configuration */
  connections?: {
    /** Horizontal layout (trunk + branches) */
    horizontal?: {
      /** Distance trunk extends from parent's left edge */
      trunkOffset?: number; // default: 0

      /** Additional offset for trunk positioning */
      trunkAdjust?: number; // default: 20
    };
  };
};

/**
 * Complete layout calculation result containing positioned nodes and their connections
 */
type LayoutResult<T extends TreeNode> = {
  nodes: PositionedNode<T>[];
  connections: Connection<T>[];
};

/**
 * Pure layout calculation engine for tree structures.
 * Requires all node dimensions before calculating positions.
 * Throws immediately on missing data or invalid state.
 */
export class LayoutEngine<T extends TreeNode> {
  private config: Required<LayoutConfig> & {
    layout: Required<NonNullable<LayoutConfig["layout"]>>;
    connections: Required<NonNullable<LayoutConfig["connections"]>> & {
      horizontal: Required<NonNullable<NonNullable<LayoutConfig["connections"]>["horizontal"]>>;
    };
  };
  private dimensions: Map<string, NodeDimensions>;

  constructor(config: LayoutConfig) {
    this.config = {
      spacing: config.spacing,
      direction: config.direction,
      layout: {
        horizontalIndent: config.layout?.horizontalIndent ?? 60,
        verticalOffset: config.layout?.verticalOffset ?? -25,
        subtreeOverlap: config.layout?.subtreeOverlap ?? 0.4,
        horizontalSiblingSpacing: config.layout?.horizontalSiblingSpacing ?? 0.833,
      },
      connections: {
        horizontal: {
          trunkOffset: config.connections?.horizontal?.trunkOffset ?? 0,
          trunkAdjust: config.connections?.horizontal?.trunkAdjust ?? 20,
        },
      },
    };

    // Validate
    invariant(
      this.config.layout.horizontalIndent !== undefined,
      "Layout horizontalIndent must be defined",
    );
    invariant(
      this.config.layout.verticalOffset !== undefined,
      "Layout verticalOffset must be defined",
    );
    invariant(
      this.config.layout.subtreeOverlap !== undefined,
      "Layout subtreeOverlap must be defined",
    );
    invariant(
      this.config.layout.horizontalSiblingSpacing !== undefined,
      "Layout horizontalSiblingSpacing must be defined",
    );
    invariant(
      this.config.connections.horizontal.trunkAdjust !== undefined,
      "Connection trunkAdjust must be defined",
    );

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
    invariant(
      this.hasAllDimensions(root),
      `Cannot calculate layout: missing dimensions for some nodes. Have ${this.dimensions.size} dimensions.`,
    );

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
  private layoutNode(node: T, level: number, parentPosition: Point): PositionedNode<T>[] {
    const positioned: PositionedNode<T>[] = [];

    const nodeDim = this.dimensions.get(node.id);
    invariant(nodeDim, `Missing dimensions for node ${node.id}`);

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
        const subtreeWidths = node.children.map((child) => this.calculateSubtreeWidth(child as T));

        node.children.forEach((child, index) => {
          const childX = this.calculateChildXPosition(nodePosition.x, index, subtreeWidths);
          const childY = nodePosition.y + nodeDim.height / 2 + this.config.spacing.y;

          const childDim = this.dimensions.get(child.id);
          invariant(childDim, `Missing dimensions for child ${child.id}`);

          const childPositioned = this.layoutNode(child as T, level + 1, {
            x: childX,
            y: childY + childDim.height / 2,
          });
          positioned.push(...childPositioned);
        });
      } else {
        // Children spread vertically to the right of parent (top to bottom)
        const subtreeHeights = node.children.map((child) =>
          this.calculateSubtreeHeight(child as T),
        );

        node.children.forEach((child, index) => {
          const childDim = this.dimensions.get(child.id);
          invariant(childDim, `Missing dimensions for child ${child.id}`);

          const childX = nodePosition.x - nodeDim.width / 2;
          const childY = this.calculateChildYPosition(
            nodePosition.y + nodeDim.height / 2 + this.config.spacing.y,
            index,
            subtreeHeights,
          );

          const childPositioned = this.layoutNode(child as T, level + 1, {
            x: childX + childDim.width / 2 + this.config.layout.horizontalIndent,
            y: childY + this.config.layout.verticalOffset,
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
    invariant(nodeDim, `Missing dimensions for node ${node.id}`);

    if (!node.children || node.children.length === 0) {
      return nodeDim.width;
    }

    const childDirection = this.getNodeDirection(node);

    if (childDirection === "vertical") {
      // Children spread horizontally
      const childWidths = node.children.map((child) => this.calculateSubtreeWidth(child as T));
      const totalChildWidth = childWidths.reduce((sum, w) => sum + w, 0);
      const spacing = (node.children.length - 1) * this.config.spacing.x;
      return Math.max(nodeDim.width, totalChildWidth + spacing);
    }

    // Children spread vertically - width is node + deepest child, but allow overlap
    const childSubtreeWidths = node.children.map((child) => this.calculateSubtreeWidth(child as T));
    const maxChildWidth = Math.max(...childSubtreeWidths);
    return (
      nodeDim.width + this.config.spacing.x + maxChildWidth * this.config.layout.subtreeOverlap
    );
  }

  /**
   * Calculate the vertical height required for each node's entire subtree.
   * Height includes the node itself plus all descendants laid out vertically.
   */
  private calculateSubtreeHeight(node: T): number {
    const nodeDim = this.dimensions.get(node.id);
    invariant(nodeDim, `Missing dimensions for node ${node.id}`);

    if (!node.children || node.children.length === 0) {
      return nodeDim.height;
    }

    const childDirection = this.getNodeDirection(node);

    if (childDirection === "horizontal") {
      // Children spread vertically (top to bottom)
      const childHeights = node.children.map((child) => this.calculateSubtreeHeight(child as T));
      const totalChildHeight = childHeights.reduce((sum, h) => sum + h, 0);
      const spacing = (node.children.length - 1) * this.config.spacing.y;
      return Math.max(nodeDim.height, totalChildHeight + spacing);
    }

    // Children spread horizontally - height is node + deepest child
    const maxChildHeight = Math.max(
      ...node.children.map((child) => this.calculateSubtreeHeight(child as T)),
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
    subtreeWidths: number[],
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
    subtreeHeights: number[],
  ): number {
    let y = startY;

    // Add half of first child's height to get to its center
    y += subtreeHeights[0] / 2;

    // Add up heights and spacing for children before this one
    for (let i = 0; i < childIndex; i++) {
      y +=
        subtreeHeights[i] / 2 +
        this.config.spacing.y * this.config.layout.horizontalSiblingSpacing +
        subtreeHeights[i + 1] / 2;
    }

    return y;
  }

  /**
   * Build connection lines between parent and child nodes.
   * Path is an ordered array of points from parent to child.
   */
  private buildConnections(positioned: PositionedNode<T>[]): Connection<T>[] {
    const connections: Connection<T>[] = [];
    const posMap = new Map(positioned.map((p) => [p.node.id, p]));

    for (const pos of positioned) {
      if (!pos.node.children) {
        continue;
      }

      const parentDim = this.dimensions.get(pos.node.id);
      invariant(
        parentDim,
        `Parent dimensions cannot be empty or undefined for node ${pos.node.id}`,
      );
      const parentEdges = getNodeEdges(pos.position, parentDim);
      const parentDirection = this.getNodeDirection(pos.node);

      for (const child of pos.node.children) {
        const childPos = posMap.get(child.id);
        invariant(childPos, `Cannot find positioned node for child ${child.id}`);

        const childDim = this.dimensions.get(child.id);
        invariant(childDim, `Child dimensions cannot be empty or undefined for node ${child.id}`);
        const childEdges = getNodeEdges(childPos.position, childDim);

        const path = this.buildConnectionPath(
          pos.position,
          parentEdges,
          childPos.position,
          childEdges,
          parentDirection,
        );

        connections.push({
          parent: pos.node,
          child: childPos.node as T,
          path,
        });
      }
    }

    return connections;
  }

  /**
   * Build a connection path between parent and child nodes.
   * Returns an ordered array of points that forms the visual connection.
   *
   * Horizontal layout (trunk-and-branch):
   * ```
   * Parent ───┐
   *           │ (vertical trunk)
   *           ├──→ Child1
   *           ├──→ Child2
   *           └──→ Child3
   * ```
   * Path: parent's left edge → trunk X → down to child Y → child's left edge
   *
   * Vertical layout (Z-shape):
   * ```
   * Parent
   *   │ (go down)
   *   └──╮ (turn right)
   *      │ (go across)
   *   ╭──┘ (turn down)
   *   │
   * Child
   * ```
   * Path: parent's bottom → halfway down → across to child X → child's top
   *
   * @param parentPos - Parent node's center position
   * @param parentEdges - Parent node's edge coordinates (left, right, top, bottom)
   * @param childPos - Child node's center position
   * @param childEdges - Child node's edge coordinates
   * @param direction - Layout direction determining path shape
   * @returns Array of points forming the connection path, ordered from parent to child
   */
  private buildConnectionPath(
    parentPos: Point,
    parentEdges: ReturnType<typeof getNodeEdges>,
    childPos: Point,
    childEdges: ReturnType<typeof getNodeEdges>,
    direction: "vertical" | "horizontal",
  ): Point[] {
    if (direction === "horizontal") {
      // Horizontal layout uses a vertical trunk with horizontal branches
      // This creates a "tree" appearance where siblings share a common trunk

      // Calculate trunk X position: offset from parent's left edge
      const trunkX =
        parentEdges.left -
        this.config.connections.horizontal.trunkOffset +
        this.config.connections.horizontal.trunkAdjust;

      // Four-point path:
      // 1. Start at parent's left edge (at parent's vertical center)
      // 2. Move left to trunk position (horizontal line)
      // 3. Move down to child's vertical position (vertical trunk)
      // 4. Move right to child's left edge (horizontal branch)
      return [
        { x: trunkX, y: parentPos.y },
        { x: trunkX, y: parentPos.y },
        { x: trunkX, y: childPos.y },
        { x: childEdges.left, y: childPos.y },
      ];
    }

    // Vertical layout uses a Z-shaped path
    // This provides clear visual separation between parent and child

    // Calculate midpoint: halfway between parent's bottom and child's top
    const verticalGap = childEdges.top - parentEdges.bottom;
    const midY = parentEdges.bottom + verticalGap * 0.5;

    // Four-point path:
    // 1. Start at parent's bottom edge (at parent's horizontal center)
    // 2. Move down to midpoint (vertical line)
    // 3. Move across to child's horizontal position (horizontal line)
    // 4. Move down to child's top edge (vertical line)
    return [
      { x: parentPos.x, y: parentEdges.bottom },
      { x: parentPos.x, y: midY },
      { x: childPos.x, y: midY },
      { x: childPos.x, y: childEdges.top },
    ];
  }
}

function getNodeEdges(position: Point, dimensions: { width: number; height: number }) {
  return {
    left: position.x - dimensions.width / 2,
    right: position.x + dimensions.width / 2,
    top: position.y - dimensions.height / 2,
    bottom: position.y + dimensions.height / 2,
  };
}

export function invariant(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}
