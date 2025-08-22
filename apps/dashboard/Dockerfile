FROM node:22.18.0

WORKDIR /unkey

# Install pnpm
RUN npm install -g pnpm

# Copy everything
COPY . .

# Install dependencies
RUN pnpm install -r

# Move to dashboard directory
WORKDIR /unkey/apps/dashboard
RUN pnpm build
EXPOSE 3000

# Set hostname to 0.0.0.0 to allow external connections
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

# Run in development mode for now
CMD ["pnpm","start", "--hostname", "0.0.0.0"]
