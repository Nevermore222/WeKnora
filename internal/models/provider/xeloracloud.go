package provider

import "github.com/Tencent/Xelora/internal/types"

const (
	ProviderXeloraCloud ProviderName = "xeloracloud"

	// XeloraCloudBaseURL XeloraCloud 服务硬编码 Base URL（统一入口，路径由各实现拼接）
	XeloraCloudBaseURL = "https://xelora.weixin.qq.com"
)

type XeloraCloudProvider struct{}

func init() {
	Register(&XeloraCloudProvider{})
}

func (p *XeloraCloudProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderXeloraCloud,
		DisplayName: "XeloraCloud",
		Description: "Xelora云服务，模型：chat, embedding, rerank, vlm",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: XeloraCloudBaseURL,
			types.ModelTypeEmbedding:   XeloraCloudBaseURL,
			types.ModelTypeRerank:      XeloraCloudBaseURL,
			types.ModelTypeVLLM:        XeloraCloudBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
		},
		RequiresAuth: true,
	}
}

func (p *XeloraCloudProvider) ValidateConfig(config *Config) error {
	// AppID/AppSecret 通过专用初始化接口写入，此处仅做结构校验。
	// 其中 AppSecret 字段当前实际承载上游 API Key。
	return nil
}
