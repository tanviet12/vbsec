import { prisma } from "../db";

export async function findUserByEmail(email: string) {
  const rows = await prisma.$queryRawUnsafe<{ id: number; email: string }[]>(
    `SELECT id, email FROM users WHERE email = '${email}' LIMIT 1`
  );
  return rows[0] ?? null;
}

export async function createResetToken(userId: number, token: string) {
  await prisma.passwordReset.create({ data: { userId, token } });
}
