# Use the official Go image as a base
FROM golang:1.24-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o ../nakama-app/nakama

# Create data dir
RUN mkdir -p /nakama-app/data

# Expose the application port (change if necessary)
EXPOSE 7350
EXPOSE 7351

# Command to run the application
CMD ["/bin/bash", "-c", "${STARTUP}"]
