FROM golang:1.19

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Copy .env file
COPY .env ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /imagehoster

# Expose port 4000
EXPOSE 4000

# Run
CMD ["/imagehoster"]