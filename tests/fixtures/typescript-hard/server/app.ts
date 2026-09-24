import express from "express";
import authRouter from "./routes/auth";
import importRouter from "./routes/import";
import filesRouter from "./routes/files";
import transferRouter from "./routes/transfer";
import adminRouter from "./routes/admin";
import reportsRouter from "./routes/reports";
import { requireUser } from "./middleware/auth";

const app = express();
app.use(express.json());

app.use("/auth", authRouter);
app.use("/files", filesRouter);
app.use("/import", requireUser, importRouter);
app.use("/transfer", requireUser, transferRouter);
app.use("/reports", requireUser, reportsRouter);
app.use("/admin", requireUser, adminRouter);

app.listen(3000);
