package agent

import (
	"context"
	"fmt"
	"log"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"myapp/services"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

type TransferInfo struct {
	UserID     uint   `json:"user_id" jsonschema:"user ID of whom the work belongs to"`
	WorkID     uint   `json:"id" jsonschema:"work ID in database"`
	WorkName   string `json:"work_name" jsonschema:"name of the work"`
	Style      string `json:"style" jsonschema:"style to be transferred to"`
	Weight     string `json:"weight" jsonschema:"weight of the style"`
	Iterations string `json:"iterations" jsonschema:"number of iterations to be performed"`
}

type Result struct {
	Msg string `json:"msg"`
}

func TransferInfoTools(ctx context.Context) ([]tool.BaseTool, []*schema.ToolInfo) {
	transferTool, _ := utils.InferTool("transfer_work", "将一个.ply格式的3DGS模型进行风格转换,转换为wikiart数据集中的风格类型", TransferFunc)

	tools := []tool.BaseTool{
		transferTool,
	}

	var toolInfos []*schema.ToolInfo
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		if err != nil {
			log.Fatalf("get ToolInfo failed, err=%v", err)
		}
		toolInfos = append(toolInfos, info)
	}

	return tools, toolInfos
}

// TODO
func GetCurrentWorkFunc(_ context.Context, transInfo *TransferInfo) (*Result, error) {
	panic("not implemented")
}

func WorkDataUpdateFunc(_ context.Context, transInfo *TransferInfo) (*Result, error) {
	var work models.Work
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserID:     transInfo.UserID,
			WorkName:   transInfo.WorkName,
			Status:     "processing",
			Iterations: transInfo.Iterations,
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		return nil, err
	}

	return &Result{"work data init successfully"}, nil
}

func TransferFunc(_ context.Context, transInfo *TransferInfo) (*Result, error) {
	dataPath := fmt.Sprintf("work%d.pth", transInfo.WorkID)
	dataPath, err := database.RetrieveFromBucket(dataPath)
	styleImage := "default.jpg"
	if err != nil {
		return nil, err
	}

	processor, err := services.NewProcessor(transInfo.Iterations)
	if err != nil {
		return nil, err
	}

	processor.Stylize(dataPath, styleImage)

	return &Result{"transfer start successfully"}, nil
}
