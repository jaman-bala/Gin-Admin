# Build stage
FROM node:24-slim AS build
WORKDIR /app

# Enable pnpm
RUN npm install -g pnpm

# Copy workspace config and package.json
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./

# Copy all source code
COPY . .

# Install dependencies and build
RUN pnpm install --no-frozen-lockfile
RUN PORT=3000 BASE_PATH=/ pnpm --filter @workspace/auth-dashboard build

# Production stage
FROM nginx:alpine
# Copy the built app
COPY --from=build /app/artifacts/auth-dashboard/dist/public /usr/share/nginx/html
# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
