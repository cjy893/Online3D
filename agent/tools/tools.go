package tools

import (
	"context"
	"encoding/json"
	"myapp/services/workService"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func GetWorkTool() tool.InvokableTool {
	return &ToolQueryWork{}
}

type ToolQueryWork struct{}

func (t *ToolQueryWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "QueryWork",
		Desc: "Query work now existing belong to the user",
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
	panic("TODO")
}

type ToolTransferWork struct{}

func (t *ToolTransferWork) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "TransferWork",
		Desc: "Transfer work to another user",
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
				Desc:     "number of iterations for the work to transfer",
				Required: true,
			},
		}),
	}, nil
}

func (t *ToolTransferWork) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	p := &workService.TransferInfo{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		return "", err
	}

	origin, err := workService.QueryWork(p.WorkID)
	if err != nil {
		return "", err
	}

	work, err := workService.NewWork(*p, origin)
	if err != nil {
		return "", err
	}

	processor, err := workService.NewProcessor(p.Iterations)
	if err != nil {
		return "", err
	}

	panic("TODO")
}
