// package tools

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	imagegenerator "myapp/agent/imageGenerator"
// 	"myapp/config"
// 	"myapp/services/websocket"
// 	"myapp/services/workService"
// 	"time"

// 	"github.com/cloudwego/eino/components/tool"
// 	"github.com/cloudwego/eino/schema"
// 	"github.com/google/uuid"
// )

// func GetWorkTool() tool.StreamableTool {
// 	return &ToolQueryWork{}
// }

// type ToolQueryWork struct{}

// func (t *ToolQueryWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
// 	return &schema.ToolInfo{
// 		Name: "QueryWork",
// 		Desc: "Query work now existing belong to the user according to the user ID",
// 		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
// 			"user_id": {
// 				Type:     "integer",
// 				Desc:     "ID of the user to query his or her works for",
// 				Required: true,
// 			},
// 		}),
// 	}, nil
// }

// func (t *ToolQueryWork) StreamableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
// 	p := &QueryWorkParams{}
// 	err := json.Unmarshal([]byte(argumentsInJSON), p)
// 	if err != nil {
// 		return nil, err
// 	}

// 	workInfos, err := workService.QueryWorkInfo(p.UserID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	infos, err := json.Marshal(workInfos)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return schema.StreamReaderFromArray([]string{string(infos)}), nil
// }

// type QueryWorkParams struct {
// 	UserID uint `json:"user_id"`
// }

// func WorkTransferTool() tool.StreamableTool {
// 	return &ToolTransferWork{}
// }

// type ToolTransferWork struct{}

// func (t *ToolTransferWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
// 	return &schema.ToolInfo{
// 		Name: "TransferWork",
// 		Desc: "Transfer work to another style",
// 		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
// 			"work_id": {
// 				Type:     "integer",
// 				Desc:     "ID of the work to be transferred",
// 				Required: true,
// 			},
// 			"work_name": {
// 				Type:     "string",
// 				Desc:     "Name of the work after transfer",
// 				Required: true,
// 			},
// 			"is_public": {
// 				Type:     "boolean",
// 				Desc:     "Whether the work is public for others to search after transfer",
// 				Required: true,
// 			},
// 			"iterations": {
// 				Type:     "string",
// 				Desc:     "Number of iterations for the work to transfer",
// 				Required: true,
// 			},
// 			"input": {
// 				Type:     "string",
// 				Desc:     "Input of user to generate style image in order to transfer work",
// 				Required: true,
// 			},
// 		}),
// 	}, nil
// }

// func (t *ToolTransferWork) StreamableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
// 	p := &WorkTransferParams{}
// 	err := json.Unmarshal([]byte(argumentsInJSON), p)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 创建 Pipe
// 	reader, writer := schema.Pipe[string](5) // 缓冲区大小为5

// 	// 异步执行任务
// 	go func() {
// 		defer writer.Close()

// 		// 发送开始消息
// 		writer.Send("Starting work transfer process...", nil)

// 		styleImagePath, err := imagegenerator.Generator.GenerateStyleImage(p.input, ctx)
// 		if err != nil {
// 			writer.Send("", fmt.Errorf("Failed to generate style image: %v", err))
// 			return
// 		}
// 		writer.Send("Style image generated successfully", nil)

// 		origin, err := workService.QueryWork(p.WorkID)
// 		if err != nil {
// 			writer.Send("", fmt.Errorf("Failed to query work: %v", err))
// 			return
// 		}
// 		writer.Send("Work found", nil)

// 		work, err := workService.NewWork(p.TransferInfo, origin)
// 		if err != nil {
// 			writer.Send("", fmt.Errorf("Failed to create new work: %v", err))
// 			return
// 		}
// 		writer.Send("New work created", nil)

// 		taskData := struct {
// 			TransferInfo   workService.TransferInfo `json:"transfer_info"`
// 			StyleImagePath string                   `json:"style_image_path"`
// 			WorkID         uint                     `json:"work_id"`
// 		}{
// 			TransferInfo:   p.TransferInfo,
// 			StyleImagePath: styleImagePath,
// 			WorkID:         work.ID,
// 		}

// 		task := &websocket.Task{
// 			Data:      taskData,
// 			ID:        uuid.New().String(),
// 			StartTime: time.Now(),
// 			Type:      "transfer",
// 			UserID:    work.UserID,
// 			WorkID:    work.ID,
// 		}

// 		config.Conf.TaskQueue.Tasks <- task

// 		writer.Send(fmt.Sprintf("Task submitted successfully. Task ID: %s", task.ID), nil)
// 		writer.Send("Work transfer process completed. You can check the progress in your task list.", nil)
// 	}()

// 	return reader, nil
// }

// type WorkTransferParams struct {
// 	workService.TransferInfo
// 	input string
// }

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	imagegenerator "myapp/agent/imageGenerator"
	"myapp/config"
	"myapp/services/websocket"
	"myapp/services/workService"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

func GetWorkTool() tool.InvokableTool {
	return &ToolQueryWork{}
}

type ToolQueryWork struct{}

func (t *ToolQueryWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "QueryWork",
		Desc: "Query work now existing belong to the user according to the user ID",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"user_id": {
				Type:     "integer",
				Desc:     "ID of the user to query his or her works for",
				Required: true,
			},
		}),
	}, nil
}

func (t *ToolQueryWork) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	p := &QueryWorkParams{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		return "", err
	}

	workInfos, err := workService.QueryWorkInfo(p.UserID)
	if err != nil {
		return "", err
	}

	infos, err := json.Marshal(workInfos)
	if err != nil {
		return "", err
	}

	return string(infos), nil
}

type QueryWorkParams struct {
	UserID uint `json:"user_id"`
}

func WorkTransferTool() tool.InvokableTool {
	return &ToolTransferWork{}
}

type ToolTransferWork struct{}

func (t *ToolTransferWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "TransferWork",
		Desc: "Transfer work to another style",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"work_id": {
				Type:     "integer",
				Desc:     "ID of the work to be transferred",
				Required: true,
			},
			"work_name": {
				Type:     "string",
				Desc:     "Name of the work after transfer",
				Required: true,
			},
			"is_public": {
				Type:     "boolean",
				Desc:     "Whether the work is public for others to search after transfer",
				Required: true,
			},
			"iterations": {
				Type:     "string",
				Desc:     "Number of iterations for the work to transfer",
				Required: true,
			},
			"input": {
				Type:     "string",
				Desc:     "Input of user to generate style image in order to transfer work",
				Required: true,
			},
		}),
	}, nil
}

type WorkTransferParams struct {
	WorkID     uint   `json:"work_id"`
	WorkName   string `json:"work_name"`
	IsPublic   bool   `json:"is_public"`
	Iterations string `json:"iterations"`
	Input      string `json:"input"`
}

func (t *ToolTransferWork) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	p := &WorkTransferParams{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		// 添加调试信息
		fmt.Printf("参数反序列化失败: %v, 原始参数: %s\n", err, argumentsInJSON)
		return "", err
	}

	// 添加参数验证
	if p.WorkID == 0 {
		return "", fmt.Errorf("work_id 不能为 0，请提供有效的作品ID")
	}

	fmt.Printf("接收到的参数: %+v\n", p)

	// 生成风格图像
	styleImagePath, err := imagegenerator.Generator.GenerateStyleImage(p.Input, ctx)
	if err != nil {
		return "", fmt.Errorf("生成风格图像失败: %v", err)
	}

	// 查询原始作品
	origin, err := workService.QueryWork(p.WorkID)
	if err != nil {
		return "", fmt.Errorf("查询作品失败 (ID: %d): %v", p.WorkID, err)
	}

	// 创建 TransferInfo
	transferInfo := workService.TransferInfo{
		WorkID:     p.WorkID,
		WorkName:   p.WorkName,
		IsPublic:   p.IsPublic,
		Iterations: p.Iterations,
	}

	// 创建新作品
	work, err := workService.NewWork(transferInfo, origin)
	if err != nil {
		return "", fmt.Errorf("创建新作品失败: %v", err)
	}

	// 创建任务数据
	taskData := struct {
		TransferInfo   workService.TransferInfo `json:"transfer_info"`
		StyleImagePath string                   `json:"style_image_path"`
		WorkID         uint                     `json:"work_id"`
	}{
		TransferInfo:   transferInfo,
		StyleImagePath: styleImagePath,
		WorkID:         work.ID,
	}

	// 创建并发送任务
	task := &websocket.Task{
		Data:      taskData,
		ID:        uuid.New().String(),
		StartTime: time.Now(),
		Type:      "transfer",
		UserID:    work.UserID,
		WorkID:    work.ID,
	}

	config.Conf.TaskQueue.Tasks <- task

	return fmt.Sprintf("作品转换任务已启动。任务ID: %s，新作品ID: %d", task.ID, work.ID), nil
}
