# Use the official Node.js 16 image as a base
FROM node:18

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json (if available) to the working directory
COPY package*.json ./

# Install dependencies
RUN npm install

# Build
RUN npm run build

# Generate migrations
RUN npm run db:migrate

# Run migrations
RUN npm run db:push

# Copy the rest of your application code to the working directory
COPY . .

# Expose the port your app runs on
EXPOSE 3000

# Command to run your app
CMD ["node", "dist/index.js"]
