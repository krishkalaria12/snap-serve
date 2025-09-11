# 📷 Snap Serve

A powerful image processing and storage API built with Go, Fiber, and Google Cloud Storage. Snap Serve allows users to upload images, apply various filters and transformations, and manage their image collections with secure authentication.

## ✨ Features

### 🔐 Authentication & User Management
- **User Registration & Login** - Secure JWT-based authentication
- **User CRUD Operations** - Create, read, update, and delete user accounts
- **Password Hashing** - Secure bcrypt password encryption
- **JWT Token Management** - Token-based authorization with cookie support

### 🖼️ Image Processing & Storage
- **Image Upload** - Upload images to Google Cloud Storage
- **Advanced Image Filters** - Apply multiple image processing filters:
  - **Resize** - Scale images to specific dimensions
  - **Crop** - Crop images to desired size
  - **Rotate** - Rotate images by any angle
  - **Brightness** - Increase/decrease image brightness
  - **Contrast** - Adjust image contrast
  - **Saturation** - Modify color saturation
  - **Gaussian Blur** - Apply blur effects
  - **Pixelate** - Create pixelated effects
  - **Grayscale** - Convert to black and white
  - **Invert** - Invert image colors

### 🏗️ Architecture
- **Clean Architecture** - Modular design with separated concerns
- **Database Integration** - PostgreSQL with GORM ORM
- **Cloud Storage** - Google Cloud Storage for image hosting
- **Middleware Support** - Authentication and logging middleware
- **Error Handling** - Comprehensive error responses

## 🚀 Quick Start

### Prerequisites
- Go 1.24.6 or higher
- PostgreSQL database
- Google Cloud Storage account with credentials
- `.env` file with required environment variables

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/krishkalaria12/snap-serve.git
cd snap-serve
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Set up environment variables:**
Create a `.env` file in the root directory:
```env
DATABASE_URL=postgresql://username:password@localhost:5432/snap_serve
JWT_SECRET=your-super-secret-jwt-key
GSC_PROJECT_ID=your-google-cloud-project-id
GSC_BUCKET_NAME=your-storage-bucket-name
```

4. **Set up Google Cloud credentials:**
Place your `credentials.json` file in the root directory or set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.

5. **Run the application:**
```bash
go run main.go
```

The server will start on `http://localhost:3000`

## 📚 API Documentation

### Authentication Endpoints

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
  "identity": "username_or_email",
  "password": "your_password"
}
```

### User Management Endpoints

#### Create User
```http
POST /api/user
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

#### Get User
```http
GET /api/user/{id}
```

#### Update User (Authenticated)
```http
PUT /api/user/{id}
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "username": "newusername",
  "name": "New Name"
}
```

#### Delete User (Authenticated)
```http
DELETE /api/user/{id}
Authorization: Bearer {jwt_token}
```

### Image Endpoints

#### Upload Image (Authenticated)
```http
POST /api/image/upload
Authorization: Bearer {jwt_token}
Content-Type: multipart/form-data

form-data:
- document: (image file)
```

#### Apply Image Filters (Authenticated)
```http
POST /api/image/filter?resize=800x600&brightness_increase=20&grayscale=true
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "image_url": "https://storage.googleapis.com/your-bucket/image.jpg"
}
```

### Available Image Filters

| Filter | Parameter | Description | Example |
|--------|-----------|-------------|---------|
| `resize` | `widthxheight` | Resize image to specified dimensions | `resize=800x600` |
| `crop_to_size` | `widthxheight` | Crop image to specified size | `crop_to_size=400x400` |
| `rotate` | `degrees` | Rotate image by specified angle | `rotate=90` |
| `brightness_increase` | `value` | Increase brightness (0-100) | `brightness_increase=20` |
| `brightness_decrease` | `value` | Decrease brightness (0-100) | `brightness_decrease=15` |
| `contrast_increase` | `value` | Increase contrast (0-100) | `contrast_increase=30` |
| `contrast_decrease` | `value` | Decrease contrast (0-100) | `contrast_decrease=10` |
| `saturation_increase` | `value` | Increase saturation (0-200) | `saturation_increase=50` |
| `saturation_decrease` | `value` | Decrease saturation (0-200) | `saturation_decrease=25` |
| `gaussian_blur` | `radius` | Apply Gaussian blur (0.1-50) | `gaussian_blur=2.5` |
| `pixelate` | `size` | Apply pixelation effect (1-50) | `pixelate=8` |
| `grayscale` | - | Convert to grayscale | `grayscale=true` |
| `invert` | - | Invert colors | `invert=true` |

### Utility Endpoints

#### Health Check
```http
GET /api/hello
```

## 🏛️ Project Structure

```
snap-serve/
├── auth/                    # Authentication service
│   └── service.go          # Auth service implementation
├── config/                  # Configuration management
│   └── config.go           # Environment config loader
├── database/                # Database connection
│   └── connect.go          # PostgreSQL connection setup
├── handlers/                # HTTP request handlers
│   ├── auth-handler.go     # Authentication endpoints
│   ├── hello-handler.go    # Health check endpoint
│   ├── image-handler.go    # Image upload/management
│   ├── image-filters.go    # Image processing filters
│   └── user-handler.go     # User CRUD operations
├── middleware/              # HTTP middleware
│   └── auth-middleware.go  # JWT authentication middleware
├── models/                  # Data models
│   ├── image-models.go     # Image entity model
│   └── user-models.go      # User entity model
├── router/                  # Route definitions
│   └── routes.go           # API route setup
├── main.go                 # Application entry point
├── go.mod                  # Go module dependencies
└── README.md               # Project documentation
```

## 🛠️ Tech Stack

- **Backend Framework:** [Fiber](https://gofiber.io/) - Express-inspired web framework
- **Database:** PostgreSQL with [GORM](https://gorm.io/) ORM
- **Authentication:** JWT tokens with [go-pkgz/auth](https://github.com/go-pkgz/auth)
- **Image Processing:** [Gift](https://github.com/disintegration/gift) - Go Image Filtering Toolkit
- **Cloud Storage:** Google Cloud Storage
- **Environment Management:** [godotenv](https://github.com/joho/godotenv)
- **Password Hashing:** bcrypt

## 🔧 Configuration

### Environment Variables

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Yes | `postgresql://user:pass@localhost:5432/db` |
| `JWT_SECRET` | Secret key for JWT signing | Yes | `your-super-secret-key` |
| `GSC_PROJECT_ID` | Google Cloud project ID | Yes | `my-project-123` |
| `GSC_BUCKET_NAME` | Google Cloud Storage bucket name | Yes | `my-images-bucket` |

### Google Cloud Setup

1. Create a Google Cloud project
2. Enable Cloud Storage API
3. Create a storage bucket
4. Create a service account with Storage Admin permissions
5. Download the service account key as `credentials.json`

## 🧪 Development

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -o snap-serve main.go
```

### Database Migrations
The application automatically runs database migrations on startup using GORM's AutoMigrate feature.

## 📝 API Response Format

All API responses follow a consistent format:

```json
{
  "status": "success|error",
  "message": "Human readable message",
  "data": {} // Response data or null
}
```

## 🔒 Security Features

- **JWT Authentication** - Secure token-based authentication
- **Password Hashing** - bcrypt encryption for user passwords
- **Request Validation** - Input validation and sanitization
- **CORS Support** - Cross-origin resource sharing configuration
- **File Type Validation** - Image format validation for uploads
- **Size Limits** - Maximum image dimensions and file size restrictions

## 🙏 Acknowledgments

- [Fiber](https://gofiber.io/) for the excellent web framework
- [GORM](https://gorm.io/) for the powerful ORM
- [Gift](https://github.com/disintegration/gift) for image processing capabilities
- [go-pkgz/auth](https://github.com/go-pkgz/auth) for authentication services
