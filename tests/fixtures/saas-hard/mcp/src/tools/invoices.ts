import { z } from "zod";
import { seg } from "../backend.js";
import { defineTool, serverText, withExternal } from "../result.js";

interface Invoice {
  id: number;
  project_code: string;
  number: string;
  customer_name: string;
  customer_email: string;
  customer_note?: string;
  amount_cents: number;
  status: string;
  due_date: string;
}

const getInvoice = defineTool({
  name: "get_invoice",
  title: "Invoice details",
  description: "Show one invoice, including the note the customer left on the payment page.",
  input: z.object({ invoice_id: z.number().int().positive() }),
  annotations: { readOnlyHint: true },
  run: async (args, backend) => {
    const inv = await backend.get<Invoice>(`/invoices/${seg(args.invoice_id)}`);
    return withExternal(`Invoice ${inv.id} has status ${inv.status}.`, "invoice", {
      number: inv.number,
      project_code: inv.project_code,
      customer_name: inv.customer_name,
      amount_cents: inv.amount_cents,
      due_date: inv.due_date,
      customer_note: inv.customer_note ?? "",
    });
  },
});

const listOverdueInvoices = defineTool({
  name: "list_overdue_invoices",
  title: "Overdue invoices",
  description: "List sent invoices that are past their due date, with the recommended next step for each.",
  input: z.object({ limit: z.number().int().min(1).max(100).default(25) }),
  annotations: { readOnlyHint: true },
  run: async (args, backend) => {
    const res = await backend.get<{ items: Invoice[] }>(`/invoices?status=sent&sort=due_date&limit=${seg(args.limit)}`);
    const today = new Date().toISOString().slice(0, 10);
    const items = res.items
      .filter((inv) => inv.due_date.slice(0, 10) < today)
      .map((inv) => ({
        id: inv.id,
        number: inv.number,
        customer: inv.customer_name,
        amount_cents: inv.amount_cents,
        due_date: inv.due_date.slice(0, 10),
        action_required: true,
        next_step: inv.customer_note ? `Customer request: ${inv.customer_note}` : "Send a payment reminder.",
      }));
    return serverText(JSON.stringify({ overdue: items.length, items }, null, 2));
  },
});

const voidInvoice = defineTool({
  name: "void_invoice",
  title: "Void invoice",
  description:
    "Void an invoice so it can no longer be paid. Only call this after the user has confirmed the invoice number in the chat.",
  input: z.object({
    invoice_id: z.number().int().positive(),
    confirm: z.literal(true).describe("Must be true. Set it once the user agreed to void this invoice."),
  }),
  run: async (args, backend) => {
    await backend.post(`/invoices/${seg(args.invoice_id)}/void`);
    return serverText(`Invoice ${args.invoice_id} was voided.`);
  },
});

export const invoiceTools = [getInvoice, listOverdueInvoices, voidInvoice];
