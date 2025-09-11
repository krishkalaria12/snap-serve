package handler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/middleware"
	"google.golang.org/genai"
)

func injectSysPrompt(prompt string) string {
	return fmt.Sprintf(`You are an AI image generation assistant. Create detailed, visual descriptions for image generation models. Focus on:

- Clear visual elements (colors, composition, lighting, style)
- Specific artistic techniques or photographic styles when relevant
- Safe, appropriate content only
- Realistic and achievable image concepts

Transform user requests into precise, descriptive prompts that will produce high-quality images.

User request: %s`, prompt)
}

func GenerateImage(c *fiber.Ctx) error {
	ctx := context.Background()

	userId, err := middleware.CheckUserLoggedIn(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Authentication required",
			"data":    nil,
		})
	}

	type GenerateImageRequest struct {
		Prompt string `json:"prompt"`
	}

	var genImage GenerateImageRequest
	if err := c.BodyParser(&genImage); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"data":    nil,
		})
	}

	if genImage.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Prompt is required",
			"data":    nil,
		})
	}

	if len(genImage.Prompt) > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Prompt too long (max 1000 characters)",
			"data":    nil,
		})
	}

	enhancedPrompt := injectSysPrompt(genImage.Prompt)

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-image-preview",
		genai.Text(enhancedPrompt),
		&genai.GenerateContentConfig{},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to generate image",
			"data":    nil,
		})
	}

	if len(result.Candidates[0].Content.Parts) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "No image content in response",
			"data":    nil,
		})
	}

	var imageBytes []byte
	var foundImage bool

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != nil {
			imageBytes = part.InlineData.Data
			foundImage = true
			break
		}
	}

	if !foundImage {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "No image data found in response",
			"data":    nil,
		})
	}

	if len(imageBytes) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Empty image data received",
			"data":    nil,
		})
	}

	reader := bytes.NewReader(imageBytes)

	outputFilename := fmt.Sprintf("generated_%d.png", time.Now().UnixNano())

	url, filename, err := uploader.UploadProcessedFile(reader, outputFilename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to upload generated image",
			"data":    nil,
		})
	}

	if err := uploadImageToDB(url, filename, userId); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to save image record",
			"data":    nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Successfully generated image",
		"data": fiber.Map{
			"url":      url,
			"filename": filename,
		},
	})
}
