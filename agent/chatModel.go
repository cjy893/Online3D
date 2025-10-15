package agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type Agent struct {
	runnable compose.Runnable[[]*schema.Message, []*schema.Message]
}

var ServerAgent Agent

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

	tools, toolInfos := TransferInfoTools(ctx)
	err = model.BindTools(toolInfos)
	if err != nil {
		return err
	}

	todoToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools: tools,
	})
	if err != nil {
		return err
	}

	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(model, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	agent, err := chain.Compile(ctx)
	if err != nil {
		return err
	}

	ServerAgent.runnable = agent

	return nil
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
