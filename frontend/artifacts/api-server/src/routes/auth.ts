import { Router, type IRouter } from "express";
import multer from "multer";
import { db, usersTable, refreshTokensTable } from "@workspace/db";
import { eq, and } from "drizzle-orm";
import {
  AuthLoginBody,
  AuthRefreshBody,
  AuthLogoutBody,
  AuthRegisterBody,
} from "@workspace/api-zod";
import {
  hashPassword,
  comparePassword,
  signAccessToken,
  signRefreshToken,
  verifyRefreshToken,
  refreshTokenExpiresAt,
  userToPublic,
} from "../lib/auth";
import { authenticate, type AuthRequest } from "../middlewares/authenticate";

const router: IRouter = Router();
const upload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 5 * 1024 * 1024 } });

router.post("/auth/login", async (req, res): Promise<void> => {
  const parsed = AuthLoginBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Invalid request body" });
    return;
  }

  const { phone, password } = parsed.data;

  const [user] = await db.select().from(usersTable).where(eq(usersTable.phone, phone));
  if (!user) {
    res.status(401).json({ message: "Invalid credentials" });
    return;
  }

  if (!user.isActive) {
    res.status(403).json({ message: "Account is inactive" });
    return;
  }

  const valid = await comparePassword(password, user.passwordHash);
  if (!valid) {
    res.status(401).json({ message: "Invalid credentials" });
    return;
  }

  const accessToken = signAccessToken(user);
  const refreshToken = signRefreshToken(user.id);

  await db.insert(refreshTokensTable).values({
    userId: user.id,
    token: refreshToken,
    expiresAt: refreshTokenExpiresAt(),
  });

  res.json({
    access_token: accessToken,
    refresh_token: refreshToken,
    message: "Success authorization",
  });
});

router.post("/auth/register", upload.single("photo"), async (req, res): Promise<void> => {
  const parsed = AuthRegisterBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Validation error", error: parsed.error.message });
    return;
  }

  const { first_name, last_name, middle_name, phone, password } = parsed.data;

  const [existing] = await db.select().from(usersTable).where(eq(usersTable.phone, phone));
  if (existing) {
    res.status(409).json({ message: "Phone number already registered" });
    return;
  }

  const passwordHash = await hashPassword(password);
  let photoData: string | undefined;

  if (req.file) {
    const b64 = req.file.buffer.toString("base64");
    photoData = `data:${req.file.mimetype};base64,${b64}`;
  }

  const [user] = await db
    .insert(usersTable)
    .values({
      firstName: first_name,
      lastName: last_name,
      middleName: middle_name ?? null,
      phone,
      passwordHash,
      photo: photoData ?? null,
    })
    .returning();

  res.status(201).json(userToPublic(user));
});

router.post("/auth/refresh", async (req, res): Promise<void> => {
  const parsed = AuthRefreshBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Invalid request body" });
    return;
  }

  const { refresh_token } = parsed.data;

  let payload: { sub: string };
  try {
    payload = verifyRefreshToken(refresh_token);
  } catch {
    res.status(401).json({ message: "Invalid or expired refresh token" });
    return;
  }

  const [storedToken] = await db
    .select()
    .from(refreshTokensTable)
    .where(
      and(
        eq(refreshTokensTable.token, refresh_token),
        eq(refreshTokensTable.isRevoked, false),
      ),
    );

  if (!storedToken || storedToken.expiresAt < new Date()) {
    res.status(401).json({ message: "Refresh token is revoked or expired" });
    return;
  }

  const [user] = await db.select().from(usersTable).where(eq(usersTable.id, payload.sub));
  if (!user || !user.isActive) {
    res.status(401).json({ message: "User not found or inactive" });
    return;
  }

  const accessToken = signAccessToken(user);
  const expiresAt = new Date(Date.now() + 15 * 60 * 1000);

  res.json({
    access_token: accessToken,
    expires_at: expiresAt.toISOString(),
    user: userToPublic(user),
  });
});

router.post("/auth/logout", authenticate, async (req: AuthRequest, res): Promise<void> => {
  const parsed = AuthLogoutBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: "Invalid request body" });
    return;
  }

  const { token } = parsed.data;

  await db
    .update(refreshTokensTable)
    .set({ isRevoked: true })
    .where(eq(refreshTokensTable.token, token));

  res.json({ message: "Logged out successfully" });
});

export default router;
