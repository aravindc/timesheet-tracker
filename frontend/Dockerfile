## Stage 1: Build the Angular application
FROM node:20-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy package.json and package-lock.json to leverage Docker cache
COPY package*.json ./

# Install project dependencies
RUN npm install

# Copy the rest of the application source code
COPY . .

# Generate the production build of the Angular app
# The output will be in the /app/dist/your-app-name/browser directory
RUN npm run build

## Stage 2: Serve the application from Nginx
FROM nginx:stable-alpine

# Copy the built assets from the "builder" stage
# IMPORTANT: Replace "your-app-name" with the actual name of your project folder in the 'dist' directory.
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy the custom Nginx configuration file
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
# EXPOSE 80