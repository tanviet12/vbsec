import { Router } from "express";
import { User } from "../lib/db";
import { requireUser } from "../lib/auth";

const router = Router();

router.put("/", requireUser, async (req, res) => {
  const userId = (req as any).user.sub;
  await User.update(req.body, { where: { id: userId } });
  res.json({ ok: true });
});

export default router;
