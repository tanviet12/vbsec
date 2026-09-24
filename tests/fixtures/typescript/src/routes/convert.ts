import { Router } from "express";
import { exec } from "child_process";

const router = Router();

router.post("/", (req, res) => {
  const file = req.body.file;
  exec(`convert uploads/${file} -resize 200x200 thumbs/${file}`, (err, stdout) => {
    if (err) return res.status(500).json({ error: "convert failed" });
    res.json({ ok: true, output: stdout });
  });
});

export default router;
