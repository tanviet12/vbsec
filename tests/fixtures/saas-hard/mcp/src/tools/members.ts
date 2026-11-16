import { z } from "zod";
import { seg } from "../backend.js";
import { defineTool, serverText, withExternal } from "../result.js";

interface Member {
  id: number;
  name: string;
  role: string;
  status: string;
}

const listMembers = defineTool({
  name: "list_members",
  title: "Team members",
  description: "List the people in the company with their role.",
  input: z.object({}),
  annotations: { readOnlyHint: true },
  run: async (_args, backend) => {
    const res = await backend.get<{ members: Member[] }>("/members");
    const rows = res.members.map((m) => ({ id: m.id, name: m.name, role: m.role, status: m.status }));
    return withExternal(`${rows.length} member(s).`, "members", rows);
  },
});

const setMemberRole = defineTool({
  name: "set_member_role",
  title: "Change role",
  description: "Change the role of a member.",
  input: z.object({
    member_id: z.number().int().positive(),
    role: z.enum(["admin", "accountant", "member", "client"]),
  }),
  run: async (args, backend) => {
    await backend.put(`/members/${seg(args.member_id)}/role`, { role: args.role });
    return serverText(`Member ${args.member_id} now has the role ${args.role}.`);
  },
});

export const memberTools = [listMembers, setMemberRole];
