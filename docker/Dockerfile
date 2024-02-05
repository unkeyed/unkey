# Use the official Node.js 18 image as a base
FROM node:18

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json (if available) to the working directory
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the rest of your application code to the working directory
COPY . .

# Build your TypeScript application
RUN npm run build

# Generate and run migrations
RUN npm run db:migrate
RUN npm run db:push

# Expose the port your app runs on
EXPOSE 3000

# Command to run your app
CMD ["node", "dist/index.js"]
