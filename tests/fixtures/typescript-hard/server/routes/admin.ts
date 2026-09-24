import { Router } from "express";
import { prisma } from "../db";
import { requireAdmin } from "../middleware/auth";

const router = Router();

router.get("/users", async (_req, res) => {
  res.json(await prisma.user.findMany({ select: { id: true, email: true, role: true } }));
});

router.get("/users/:id/transfers", async (req, res) => {
  res.json(await prisma.transfer.findMany({ where: { senderId: Number(req.params.id) } }));
});

router.use(requireAdmin);

router.post("/users/:id/ban", async (req, res) => {
  await prisma.user.update({ where: { id: Number(req.params.id) }, data: { banned: true } });
  res.json({ ok: true });
});

export default router;
