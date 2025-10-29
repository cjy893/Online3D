package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	imagegenerator "myapp/agent/imageGenerator"
	"myapp/agent/tools"
	"net/http"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type Agent struct {
	runnable *react.Agent
}

var ServerAgent Agent

var ImageGenerationModel *ark.ImageGenerationModel

func InitAgent() error {

	ctx := context.Background()
	model, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		Model:     "deepseek-reasoner",
		APIKey:    "sk-f174f0645e1c4bbbb92595efa9fef8f5",
		MaxTokens: 2000,
	})
	if err != nil {
		return fmt.Errorf("NewChatModel failed, err=%v", err)
	}

	getWorkTool := tools.GetWorkTool()
	workTransferTool := tools.WorkTransferTool()

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{getWorkTool, workTransferTool},
		},
	})

	if err := imagegenerator.InitImageGenerationModel(ctx); err != nil {
		return fmt.Errorf("InitImageGenerationModel failed, err=%v", err)
	}

	ServerAgent.runnable = agent
	return nil
}

func (a *Agent) Stream(ctx context.Context, content string, userID uint, writer io.Writer) error {
	// 将 userID 添加到角色描述中，以便模型可以根据用户信息提供个性化服务
	persona := fmt.Sprintf(`# Character:
	你是一个帮助用户对指定3D模型进行风格转换的助手，你需要根据用户的输入，通过调用生成工具，将3D模型进行风格转换。
	当前用户ID: %d
	`, userID)

	sr, err := a.runnable.Stream(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: content,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to stream: %v", err)
	}

	defer sr.Close()

	flusher, ok := writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to recv: %v", err)
		}

		_, writeErr := writer.Write([]byte(msg.Content))
		if writeErr != nil {
			return writeErr
		}

		flusher.Flush()
	}

	return nil
}
