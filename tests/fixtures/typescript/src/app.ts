import express from "express";
import cors from "cors";
import searchRouter from "./routes/search";
import productsRouter from "./routes/products";
import convertRouter from "./routes/convert";
import accountRouter from "./routes/account";

const app = express();

app.use(express.json());
app.use(cors({ origin: true, credentials: true }));

app.use("/search", searchRouter);
app.use("/products", productsRouter);
app.use("/convert", convertRouter);
app.use("/account", accountRouter);

app.listen(3000);
