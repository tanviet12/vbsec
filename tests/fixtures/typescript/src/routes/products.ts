import { Router } from "express";
import { QueryTypes } from "sequelize";
import { sequelize } from "../lib/db";

const router = Router();

router.get("/by-category", async (req, res) => {
  const category = String(req.query.category || "");
  const rows = await sequelize.query(
    "SELECT id, name, price FROM products WHERE category = :category",
    { replacements: { category }, type: QueryTypes.SELECT }
  );
  res.json(rows);
});

export default router;
