package imagegenerator

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type ImageGenerator struct {
	Model *ark.ImageGenerationModel
}

var Generator *ImageGenerator

func InitImageGenerationModel(ctx context.Context) error {
	imageGenerationModel, err := ark.NewImageGenerationModel(ctx, &ark.ImageGenerationConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_IMAGE_MODEL_ID"), // Use an appropriate image model ID
	})

	if err != nil {
		return fmt.Errorf("NewImageGenerationModel failed, err=%v", err)
	}

	Generator.Model = imageGenerationModel
	return nil
}

func (g *ImageGenerator) GenerateStyleImage(input string, ctx context.Context) (string, error) {
	inMsgs := []*schema.Message{
		{
			Role:    schema.User,
			Content: input,
		},
	}

	msg, err := g.Model.Generate(ctx, inMsgs)
	if err != nil {
		return "", fmt.Errorf("Generate failed, err=%v", err)
	}

	imgPath := filepath.Join("transfer_img_tmp", uuid.NewString()+".jpg")
	if len(msg.MultiContent) > 0 {
		imgContent := msg.MultiContent[0]
		imageURL := imgContent.ImageURL.URL
		log.Printf("Generated image URL: %s", imageURL)

		// 立即下载图片，因为URL有时效性
		err = downloadImage(imageURL, imgPath)
		if err != nil {
			return "", fmt.Errorf("Failed to download image: %v", err)
		}
	}

	return imgPath, nil
}

func downloadImage(imageURL string, filename string) error {
	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
