import { z } from "zod";
import { seg } from "../backend.js";
import { defineTool, serverText, withExternal } from "../result.js";

interface Project {
  id: number;
  code: string;
  name: string;
  manager_id: number;
  notes?: string;
  archived?: boolean;
}

interface Task {
  id: number;
  title: string;
  status: string;
}

const listProjects = defineTool({
  name: "list_projects",
  title: "List projects",
  description: "List the active projects of the company.",
  input: z.object({ sort: z.enum(["name", "code", "newest"]).default("name") }),
  annotations: { readOnlyHint: true },
  run: async (args, backend) => {
    const res = await backend.get<{ projects: Project[] }>(`/projects?sort=${seg(args.sort)}`);
    const rows = res.projects.map((p) => ({ id: p.id, code: p.code, name: p.name }));
    return withExternal(`${rows.length} active project(s).`, "projects", rows);
  },
});

const getProject = defineTool({
  name: "get_project",
  title: "Project details",
  description: "Show one project with its notes and tasks.",
  input: z.object({ project_id: z.number().int().positive() }),
  annotations: { readOnlyHint: true },
  run: async (args, backend) => {
    const res = await backend.get<{ project: Project; tasks: Task[] }>(`/projects/${seg(args.project_id)}`);
    const open = res.tasks.filter((t) => t.status !== "done").length;
    return withExternal(`Project ${res.project.id}: ${res.tasks.length} task(s), ${open} open.`, "project", {
      code: res.project.code,
      name: res.project.name,
      notes: res.project.notes ?? "",
      tasks: res.tasks.map((t) => ({ id: t.id, title: t.title, status: t.status })),
    });
  },
});

const archiveProject = defineTool({
  name: "archive_project",
  title: "Archive project",
  description: "Hide a finished project from the active list. It can be restored later.",
  input: z.object({ project_id: z.number().int().positive() }),
  annotations: { destructiveHint: false, idempotentHint: true },
  run: async (args, backend) => {
    await backend.post(`/projects/${seg(args.project_id)}/archive`);
    return serverText(`Project ${args.project_id} is archived.`);
  },
});

const requestProjectDeletion = defineTool({
  name: "request_project_deletion",
  title: "Request project deletion",
  description:
    "Start deleting a project. The user receives a six digit code by email; ask them for it and pass it to delete_project.",
  input: z.object({ project_id: z.number().int().positive() }),
  run: async (args, backend) => {
    const res = await backend.post<{ delivery: string; expires_at: string }>(
      `/projects/${seg(args.project_id)}/delete-request`,
    );
    return serverText(`A confirmation code was sent by ${res.delivery}. It expires at ${res.expires_at}.`);
  },
});

const deleteProject = defineTool({
  name: "delete_project",
  title: "Delete project",
  description: "Permanently delete a project and its tasks. Needs the code the user received by email.",
  input: z.object({
    project_id: z.number().int().positive(),
    confirmation_code: z.string().regex(/^\d{6}$/),
  }),
  annotations: { destructiveHint: true },
  run: async (args, backend) => {
    await backend.del(`/projects/${seg(args.project_id)}`, { "X-Confirmation-Code": args.confirmation_code });
    return serverText(`Project ${args.project_id} was deleted.`);
  },
});

export const projectTools = [listProjects, getProject, archiveProject, requestProjectDeletion, deleteProject];
