package client

// APIResponse API 响应通用包装，根据 Status 将结果分离为成功数据或错误信息。
type APIResponse[T any] struct {
	Success bool      `json:"success"`         // 请求是否成功
	Data    *T        `json:"data,omitempty"`  // 成功时的业务数据
	Error   *APIError `json:"error,omitempty"` // 失败时的错误信息
}

// APIError API 错误信息。
type APIError struct {
	Code    int    `json:"code"`              // 状态码（Status）
	Message string `json:"message,omitempty"` // 错误消息
	ErrCode int    `json:"errcode,omitempty"` // 内部错误码（Errcode）
}

// WrapResponse 将原始 BaseResponse 包装为 APIResponse[T]。
// Status == 0 → 成功，填充 Data；Status != 0 → 失败，填充 Error。
func WrapResponse[T any](status int, message string, errcode int, data *T) *APIResponse[T] {
	if status != 0 {
		return &APIResponse[T]{
			Success: false,
			Error: &APIError{
				Code:    status,
				Message: message,
				ErrCode: errcode,
			},
		}
	}
	return &APIResponse[T]{
		Success: true,
		Data:    data,
	}
}
