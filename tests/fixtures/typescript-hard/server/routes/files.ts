import path from "path";
import { Router } from "express";

const router = Router();
const PUBLIC_DIR = "/srv/wallet/public-docs";

router.get("/:name", (req, res) => {
  const raw = req.params.name;
  if (raw.includes("..") || raw.includes("/")) {
    return res.status(400).end();
  }
  const name = decodeURIComponent(raw);
  res.sendFile(path.join(PUBLIC_DIR, name));
});

export default router;
