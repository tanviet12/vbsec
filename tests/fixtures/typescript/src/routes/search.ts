import { Router } from "express";
import { QueryTypes } from "sequelize";
import { sequelize } from "../lib/db";

const router = Router();

router.get("/", async (req, res) => {
  const q = req.query.q as string;
  const rows = await sequelize.query(
    `SELECT id, name, price FROM products WHERE name LIKE '%${q}%'`,
    { type: QueryTypes.SELECT }
  );
  res.json(rows);
});

export default router;
