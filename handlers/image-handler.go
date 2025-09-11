package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/config"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/middleware"
	"github.com/krishkalaria12/snap-serve/models"
	"gorm.io/gorm"
)

var projectId string = config.Config("GSC_PROJECT_ID")
var bucketName string = config.Config("GSC_BUCKET_NAME")

type ClientUploader struct {
	cl         *storage.Client
	projectID  string
	bucketName string
	uploadPath string
}

var uploader *ClientUploader

func init() {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./credentials.json")
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	uploader = &ClientUploader{
		cl:         client,
		bucketName: bucketName,
		projectID:  projectId,
		uploadPath: "images/",
	}
}

func uploadImageToDB(url, filename string, userID uint) error {
	db := database.GetDB()

	image := models.Image{
		UserID:      userID,
		Filename:    filename,
		OriginalURL: url,
		Status:      "completed",
	}

	if err := db.Create(&image).Error; err != nil {
		return err
	}

	return nil
}

func GetImageFromDB(url string) (models.Image, error) {
	db := database.GetDB()
	var image models.Image

	result := db.Where("original_url = ?", url).First(&image)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return image, errors.New("image not found")
		}

		return image, result.Error
	}

	return image, nil
}

func UploadImage(c *fiber.Ctx) error {
	userID, err := middleware.CheckUserLoggedIn(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized Request",
			"data":    nil,
		})
	}

	file, err := c.FormFile("document")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "No file provided",
			"data":    nil,
		})
	}

	blobFile, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error opening the file",
			"data":    nil,
		})
	}
	defer blobFile.Close() // Important: close the file

	url, originalFilename, err := uploader.UploadFile(blobFile, file.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error uploading the file",
			"data":    nil,
		})
	}

	if err := uploadImageToDB(url, originalFilename, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error saving to database",
			"data":    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Successfully uploaded the file",
		"data":    url,
	})
}

// Update your UploadFile method signature
func (c *ClientUploader) UploadProcessedFile(file io.Reader, object string) (string, string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Better unique filename generation
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	uniqueFilename := timestamp + "_" + object

	// Full object path
	objectPath := c.uploadPath + uniqueFilename

	// Upload an object with storage.Writer.
	wc := c.cl.Bucket(c.bucketName).Object(objectPath).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return "", "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", "", fmt.Errorf("Writer.Close: %v", err)
	}

	// Generate the public URL
	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, objectPath)
	return url, object, nil
}

// UploadFile uploads an object and returns the public URL
func (c *ClientUploader) UploadFile(file multipart.File, originalFilename string) (string, string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Better unique filename generation
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	uniqueFilename := timestamp + "_" + originalFilename

	// Full object path
	objectPath := c.uploadPath + uniqueFilename

	// Upload an object with storage.Writer.
	wc := c.cl.Bucket(c.bucketName).Object(objectPath).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return "", "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", "", fmt.Errorf("Writer.Close: %v", err)
	}

	// Generate the public URL
	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, objectPath)
	return url, originalFilename, nil
}

// Alternative: Generate signed URL (if bucket is private)
func (c *ClientUploader) UploadFileWithSignedURL(file multipart.File, object string) (string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	object = timestamp + "_" + object
	objectPath := c.uploadPath + object

	// Upload file
	wc := c.cl.Bucket(c.bucketName).Object(objectPath).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %v", err)
	}

	// Generate signed URL (valid for 24 hours)
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(24 * time.Hour),
	}

	signedURL, err := c.cl.Bucket(c.bucketName).SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}

	return signedURL, nil
}

// Make bucket/object public (call this once for public access)
func (c *ClientUploader) MakeBucketPublic() error {
	ctx := context.Background()
	bucket := c.cl.Bucket(c.bucketName)

	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return err
	}

	// Add allUsers with objectViewer role
	policy.Add("allUsers", "roles/storage.objectViewer")

	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
		return err
	}

	return nil
}

// func MakeBucketPublic() error {
// 	if uploader == nil {
// 		return fmt.Errorf("uploader not initialized")
// 	}
// 	return uploader.MakeBucketPublic()
// }
