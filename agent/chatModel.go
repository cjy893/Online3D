package agent

import (
	"context"
	"fmt"
	imagegenerator "myapp/agent/imageGenerator"
	"myapp/agent/tools"

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
		Model:     "deepseek-chat",
		APIKey:    "sk-f174f0645e1c4bbbb92595efa9fef8f5",
		MaxTokens: 2000,
	})
	if err != nil {
		return fmt.Errorf("NewChatModel failed, err=%v", err)
	}

	getWorkTool := tools.GetWorkTool()
	workTransferTool := tools.WorkTransferTool()

	workInfo, _ := getWorkTool.Info(ctx)
	fmt.Printf("GetWorkTool Info: %+v\n", workInfo)

	transferInfo, _ := workTransferTool.Info(ctx)
	fmt.Printf("WorkTransferTool Info: %+v\n", transferInfo)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []tool.BaseTool{getWorkTool, workTransferTool},
			ExecuteSequentially: true,
		},
		MaxStep: 4,
	})

	if err := imagegenerator.InitImageGenerationModel(ctx); err != nil {
		return fmt.Errorf("InitImageGenerationModel failed, err=%v", err)
	}

	ServerAgent.runnable = agent
	return nil
}

func (a *Agent) Generate(ctx context.Context, content string, userID uint) (string, error) {
	persona := fmt.Sprintf(`# 角色
		你是一个帮助用户对3D模型进行风格转换的AI助手。

		# 用户信息
		当前用户ID: %d

		# 操作流程
		当用户请求风格转换时，你必须按照以下步骤执行：
		1. 首先使用 QueryWork 工具查询用户的所有作品，确认指定的作品ID是否存在
		2. 如果作品存在，再使用 WorkTransfer 工具进行风格转换

		# 重要规则
		- 不要猜测作品是否存在，必须通过工具查询确认
		- 只有在确认作品存在后，才能进行风格转换
		- 在回复中不要提前透露会调用哪个工具，直接执行工具调用
		- 从用户请求中提取工具所需的所有参数

		# 可用工具
		1. QueryWork: 查询用户的作品列表，需要参数 user_id
		2. WorkTransfer: 进行风格转换，需要参数 work_id, new_work_name, visibility, iterations, style_description`, userID)

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: content,
		},
	}

	response, err := a.runnable.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to run: %v", err)
	}

	// 检查是否有工具调用，并执行
	// 注意：这里可能需要根据框架的实际行为调整
	// 因为Run方法可能已经自动执行了工具调用并返回了结果
	// 所以这里我们直接返回response的Content
	// 但为了调试，我们可以打印整个response
	fmt.Printf("Run response: %+v\n", response)

	// 假设response的Content是模型返回的文本，包括工具调用的结果
	return response.Content, nil
}

// func (a *Agent) Stream(ctx context.Context, content string, userID uint, writer io.Writer) error {
// 	// 将 userID 添加到角色描述中，以便模型可以根据用户信息提供个性化服务
// 	persona := fmt.Sprintf(`# 角色
// 		你是一个帮助用户对3D模型进行风格转换的AI助手。

// 		# 用户信息
// 		当前用户ID: %d

// 		# 重要指令
// 		你必须按照以下流程执行：

// 		1. 首先，必须调用QueryWork工具查询用户的作品。
// 		2. 等待QueryWork的结果返回。
// 		3. 根据QueryWork的结果：
// 		- 如果返回了作品列表（至少1个作品）：立即调用TransferWork工具进行风格转换
// 		- 如果没有作品：告诉用户"您还没有可用的作品"

// 		# TransferWork参数规则
// 		当调用TransferWork时，请遵循：
// 		1. work_id: 使用QueryWork返回的第一个作品的ID，或者用户明确指定的作品ID
// 		2. work_name: 从用户输入中提取新作品名称
// 		3. input: 从用户输入中提取风格描述
// 		4. 其他参数：从用户输入中提取

// 		# 注意
// 		- 不要询问任何问题
// 		- QueryWork之后必须立即调用TransferWork（如果有作品）
// 		- 这是强制流程，不要跳过任何步骤`, userID)

// 	sr, err := a.runnable.Stream(ctx, []*schema.Message{
// 		{
// 			Role:    schema.System,
// 			Content: persona,
// 		},
// 		{
// 			Role:    schema.User,
// 			Content: content,
// 		},
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to stream: %v", err)
// 	}

// 	defer sr.Close()

// 	flusher, ok := writer.(http.Flusher)
// 	if !ok {
// 		return fmt.Errorf("streaming unsupported")
// 	}

// 	// 用于累积工具调用的缓冲区
// 	var currentToolCallID string
// 	var toolCallBuffer bytes.Buffer

// 	for {
// 		msg, err := sr.Recv()
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				break
// 			}
// 			return fmt.Errorf("failed to recv: %v", err)
// 		}

// 		// 添加更详细的调试日志
// 		fmt.Printf("收到消息: Role=%s, Content='%s', ToolCalls数量=%d\n",
// 			msg.Role, msg.Content, len(msg.ToolCalls))

// 		if len(msg.ToolCalls) > 0 {
// 			for i, tc := range msg.ToolCalls {
// 				fmt.Printf("工具调用 %d: ID=%s, Name='%s', Arguments='%s'\n",
// 					i, tc.ID, tc.Function.Name, tc.Function.Arguments)

// 				// 检查是否是新的工具调用
// 				if tc.ID != "" && tc.ID != currentToolCallID {
// 					// 开始新的工具调用
// 					if toolCallBuffer.Len() > 0 {
// 						fmt.Printf("完成的工具调用参数: %s\n", toolCallBuffer.String())
// 						toolCallBuffer.Reset()
// 					}
// 					currentToolCallID = tc.ID
// 				}

// 				// 累积参数
// 				if tc.Function.Arguments != "" {
// 					toolCallBuffer.WriteString(tc.Function.Arguments)
// 				}

// 				// 检查工具调用是否完成（这里需要根据实际情况判断）
// 				// 可以根据参数是否包含完整的JSON来判断
// 				bufferStr := toolCallBuffer.String()
// 				if strings.Contains(bufferStr, "{") && strings.Contains(bufferStr, "}") {
// 					// 尝试解析JSON，如果成功则认为完成
// 					var js map[string]interface{}
// 					if json.Unmarshal([]byte(bufferStr), &js) == nil {
// 						fmt.Printf("工具调用完成: %s -> %s\n", tc.Function.Name, bufferStr)
// 						toolCallBuffer.Reset()
// 					}
// 				}
// 			}
// 		}

// 		// 输出内容到前端
// 		if msg.Content != "" {
// 			_, writeErr := writer.Write([]byte(msg.Content))
// 			if writeErr != nil {
// 				return writeErr
// 			}
// 			flusher.Flush()
// 		}
// 	}

// 	return nil
// }
