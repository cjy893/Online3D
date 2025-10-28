package agent

import (
	"context"
	"myapp/agent/tools"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type Agent struct {
	runnable compose.Runnable[[]*schema.Message, []*schema.Message]
}

var ServerAgent Agent

var handlers []callbacks.Handler

func InitAgent() error {

	ctx := context.Background()
	model, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		Model:     "deepseek-reasoner",
		APIKey:    "sk-f174f0645e1c4bbbb92595efa9fef8f5",
		MaxTokens: 2000,
	})
	if err != nil {
		return err
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{tools.GetWorkTool(), tools.GetStyleImageTool(), tools.WorkTransferTool()},
		},
	})

	panic("TODO")
}

func (a *Agent) Invoke(ctx context.Context, content string) ([]*schema.Message, error) {
	resp, err := a.runnable.Invoke(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: content,
		},
	})
	if err != nil {
		return resp, err
	}

	return resp, nil
}
