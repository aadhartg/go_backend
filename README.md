# DNA Collection Management System

Welcome to the DNA Collection Management System repository! This project is a web application built with Go, leveraging PostgreSQL for data storage and Gorilla Mux for routing.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [Folder Structure](#folder-structure)
- [API Documentation](#api-documentation)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Environment-Based Configuration**: Supports development, production, testing, and localhost environments.
- **Database Management**: Automatically checks and creates the target PostgreSQL database if it doesn't exist.
- **API Routing**: Organized using Gorilla Mux with middleware support.
- **Logging**: Middleware for logging all API requests.
- **Standardized Responses**: Consistent API response structure for easy client-side handling.
- **ORM Integration**: Utilizes GORM for database interactions and migrations.

## Prerequisites

Before you begin, ensure you have met the following requirements:

- **Go**: Version 1.18 or higher installed. [Download Go](https://golang.org/dl/)
- **PostgreSQL**: Installed and running. [Download PostgreSQL](https://www.postgresql.org/download/)
- **Git**: Installed for version control. [Download Git](https://git-scm.com/downloads)

## Installation

1. **Clone the Repository**

   ```bash
   git clone https://github.com/aadhartg/Theranostics-DNA-Backend.git
   cd Theranostics-DNA-Backend
   ```

2. **Install Dependencies**

   Ensure you have Go modules enabled. Then, run:

   ```bash
      go mod download
      go mod tidy
   ```

## Configuration

1. **Environment Variables**

   Copy the `.env.example` to `.env`:

   ```bash
   cp .env.example .env
   ```

2. **Edit the .env File**

   Open the `.env` file and set your environment variables accordingly:

   ```env
   ENVIRONMENT=development  # development, testing, production, localhost
   ```

# Development Database Config

DEV_DB_HOST=localhost
DEV_DB_PORT=5432
DEV_DB_USER=dev_user
DEV_DB_PASSWORD=dev_password
DEV_DB_NAME=dev_db
DEV_SERVER_PORT=8080
DEV_JWT_SECRET_KEY="**\*\*\***"
DEV_ALLOWED_ORIGINS="http://example.com,http://example.com,http://example.com"
DEV_SMTP_FROM_EMAIL = "enter email address"
DEV_SMTP_EMAIL = Enter email address
DEV_SMTP_PASSWORD = "Enter email password of nodemailer"
DEV_SMTP_SERVER = smtp.gmail.com
DEV_API_URL="**\*\*\***"
DEV_APP_URL="**\*\***"

# Encryption Keys (base64-encoded, 32 bytes when decoded)

DEV_ENCRYPTION_KEY1=your-base64-encoded-32-byte-key1==
DEV_ENCRYPTION_KEY2=your-base64-encoded-32-byte-key2==

# Production Database Config

PROD_DB_HOST=prod_host
PROD_DB_PORT=5432
PROD_DB_USER=prod_user
PROD_DB_PASSWORD=prod_password
PROD_DB_NAME=prod_db
PROD_SERVER_PORT=8080
PROD_JWT_SECRET_KEY="**\*\*\***"
PROD_ALLOWED_ORIGINS="http://example.com,http://example.com,http://example.com"
PROD_SMTP_FROM_EMAIL = "enter email address"
PROD_SMTP_EMAIL = Enter email address
PROD_SMTP_PASSWORD = "Enter email password of nodemailer"
PROD_SMTP_SERVER = smtp.gmail.com
PROD_API_URL="**\*\*\***"
PROD_APP_URL="**\*\***"

# Encryption Keys (base64-encoded, 32 bytes when decoded)

PROD_ENCRYPTION_KEY1=your-base64-encoded-32-byte-key1==
PROD_ENCRYPTION_KEY2=your-base64-encoded-32-byte-key2==

# Testing Database Config

TEST_DB_HOST=TEST_host
TEST_DB_PORT=5432
TEST_DB_USER=TEST_user
TEST_DB_PASSWORD=TEST_password
TEST_DB_NAME=TEST_db
TEST_SERVER_PORT=8080
TEST_JWT_SECRET_KEY="**\*\*\***"
TEST_ALLOWED_ORIGINS="http://example.com,http://example.com,http://example.com"
TEST_SMTP_FROM_EMAIL = "enter email address"
TEST_SMTP_EMAIL = Enter email address
TEST_SMTP_PASSWORD = "Enter email password of nodemailer"
TEST_SMTP_SERVER = smtp.gmail.com
TEST_API_URL="**\*\*\***"
TEST_APP_URL="**\*\***"

# Encryption Keys (base64-encoded, 32 bytes when decoded)

TEST_ENCRYPTION_KEY1=your-base64-encoded-32-byte-key1==
TEST_ENCRYPTION_KEY2=your-base64-encoded-32-byte-key2==

# Localhost Database Config

LOCAL_DB_HOST=LOCAL_host
LOCAL_DB_PORT=5432
LOCAL_DB_USER=LOCAL_user
LOCAL_DB_PASSWORD=LOCAL_password
LOCAL_DB_NAME=LOCAL_db
LOCAL_SERVER_PORT=8080
LOCAL_JWT_SECRET_KEY="**\*\*\***"
LOCAL_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:8080"
LOCAL_SMTP_FROM_EMAIL = "enter email address"
LOCAL_SMTP_EMAIL = Enter email address
LOCAL_SMTP_PASSWORD = "Enter email password of nodemailer"
LOCAL_SMTP_SERVER = smtp.gmail.com
LOCAL_API_URL="http://localhost:8080"
LOCAL_APP_URL="http://localhost:3000"

# Encryption Keys (base64-encoded, 32 bytes when decoded)

LOCAL_ENCRYPTION_KEY1=MTIzNDU2Nzg5Cg==
LOCAL_ENCRYPTION_KEY2=MTIzNDU2Nzg5Cg==

````

Note: Ensure the `.env` file is not committed to version control as it contains sensitive information.

**Note**: Generate secure keys using a reliable method, such as:

```bash
openssl rand -base64 32
````

## Running the Application

1. **Run the Server**

   ```bash
   go run main.go
   ```

   to real environment changes

   ```bash
     air
   ```

   You should see output indicating that the database is connected, migrated, and the server has started:

   ```
   Database connection established for development environment
   Database migrated successfully
   Server started at port 8080
   ```

2. **Access the Welcome Endpoint**

   Open your browser or use curl to access `http://localhost:8080/`:

   ```bash
   curl http://localhost:8080/
   ```

   Response:

   ```json
   {
     "status": 200,
     "success": true,
     "message": "Welcome to the project."
   }
   ```

## Folder Structure

```
project-root/
├── config/
│   └── config.go
├── controllers/
│   └── welcome_controller.go
├── middlewares/
│   ├── logging_middleware.go
│   └── middleware.go
├── models/
│   ├── role.go
│   └── user.go
├── routes/
│   └── routes.go
├── services/
│   └── service.go
├── utils/
│   ├── constants.go
│   ├── env.go
│   └── response.go
├── .env
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── main.go
```

Description of Each Folder:

- `config/`: Contains configuration-related files, such as database initialization.
- `controllers/`: Houses the logic for handling HTTP requests.
- `middlewares/`: Contains middleware functions for tasks like logging, authentication, etc.
- `models/`: Defines the data models and interacts with the database.
- `routes/`: Defines all API routes.
- `services/`: Contains business logic separate from controllers.
- `utils/`: Includes utility functions, constants, environment variable management, and response handling.
- `.env` & `.env.example`: Environment variable files.
- `main.go`: Entry point of the application.

## API Documentation

### Welcome Endpoint

- **URL**: `/`
- **Method**: GET
- **Description**: Returns a welcome message.
- **Response**:
  ```json
  {
    "status": 200,
    "success": true,
    "message": "Welcome to the project."
  }
  ```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the Repository
2. Create a Feature Branch
   ```bash
   git checkout -b feature/YourFeature
   ```
3. Commit Your Changes
   ```bash
   git commit -m "Add Your Feature"
   ```
4. Push to the Branch
   ```bash
   git push origin feature/YourFeature
   ```
5. Open a Pull Request

## License

This project is licensed under the MIT License.
