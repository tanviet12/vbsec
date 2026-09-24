import jwt from "jsonwebtoken";
import { Request, Response, NextFunction } from "express";

export function requireUser(req: Request, res: Response, next: NextFunction) {
  const token = (req.headers.authorization || "").replace("Bearer ", "");
  const payload = jwt.decode(token) as { sub: string; role: string } | null;
  if (!payload) return res.status(401).end();
  (req as any).user = payload;
  next();
}
