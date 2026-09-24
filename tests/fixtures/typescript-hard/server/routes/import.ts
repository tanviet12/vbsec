import { Router } from "express";

const router = Router();

router.post("/statement", async (req, res) => {
  let url: URL;
  try {
    url = new URL(String(req.body.url));
  } catch {
    return res.status(400).json({ error: "invalid url" });
  }
  if (url.protocol !== "https:" || !url.hostname.endsWith("example.com")) {
    return res.status(400).json({ error: "host not allowed" });
  }
  const resp = await fetch(url);
  const text = await resp.text();
  res.json({ rows: text.split("\n").length, preview: text.slice(0, 200) });
});

export default router;
