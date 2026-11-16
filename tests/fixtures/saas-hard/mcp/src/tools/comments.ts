import { z } from "zod";
import { seg } from "../backend.js";
import { defineTool, serverText } from "../result.js";

interface Comment {
  id: number;
  author_name: string;
  author_role: string;
  body: string;
  visibility: "team" | "client";
  created_at: string;
}

const listTaskComments = defineTool({
  name: "list_task_comments",
  title: "Task discussion",
  description: "Read the discussion on a task, oldest first. Includes messages from client guests.",
  input: z.object({ task_id: z.number().int().positive() }),
  annotations: { readOnlyHint: true },
  run: async (args, backend) => {
    const res = await backend.get<{ comments: Comment[] }>(`/tasks/${seg(args.task_id)}/comments`);
    const thread = res.comments.map((c) => ({
      from: c.author_name,
      role: c.author_role,
      shared_with_client: c.visibility === "client",
      at: c.created_at,
      message: c.body,
    }));
    return serverText(JSON.stringify({ task_id: args.task_id, thread }, null, 2));
  },
});

const postComment = defineTool({
  name: "post_comment",
  title: "Post comment",
  description:
    "Post a comment on a task. Use visibility \"client\" to share the update with the client guests of the project.",
  input: z.object({
    task_id: z.number().int().positive(),
    body: z.string().min(1).max(4000),
    visibility: z.enum(["team", "client"]).default("team"),
  }),
  run: async (args, backend) => {
    const created = await backend.post<{ id: number; visibility: string }>(`/tasks/${seg(args.task_id)}/comments`, {
      body: args.body,
      visibility: args.visibility,
    });
    return serverText(`Comment ${created.id} posted to the ${created.visibility} thread.`);
  },
});

export const commentTools = [listTaskComments, postComment];
