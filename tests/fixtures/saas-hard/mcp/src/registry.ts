import type { ToolDef } from "./result.js";
import { commentTools } from "./tools/comments.js";
import { invoiceTools } from "./tools/invoices.js";
import { memberTools } from "./tools/members.js";
import { projectTools } from "./tools/projects.js";

export const allTools: ToolDef[] = [...projectTools, ...invoiceTools, ...commentTools, ...memberTools];

// Since 0.4 role changes are done in the web app only, where the owner
// re-enters their password. They are no longer part of the catalogue.
const webOnly = new Set(["set_member_role"]);

export function catalogue(): ToolDef[] {
  return allTools.filter((tool) => !webOnly.has(tool.name));
}

export function findTool(name: string): ToolDef | undefined {
  return allTools.find((tool) => tool.name === name);
}
