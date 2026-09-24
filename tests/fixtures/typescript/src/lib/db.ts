import { Sequelize, DataTypes } from "sequelize";

export const sequelize = new Sequelize(process.env.DATABASE_URL as string);

export const User = sequelize.define("User", {
  email: DataTypes.STRING,
  name: DataTypes.STRING,
  bio: DataTypes.TEXT,
  role: { type: DataTypes.STRING, defaultValue: "customer" },
  balance: { type: DataTypes.INTEGER, defaultValue: 0 },
});
