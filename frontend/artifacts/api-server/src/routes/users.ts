import { Router, type IRouter } from "express";
import multer from "multer";
import { db, usersTable } from "@workspace/db";
import { eq, ilike, or, and, count, sql } from "drizzle-orm";
import {
  CreateUserBody,
  UpdateMeBody,
  UpdateUserBody,
  GetUserParams,
  UpdateUserParams,
  DeleteUserParams,
  ListUsersQueryParams,
} from "@workspace/api-zod";
import { hashPassword, userToPublic } from "../lib/auth";
import { authenticate, requireAdmin, type AuthRequest } from "../middlewares/authenticate";

const router: IRouter = Router();
const upload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 5 * 1024 * 1024 } });

router.get("/users", authenticate, requireAdmin, async (req, res): Promise<void> => {
  const queryParsed = ListUsersQueryParams.safeParse(req.query);
  const page = queryParsed.success ? (queryParsed.data.page ?? 1) : 1;
  const limit = queryParsed.success ? (queryParsed.data.limit ?? 20) : 20;
  const search = queryParsed.success ? queryParsed.data.search : undefined;
  const isActiveFilter = queryParsed.success ? queryParsed.data.is_active : undefined;

  const offset = (page - 1) * limit;

  const conditions = [];
  if (search) {
    conditions.push(
      or(
        ilike(usersTable.firstName, `%${search}%`),
        ilike(usersTable.lastName, `%${search}%`),
        ilike(usersTable.phone, `%${search}%`),
      ),
    );
  }
  if (isActiveFilter !== undefined) {
    conditions.push(eq(usersTable.isActive, isActiveFilter));
  }

  const whereClause = conditions.length > 0 ? and(...conditions) : undefined;

  const [users, totalResult] = await Promise.all([
    db
      .select()
      .from(usersTable)
      .where(whereClause)
      .limit(limit)
      .offset(offset)
      .orderBy(usersTable.createdAt),
    db.select({ count: count() }).from(usersTable).where(whereClause),
  ]);

  const total = Number(totalResult[0]?.count ?? 0);

  res.json({
    users: users.map(userToPublic),
    total,
    page,
    limit,
  });
});

router.post("/users", authenticate, requireAdmin, upload.single("photo"), async (req, res): Promise<void> => {
  const parsed = CreateUserBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Validation error", error: parsed.error.message });
    return;
  }

  const { first_name, last_name, middle_name, phone, password, role } = parsed.data;

  const [existing] = await db.select().from(usersTable).where(eq(usersTable.phone, phone));
  if (existing) {
    res.status(409).json({ message: "Phone number already registered" });
    return;
  }

  const passwordHash = await hashPassword(password);
  let photoData: string | undefined;
  if (req.file) {
    photoData = `data:${req.file.mimetype};base64,${req.file.buffer.toString("base64")}`;
  }

  const [user] = await db
    .insert(usersTable)
    .values({
      firstName: first_name,
      lastName: last_name,
      middleName: middle_name ?? null,
      phone,
      passwordHash,
      role: role ?? "user",
      photo: photoData ?? null,
    })
    .returning();

  res.status(201).json(userToPublic(user));
});

router.get("/users/me", authenticate, async (req: AuthRequest, res): Promise<void> => {
  const [user] = await db.select().from(usersTable).where(eq(usersTable.id, req.userId!));
  if (!user) {
    res.status(404).json({ message: "User not found" });
    return;
  }
  res.json(userToPublic(user));
});

router.put("/users/me", authenticate, upload.single("photo"), async (req: AuthRequest, res): Promise<void> => {
  const parsed = UpdateMeBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Validation error", error: parsed.error.message });
    return;
  }

  const updates: Record<string, unknown> = {};
  if (parsed.data.first_name !== undefined) updates.firstName = parsed.data.first_name;
  if (parsed.data.last_name !== undefined) updates.lastName = parsed.data.last_name;
  if (parsed.data.middle_name !== undefined) updates.middleName = parsed.data.middle_name;
  if (parsed.data.phone !== undefined) updates.phone = parsed.data.phone;
  if (parsed.data.password !== undefined) updates.passwordHash = await hashPassword(parsed.data.password);

  if (req.file) {
    updates.photo = `data:${req.file.mimetype};base64,${req.file.buffer.toString("base64")}`;
  }

  const [user] = await db
    .update(usersTable)
    .set(updates)
    .where(eq(usersTable.id, req.userId!))
    .returning();

  if (!user) {
    res.status(404).json({ message: "User not found" });
    return;
  }

  res.json(userToPublic(user));
});

router.get("/users/stats", authenticate, requireAdmin, async (_req, res): Promise<void> => {
  const now = new Date();
  const firstOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);

  const [totalRow, activeRow, adminsRow, newThisMonthRow] = await Promise.all([
    db.select({ count: count() }).from(usersTable),
    db.select({ count: count() }).from(usersTable).where(eq(usersTable.isActive, true)),
    db.select({ count: count() }).from(usersTable).where(eq(usersTable.role, "admin")),
    db.select({ count: count() }).from(usersTable).where(
      sql`${usersTable.createdAt} >= ${firstOfMonth}`
    ),
  ]);

  const total = Number(totalRow[0]?.count ?? 0);
  const active = Number(activeRow[0]?.count ?? 0);
  const admins = Number(adminsRow[0]?.count ?? 0);
  const newThisMonth = Number(newThisMonthRow[0]?.count ?? 0);

  res.json({
    total,
    active,
    inactive: total - active,
    admins,
    new_this_month: newThisMonth,
  });
});

router.get("/users/:id", authenticate, requireAdmin, async (req, res): Promise<void> => {
  const params = GetUserParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ message: "Invalid user ID" });
    return;
  }

  const [user] = await db.select().from(usersTable).where(eq(usersTable.id, params.data.id));
  if (!user) {
    res.status(404).json({ message: "User not found" });
    return;
  }

  res.json(userToPublic(user));
});

router.put("/users/:id", authenticate, requireAdmin, upload.single("photo"), async (req, res): Promise<void> => {
  const params = UpdateUserParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ message: "Invalid user ID" });
    return;
  }

  const parsed = UpdateUserBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Validation error", error: parsed.error.message });
    return;
  }

  const updates: Record<string, unknown> = {};
  if (parsed.data.first_name !== undefined) updates.firstName = parsed.data.first_name;
  if (parsed.data.last_name !== undefined) updates.lastName = parsed.data.last_name;
  if (parsed.data.middle_name !== undefined) updates.middleName = parsed.data.middle_name;
  if (parsed.data.phone !== undefined) updates.phone = parsed.data.phone;
  if (parsed.data.is_active !== undefined) updates.isActive = parsed.data.is_active;
  if (parsed.data.password !== undefined) updates.passwordHash = await hashPassword(parsed.data.password);

  if (req.file) {
    updates.photo = `data:${req.file.mimetype};base64,${req.file.buffer.toString("base64")}`;
  }

  const [user] = await db
    .update(usersTable)
    .set(updates)
    .where(eq(usersTable.id, params.data.id))
    .returning();

  if (!user) {
    res.status(404).json({ message: "User not found" });
    return;
  }

  res.json(userToPublic(user));
});

router.delete("/users/:id", authenticate, requireAdmin, async (req, res): Promise<void> => {
  const params = DeleteUserParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ message: "Invalid user ID" });
    return;
  }

  const [user] = await db
    .delete(usersTable)
    .where(eq(usersTable.id, params.data.id))
    .returning();

  if (!user) {
    res.status(404).json({ message: "User not found" });
    return;
  }

  res.json({ message: "User deleted successfully" });
});

export default router;
