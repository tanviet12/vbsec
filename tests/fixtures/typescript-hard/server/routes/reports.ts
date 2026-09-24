import { Router } from "express";
import { monthlyTotals } from "../services/reportService";

const router = Router();

router.get("/monthly", async (req, res) => {
  const month = String(req.query.month || "");
  if (!/^\d{4}-\d{2}$/.test(month)) return res.status(400).end();
  res.json(await monthlyTotals((req as any).user.id, month));
});

export default router;
