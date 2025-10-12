package usage

type ModelPricing struct {
	InputTokenCost  float64
	OutputTokenCost float64
}

type ProviderPricing map[string]ModelPricing

var GlobalPricing = map[string]ProviderPricing{
	"openai": {
		"gpt-5": {
			InputTokenCost:  1.25,
			OutputTokenCost: 10.0,
		},
		"gpt-5-pro": {
			InputTokenCost:  15.0,
			OutputTokenCost: 75.0,
		},
		"gpt-5-mini": {
			InputTokenCost:  0.25,
			OutputTokenCost: 2.0,
		},
		"gpt-5-nano": {
			InputTokenCost:  0.05,
			OutputTokenCost: 0.4,
		},
		"gpt-4.1": {
			InputTokenCost:  30.0,
			OutputTokenCost: 60.0,
		},
		"gpt-4.1-mini": {
			InputTokenCost:  5.0,
			OutputTokenCost: 10.0,
		},
		"gpt-4.1-nano": {
			InputTokenCost:  0.5,
			OutputTokenCost: 1.0,
		},
		"gpt-4o": {
			InputTokenCost:  2.5,
			OutputTokenCost: 10.0,
		},
		"gpt-4o-mini": {
			InputTokenCost:  0.15,
			OutputTokenCost: 0.6,
		},
		"o3": {
			InputTokenCost:  60.0,
			OutputTokenCost: 240.0,
		},
		"o3-pro": {
			InputTokenCost:  120.0,
			OutputTokenCost: 480.0,
		},
		"o4-mini": {
			InputTokenCost:  10.0,
			OutputTokenCost: 40.0,
		},
	},
	"anthropic": {
		"claude-opus-4.1": {
			InputTokenCost:  15.0,
			OutputTokenCost: 75.0,
		},
		"claude-opus-4": {
			InputTokenCost:  15.0,
			OutputTokenCost: 75.0,
		},
		"claude-sonnet-4-5-20250929": {
			InputTokenCost:  3.0,
			OutputTokenCost: 15.0,
		},
		"claude-sonnet-3.7": {
			InputTokenCost:  3.0,
			OutputTokenCost: 15.0,
		},
		"claude-3-5-sonnet-20241022": {
			InputTokenCost:  3.0,
			OutputTokenCost: 15.0,
		},
		"claude-3-5-haiku-20241022": {
			InputTokenCost:  0.8,
			OutputTokenCost: 4.0,
		},
	},
	"gemini": {
		"gemini-2.5-pro": {
			InputTokenCost:  1.25,
			OutputTokenCost: 10.0,
		},
		"gemini-2.5-flash": {
			InputTokenCost:  0.3,
			OutputTokenCost: 1.2,
		},
		"gemini-2.5-flash-lite": {
			InputTokenCost:  0.1,
			OutputTokenCost: 0.4,
		},
		"gemini-2.0-flash": {
			InputTokenCost:  0.1,
			OutputTokenCost: 0.4,
		},
		"gemini-2.0-flash-live": {
			InputTokenCost:  0.15,
			OutputTokenCost: 0.6,
		},
	},
	"deepseek": {
		"deepseek-chat": {
			InputTokenCost:  0.27,
			OutputTokenCost: 1.1,
		},
		"deepseek-reasoner": {
			InputTokenCost:  0.55,
			OutputTokenCost: 2.19,
		},
		"deepseek-v3-0324": {
			InputTokenCost:  0.27,
			OutputTokenCost: 1.1,
		},
		"deepseek-r1": {
			InputTokenCost:  0.55,
			OutputTokenCost: 2.19,
		},
		"deepseek-r1-0528": {
			InputTokenCost:  0.75,
			OutputTokenCost: 2.99,
		},
		"deepseek-coder-v2": {
			InputTokenCost:  0.27,
			OutputTokenCost: 1.1,
		},
	},
	"groq": {
		"llama-3.3-70b-versatile": {
			InputTokenCost:  0.59,
			OutputTokenCost: 0.79,
		},
		"llama-3.1-8b-instant": {
			InputTokenCost:  0.05,
			OutputTokenCost: 0.08,
		},
		"deepseek-r1-distill-llama-70b": {
			InputTokenCost:  0.75,
			OutputTokenCost: 0.99,
		},
		"llama-3-groq-70b-tool-use": {
			InputTokenCost:  0.59,
			OutputTokenCost: 0.79,
		},
		"llama-3-groq-8b-tool-use": {
			InputTokenCost:  0.05,
			OutputTokenCost: 0.08,
		},
		"llama-guard-4-12b": {
			InputTokenCost:  0.2,
			OutputTokenCost: 0.2,
		},
	},
	"grok": {
		"grok-4": {
			InputTokenCost:  15.0,
			OutputTokenCost: 75.0,
		},
		"grok-4-heavy": {
			InputTokenCost:  25.0,
			OutputTokenCost: 125.0,
		},
		"grok-4-fast": {
			InputTokenCost:  5.0,
			OutputTokenCost: 25.0,
		},
		"grok-code-fast-1": {
			InputTokenCost:  3.0,
			OutputTokenCost: 15.0,
		},
		"grok-3": {
			InputTokenCost:  3.0,
			OutputTokenCost: 15.0,
		},
		"grok-3-mini": {
			InputTokenCost:  0.3,
			OutputTokenCost: 0.5,
		},
	},
	"huggingface": {
		"meta-llama/Llama-3.3-70B-Instruct": {
			InputTokenCost:  0.05,
			OutputTokenCost: 0.08,
		},
		"meta-llama/Llama-3.1-8B-Instruct": {
			InputTokenCost:  0.01,
			OutputTokenCost: 0.02,
		},
		"deepseek-ai/DeepSeek-R1-Distill-Qwen-14B": {
			InputTokenCost:  0.02,
			OutputTokenCost: 0.04,
		},
		"deepseek-ai/DeepSeek-R1-Distill-Llama-8B": {
			InputTokenCost:  0.01,
			OutputTokenCost: 0.02,
		},
		"Qwen/Qwen3-235B-A22B": {
			InputTokenCost:  0.1,
			OutputTokenCost: 0.2,
		},
		"Qwen/Qwen3-30B-A3B": {
			InputTokenCost:  0.02,
			OutputTokenCost: 0.04,
		},
	},
}

const (
	InputTokenOverhead  = 0.10
	OutputTokenOverhead = 0.20
)

func CalculateCost(provider, model string, inputTokens, outputTokens int) float64 {
	providerPricing, exists := GlobalPricing[provider]
	if !exists {
		return 0.0
	}

	modelPricing, exists := providerPricing[model]
	if !exists {
		return 0.0
	}

	baseCost := (float64(inputTokens)*modelPricing.InputTokenCost + float64(outputTokens)*modelPricing.OutputTokenCost) / 1_000_000.0

	overhead := (float64(inputTokens)*InputTokenOverhead + float64(outputTokens)*OutputTokenOverhead) / 1_000_000.0

	return baseCost + overhead
}
