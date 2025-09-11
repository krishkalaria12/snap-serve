package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"strconv"
	"strings"

	"github.com/disintegration/gift"
	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/middleware"
)

const (
	MaxImageWidth  = 4000
	MaxImageHeight = 4000
	JPEGQuality    = 90
	MaxBlurRadius  = 50
	MaxBrightness  = 100
	MaxContrast    = 100
	MaxSaturation  = 200
)

var supportedFilters = map[string]bool{
	"resize":              true,
	"crop_to_size":        true,
	"rotate":              true,
	"brightness_increase": true,
	"brightness_decrease": true,
	"contrast_increase":   true,
	"contrast_decrease":   true,
	"saturation_increase": true,
	"saturation_decrease": true,
	"gaussian_blur":       true,
	"pixelate":            true,
	"grayscale":           true,
	"invert":              true,
}

type ImageRequest struct {
	ImageUrl string `json:"image_url"`
}

type FilterError struct {
	FilterName string
	Message    string
}

func (e FilterError) Error() string {
	return fmt.Sprintf("filter '%s': %s", e.FilterName, e.Message)
}

func validateURL(imageURL string) error {
	_, err := GetImageFromDB(imageURL)

	if err != nil {
		return err
	}

	return nil
}

func loadImage(imageURL string) (image.Image, error) {
	if err := validateURL(imageURL); err != nil {
		return nil, err
	}

	res, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d", res.StatusCode)
	}

	// Check content type
	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("URL does not point to an image")
	}

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// Check image dimensions
	bounds := img.Bounds()
	if bounds.Dx() > MaxImageWidth || bounds.Dy() > MaxImageHeight {
		return nil, fmt.Errorf("image too large (max %dx%d)", MaxImageWidth, MaxImageHeight)
	}

	return img, nil
}

func parseIntParam(param, paramName string) (int, error) {
	if param == "" {
		return 0, fmt.Errorf("%s parameter is required", paramName)
	}

	value, err := strconv.Atoi(param)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be an integer", paramName)
	}

	if value < 0 {
		return 0, fmt.Errorf("%s must be positive", paramName)
	}

	return value, nil
}

func parseFloatParam(param, paramName string, min, max float32) (float32, error) {
	if param == "" {
		return 0, fmt.Errorf("%s parameter is required", paramName)
	}

	value, err := strconv.ParseFloat(param, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be a number", paramName)
	}

	floatVal := float32(value)
	if floatVal < min || floatVal > max {
		return 0, fmt.Errorf("%s must be between %.1f and %.1f", paramName, min, max)
	}

	return floatVal, nil
}

func parseDimensions(param, filterName string) (int, int, error) {
	if param == "" {
		return 0, 0, FilterError{filterName, "dimensions parameter is required"}
	}

	parts := strings.Split(param, "x")
	if len(parts) != 2 {
		return 0, 0, FilterError{filterName, "dimensions must be in format 'widthxheight'"}
	}

	width, err := parseIntParam(parts[0], "width")
	if err != nil {
		return 0, 0, FilterError{filterName, err.Error()}
	}

	height, err := parseIntParam(parts[1], "height")
	if err != nil {
		return 0, 0, FilterError{filterName, err.Error()}
	}

	if width > MaxImageWidth || height > MaxImageHeight {
		return 0, 0, FilterError{filterName, fmt.Sprintf("dimensions too large (max %dx%d)", MaxImageWidth, MaxImageHeight)}
	}

	return width, height, nil
}

func createFilter(filterName, param string) (gift.Filter, error) {
	switch filterName {
	case "resize":
		width, height, err := parseDimensions(param, filterName)
		if err != nil {
			return nil, err
		}
		return gift.Resize(width, height, gift.LanczosResampling), nil

	case "crop_to_size":
		width, height, err := parseDimensions(param, filterName)
		if err != nil {
			return nil, err
		}
		return gift.CropToSize(width, height, gift.LeftAnchor), nil

	case "rotate":
		degree, err := parseFloatParam(param, "rotation angle", -360, 360)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Rotate(degree, color.Transparent, gift.CubicInterpolation), nil

	case "brightness_increase":
		value, err := parseFloatParam(param, "brightness", 0, MaxBrightness)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Brightness(value), nil

	case "brightness_decrease":
		value, err := parseFloatParam(param, "brightness", 0, MaxBrightness)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Brightness(-value), nil

	case "contrast_increase":
		value, err := parseFloatParam(param, "contrast", 0, MaxContrast)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Contrast(value), nil

	case "contrast_decrease":
		value, err := parseFloatParam(param, "contrast", 0, MaxContrast)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Contrast(-value), nil

	case "saturation_increase":
		value, err := parseFloatParam(param, "saturation", 0, MaxSaturation)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Saturation(value), nil

	case "saturation_decrease":
		value, err := parseFloatParam(param, "saturation", 0, MaxSaturation)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.Saturation(-value), nil

	case "gaussian_blur":
		value, err := parseFloatParam(param, "blur radius", 0.1, MaxBlurRadius)
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		return gift.GaussianBlur(value), nil

	case "pixelate":
		value, err := parseIntParam(param, "pixelate size")
		if err != nil {
			return nil, FilterError{filterName, err.Error()}
		}
		if value > 50 {
			return nil, FilterError{filterName, "pixelate size too large (max 50)"}
		}
		return gift.Pixelate(value), nil

	case "grayscale":
		return gift.Grayscale(), nil

	case "invert":
		return gift.Invert(), nil

	default:
		return nil, FilterError{filterName, "unsupported filter"}
	}
}

func parseFilters(queryParams map[string]string) ([]gift.Filter, error) {
	var filters []gift.Filter

	for filterName, param := range queryParams {
		if !supportedFilters[filterName] {
			continue // Skip unknown parameters
		}

		filter, err := createFilter(filterName, param)
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	if len(filters) == 0 {
		return nil, fmt.Errorf("no valid filters specified")
	}

	return filters, nil
}

func processImage(src image.Image, filters []gift.Filter) (image.Image, error) {
	g := gift.New(filters...)
	dst := image.NewRGBA(g.Bounds(src.Bounds()))
	g.Draw(dst, src)
	return dst, nil
}

func encodeImage(img image.Image) (*bytes.Reader, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}
	return bytes.NewReader(buf.Bytes()), nil
}

func ApplyFilterToImage(c *fiber.Ctx) error {
	// Authenticate user
	userId, err := middleware.CheckUserLoggedIn(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Authentication required",
			"data":    nil,
		})
	}

	// Parse request body
	var imageData ImageRequest
	if err := c.BodyParser(&imageData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"data":    nil,
		})
	}

	if imageData.ImageUrl == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "image_url is required",
			"data":    nil,
		})
	}

	// Load image from URL
	img, err := loadImage(imageData.ImageUrl)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to load image: %v", err),
			"data":    nil,
		})
	}

	// Parse filters from query parameters
	filters, err := parseFilters(c.Queries())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
			"data":    nil,
		})
	}

	// Process image
	processedImg, err := processImage(img, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to process image",
			"data":    nil,
		})
	}

	// Encode processed image
	reader, err := encodeImage(processedImg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to encode processed image",
			"data":    nil,
		})
	}

	// Upload processed image
	processedFilename := "processed_image.jpg"
	url, filename, err := uploader.UploadProcessedFile(reader, processedFilename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to upload processed image",
			"data":    nil,
		})
	}

	// Save to database
	if err := uploadImageToDB(url, filename, userId); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to save image record",
			"data":    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Successfully processed image",
		"data": fiber.Map{
			"url":      url,
			"filename": filename,
		},
	})
}
