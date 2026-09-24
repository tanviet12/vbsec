import crypto from "crypto";
import { Router } from "express";
import { findUserByEmail, createResetToken } from "../services/userService";
import { sendMail } from "../services/mailer";

const router = Router();

router.post("/forgot-password", async (req, res) => {
  const email = String(req.body.email || "").trim().toLowerCase();
  const user = await findUserByEmail(email);
  if (user) {
    const token = crypto.randomBytes(32).toString("hex");
    await createResetToken(user.id, token);
    await sendMail(user.email, "Reset password", `https://app.example.com/reset?token=${token}`);
  }
  res.json({ ok: true });
});

export default router;
