# Build stage
FROM node:24-slim AS build
WORKDIR /app

RUN npm install -g pnpm@10.33.2

COPY .npmrc pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY . .

RUN pnpm install --frozen-lockfile
RUN pnpm --filter @workspace/auth-dashboard build

# Production stage
FROM nginx:alpine
COPY --from=build /app/artifacts/auth-dashboard/dist/public /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]