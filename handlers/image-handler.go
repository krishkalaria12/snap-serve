package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"sync"
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

type UploadResult struct {
	URL      string
	Filename string
	Error    error
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

func UploadMultipleImages(c *fiber.Ctx) error {
	userID, err := middleware.CheckUserLoggedIn(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized Request",
			"data":    nil,
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Error parsing multipart form",
			"data":    nil,
		})
	}

	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "No files provided",
			"data":    nil,
		})
	}

	uploadResults := routineUploadMultipleImages(files)
	
	successfulUploads := []UploadResult{}
	var uploadErrors []string
	
	for _, result := range uploadResults {
		if result.Error != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("Error uploading %s: %v", result.Filename, result.Error))
		} else {
			successfulUploads = append(successfulUploads, result)
		}
	}

	if len(successfulUploads) > 0 {
		dbErrors := routineSaveImageRecords(successfulUploads, userID)
		if len(dbErrors) > 0 {
			for _, dbErr := range dbErrors {
				uploadErrors = append(uploadErrors, fmt.Sprintf("Database error: %v", dbErr))
			}
		}
	}

	urls := make([]string, 0, len(successfulUploads))
	for _, result := range successfulUploads {
		urls = append(urls, result.URL)
	}

	responseData := fiber.Map{
		"uploaded_urls": urls,
		"success_count": len(successfulUploads),
		"total_count":   len(files),
	}

	if len(uploadErrors) > 0 {
		responseData["errors"] = uploadErrors
		return c.Status(fiber.StatusPartialContent).JSON(fiber.Map{
			"status":  "partial_success",
			"message": fmt.Sprintf("Uploaded %d out of %d files", len(successfulUploads), len(files)),
			"data":    responseData,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("Successfully uploaded %d files", len(successfulUploads)),
		"data":    responseData,
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

func routineUploadImages(readers []*bytes.Reader, baseFilename string) []UploadResult {
	uploadResults := make(chan UploadResult, len(readers))
	var wg sync.WaitGroup

	for i, reader := range readers {
		wg.Add(1)
		go func(r *bytes.Reader, index int) {
			defer wg.Done()
			filename := fmt.Sprintf("%s_%d.jpg", baseFilename, index)
			url, uploadedFilename, err := uploader.UploadProcessedFile(r, filename)
			uploadResults <- UploadResult{
				URL:      url,
				Filename: uploadedFilename,
				Error:    err,
			}
		}(reader, i)
	}

	go func() {
		wg.Wait()
		close(uploadResults)
	}()

	results := []UploadResult{}
	for result := range uploadResults {
		results = append(results, result)
	}

	return results
}

func routineUploadMultipleImages(files []*multipart.FileHeader) []UploadResult {
	uploadResults := make(chan UploadResult, len(files))
	var wg sync.WaitGroup

	for _, fileHeader := range files {
		wg.Add(1)
		go func(fh *multipart.FileHeader) {
			defer wg.Done()
			
			file, err := fh.Open()
			if err != nil {
				uploadResults <- UploadResult{
					URL:      "",
					Filename: fh.Filename,
					Error:    fmt.Errorf("failed to open file %s: %v", fh.Filename, err),
				}
				return
			}
			defer file.Close()

			url, uploadedFilename, err := uploader.UploadFile(file, fh.Filename)
			uploadResults <- UploadResult{
				URL:      url,
				Filename: uploadedFilename,
				Error:    err,
			}
		}(fileHeader)
	}

	go func() {
		wg.Wait()
		close(uploadResults)
	}()

	results := []UploadResult{}
	for result := range uploadResults {
		results = append(results, result)
	}

	return results
}

func routineSaveImageRecords(uploadResults []UploadResult, userId uint) []error {
	saveErrors := make(chan error, len(uploadResults))
	var wg sync.WaitGroup

	for _, result := range uploadResults {
		if result.Error != nil {
			continue
		}
		wg.Add(1)
		go func(url, filename string) {
			defer wg.Done()
			err := uploadImageToDB(url, filename, userId)
			saveErrors <- err
		}(result.URL, result.Filename)
	}

	go func() {
		wg.Wait()
		close(saveErrors)
	}()

	var errors []error
	for err := range saveErrors {
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// func MakeBucketPublic() error {
// 	if uploader == nil {
// 		return fmt.Errorf("uploader not initialized")
// 	}
// 	return uploader.MakeBucketPublic()
// }
