import jwt from "jsonwebtoken";
import bcrypt from "bcryptjs";
import type { User } from "@workspace/db";

const ACCESS_SECRET = process.env.SESSION_SECRET ?? "access-secret-dev";
const REFRESH_SECRET = (process.env.SESSION_SECRET ?? "refresh-secret-dev") + "-refresh";
const ACCESS_EXPIRES = "15m";
const REFRESH_EXPIRES_DAYS = 30;

export function hashPassword(password: string): Promise<string> {
  return bcrypt.hash(password, 10);
}

export function comparePassword(password: string, hash: string): Promise<boolean> {
  return bcrypt.compare(password, hash);
}

export interface JwtPayload {
  sub: string;
  role: string;
}

export function signAccessToken(user: Pick<User, "id" | "role">): string {
  return jwt.sign({ sub: user.id, role: user.role }, ACCESS_SECRET, {
    expiresIn: ACCESS_EXPIRES,
  });
}

export function signRefreshToken(userId: string): string {
  return jwt.sign({ sub: userId }, REFRESH_SECRET, {
    expiresIn: `${REFRESH_EXPIRES_DAYS}d`,
  });
}

export function verifyAccessToken(token: string): JwtPayload {
  return jwt.verify(token, ACCESS_SECRET) as JwtPayload;
}

export function verifyRefreshToken(token: string): { sub: string } {
  return jwt.verify(token, REFRESH_SECRET) as { sub: string };
}

export function refreshTokenExpiresAt(): Date {
  const d = new Date();
  d.setDate(d.getDate() + REFRESH_EXPIRES_DAYS);
  return d;
}

export function userToPublic(user: User) {
  return {
    id: user.id,
    first_name: user.firstName,
    last_name: user.lastName,
    middle_name: user.middleName ?? undefined,
    phone: user.phone,
    role: user.role,
    photo: user.photo ?? undefined,
    is_active: user.isActive,
    created_at: user.createdAt.toISOString(),
    updated_at: user.updatedAt.toISOString(),
  };
}
