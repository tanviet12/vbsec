export async function sendMail(to: string, subject: string, body: string) {
  await fetch(process.env.MAIL_API_URL as string, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${process.env.MAIL_API_KEY}` },
    body: JSON.stringify({ to, subject, body }),
  });
}
