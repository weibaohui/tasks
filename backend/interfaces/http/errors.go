/**
 * HTTP 通用类型定义
 */
package http

// HTTPError HTTP错误响应
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
