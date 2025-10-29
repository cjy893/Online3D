package tools

import (
	"context"
	"encoding/json"
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
				Type:     "uint",
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
				Type:     "uint",
				Desc:     "ID of the work to be transferred",
				Required: true,
			},
			"work_name": {
				Type:     "string",
				Desc:     "Name of the work after transfer",
				Required: true,
			},
			"is_public": {
				Type:     "bool",
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

func (t *ToolTransferWork) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	p := &WorkTransferParams{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		return "", err
	}

	styleImagePath, err := imagegenerator.Generator.GenerateStyleImage(p.input, ctx)

	origin, err := workService.QueryWork(p.WorkID)
	if err != nil {
		return "", err
	}

	work, err := workService.NewWork(p.TransferInfo, origin)
	if err != nil {
		return "", err
	}

	taskData := struct {
		TransferInfo   workService.TransferInfo `json:"transfer_info"`
		StyleImagePath string                   `json:"style_image_path"`
		WorkID         uint                     `json:"work_id"`
	}{
		TransferInfo:   p.TransferInfo,
		StyleImagePath: styleImagePath,
		WorkID:         work.ID,
	}

	task := &websocket.Task{
		Data:      taskData,
		ID:        uuid.New().String(),
		StartTime: time.Now(),
		Type:      "transfer",
		UserID:    work.UserID,
		WorkID:    work.ID,
	}

	config.Conf.TaskQueue.Tasks <- task

	return "Work transfer start successfully", nil
}

type WorkTransferParams struct {
	workService.TransferInfo
	input string
}
