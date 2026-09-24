import { Prisma } from "@prisma/client";
import { prisma } from "../db";

export async function monthlyTotals(userId: number, month: string) {
  return prisma.$queryRaw<{ day: string; total: number }[]>(
    Prisma.sql`SELECT DATE(created_at) AS day, SUM(amount) AS total
               FROM transfers
               WHERE sender_id = ${userId} AND DATE_FORMAT(created_at, '%Y-%m') = ${month}
               GROUP BY day ORDER BY day`
  );
}
