import { Router } from "express";
import { prisma } from "../db";

const router = Router();

router.post("/", async (req, res) => {
  const senderId = (req as any).user.id as number;
  const amount = Number(req.body.amount);
  const receiverId = Number(req.body.receiverId);
  if (!Number.isInteger(amount) || amount <= 0) return res.status(400).end();

  const sender = await prisma.wallet.findUniqueOrThrow({ where: { userId: senderId } });
  if (sender.balance < amount) return res.status(409).json({ error: "insufficient balance" });

  await prisma.wallet.update({ where: { userId: senderId }, data: { balance: sender.balance - amount } });
  await prisma.wallet.update({ where: { userId: receiverId }, data: { balance: { increment: amount } } });
  await prisma.transfer.create({ data: { senderId, receiverId, amount } });
  res.json({ ok: true });
});

export default router;
