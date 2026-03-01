package restful

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
)

// Validate 基于 struct tag `validate` 校验结构体字段。
// 支持规则: required, email, min=N, max=N
// 返回字段名→错误消息的映射，空 map 表示全部通过。
//
// 示例用法:
//
//	type CreateUserRequest struct {
//	    Name  string `json:"name"  validate:"required,min=2,max=50"`
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age"   validate:"min=0,max=150"`
//	}
func Validate(v any) map[string]string {
	errs := make(map[string]string)
	val := reflect.ValueOf(v)
	typ := val.Type()

	// 支持指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			errs["_"] = "input must not be nil"
			return errs
		}
		val = val.Elem()
		typ = val.Type()
	}

	if val.Kind() != reflect.Struct {
		errs["_"] = "input must be a struct"
		return errs
	}

	for i := range val.NumField() {
		field := typ.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := val.Field(i)
		fieldName := jsonFieldName(field)
		rules := strings.Split(tag, ",")

		for _, rule := range rules {
			if errMsg := applyRule(rule, fieldVal, fieldName); errMsg != "" {
				errs[fieldName] = errMsg
				break // 每个字段只报第一个错误
			}
		}
	}

	return errs
}

func jsonFieldName(f reflect.StructField) string {
	jsonTag := f.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return f.Name
	}
	name, _, _ := strings.Cut(jsonTag, ",")
	return name
}

func applyRule(rule string, val reflect.Value, name string) string {
	switch {
	case rule == "required":
		return checkRequired(val, name)
	case rule == "email":
		return checkEmail(val, name)
	case strings.HasPrefix(rule, "min="):
		return checkMin(rule, val, name)
	case strings.HasPrefix(rule, "max="):
		return checkMax(rule, val, name)
	default:
		return ""
	}
}

func checkRequired(val reflect.Value, name string) string {
	if val.IsZero() {
		return fmt.Sprintf("%s is required", name)
	}
	return ""
}

func checkEmail(val reflect.Value, name string) string {
	if val.Kind() != reflect.String {
		return ""
	}
	s := val.String()
	if s == "" {
		return "" // required 规则负责检查空值
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return fmt.Sprintf("%s must be a valid email address", name)
	}
	return ""
}

func checkMin(rule string, val reflect.Value, name string) string {
	minStr := strings.TrimPrefix(rule, "min=")
	minVal, err := strconv.ParseInt(minStr, 10, 64)
	if err != nil {
		return ""
	}

	switch val.Kind() {
	case reflect.String:
		if int64(len(val.String())) < minVal {
			return fmt.Sprintf("%s must be at least %d characters", name, minVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() < minVal {
			return fmt.Sprintf("%s must be at least %d", name, minVal)
		}
	}
	return ""
}

func checkMax(rule string, val reflect.Value, name string) string {
	maxStr := strings.TrimPrefix(rule, "max=")
	maxVal, err := strconv.ParseInt(maxStr, 10, 64)
	if err != nil {
		return ""
	}

	switch val.Kind() {
	case reflect.String:
		if int64(len(val.String())) > maxVal {
			return fmt.Sprintf("%s must be at most %d characters", name, maxVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() > maxVal {
			return fmt.Sprintf("%s must be at most %d", name, maxVal)
		}
	}
	return ""
}
