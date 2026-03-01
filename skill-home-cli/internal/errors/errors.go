package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode 错误代码
type ErrorCode string

const (
	// 通用错误
	ErrUnknown        ErrorCode = "UNKNOWN"
	ErrInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrPermission     ErrorCode = "PERMISSION_DENIED"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrTimeout        ErrorCode = "TIMEOUT"
	ErrNetwork        ErrorCode = "NETWORK_ERROR"

	// 技能相关
	ErrInvalidSkill    ErrorCode = "INVALID_SKILL"
	ErrSkillNotFound   ErrorCode = "SKILL_NOT_FOUND"
	ErrInvalidManifest ErrorCode = "INVALID_MANIFEST"
	ErrScanFailed      ErrorCode = "SCAN_FAILED"

	// 注册中心相关
	ErrRegistryUnavailable ErrorCode = "REGISTRY_UNAVAILABLE"
	ErrVersionExists       ErrorCode = "VERSION_EXISTS"
	ErrValidationFailed    ErrorCode = "VALIDATION_FAILED"
)

// Error 应用错误
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
	Details map[string]interface{}
}

// Error 实现 error 接口
func (e *Error) Error() string {
	var parts []string

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("(原因: %v)", e.Cause))
	}

	return strings.Join(parts, " ")
}

// Unwrap 返回底层错误
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetail 添加详情
func (e *Error) WithDetail(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Is 检查错误是否匹配
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}

	return false
}

// New 创建新的错误
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf 创建格式化的错误
func Newf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap 包装已有错误
func Wrap(cause error, code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf 包装并格式化
func Wrapf(cause error, code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// Is 检查错误代码
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(*Error); ok {
		return e.Code == code
	}

	return false
}

// AsError 将普通 error 转为 *Error
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return e
	}

	return &Error{
		Code:    ErrUnknown,
		Message: err.Error(),
	}
}

// FormatErrorMessage 格式化错误消息
func FormatErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var messages []string
	for err != nil {
		if e, ok := err.(*Error); ok {
			if e.Message != "" {
				messages = append(messages, e.Message)
			}
			err = e.Cause
		} else {
			messages = append(messages, err.Error())
			break
		}
	}

	return strings.Join(messages, ": ")
}

// 标准库 errors 的别名
var (
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
)

// 预定义错误
var (
	ErrNotLoggedIn = New(ErrUnauthorized, "未登录，请先运行 'skill-home login'")
	ErrNoSkillFile = New(ErrNotFound, "未找到 SKILL.md 文件")
)
