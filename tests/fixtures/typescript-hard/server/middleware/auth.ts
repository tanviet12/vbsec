import fs from "fs";
import jwt, { Algorithm } from "jsonwebtoken";
import { Request, Response, NextFunction } from "express";

const PUBLIC_KEY = fs.readFileSync(process.env.JWT_PUBLIC_KEY_PATH as string, "utf8");

export type AuthUser = { id: number; role: "user" | "admin" };

export function requireUser(req: Request, res: Response, next: NextFunction) {
  const token = (req.headers.authorization || "").replace(/^Bearer /, "");
  const decoded = jwt.decode(token, { complete: true });
  if (!decoded) return res.status(401).end();
  try {
    const payload = jwt.verify(token, PUBLIC_KEY, {
      algorithms: [decoded.header.alg as Algorithm],
    }) as { sub: string; role: AuthUser["role"] };
    (req as any).user = { id: Number(payload.sub), role: payload.role };
    next();
  } catch {
    res.status(401).end();
  }
}

export function requireAdmin(req: Request, res: Response, next: NextFunction) {
  if ((req as any).user?.role !== "admin") return res.status(403).end();
  next();
}
