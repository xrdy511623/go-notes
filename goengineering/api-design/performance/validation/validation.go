// Package validation 对比手动校验与反射校验的性能开销。
//
// 结论: 反射校验的便利性在大多数 Web 应用中远超性能开销。
// 仅在极高频内部调用（>100 万次/秒）场景下才需要考虑手动校验。
package validation

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
)

// UserRequest 是测试用的请求结构体。
type UserRequest struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"min=0,max=150"`
}

// ValidateManual 使用手写代码校验，无反射开销。
func ValidateManual(req UserRequest) map[string]string {
	errs := make(map[string]string)

	if req.Name == "" {
		errs["name"] = "name is required"
	} else if len(req.Name) < 2 {
		errs["name"] = "name must be at least 2 characters"
	} else if len(req.Name) > 50 {
		errs["name"] = "name must be at most 50 characters"
	}

	if req.Email == "" {
		errs["email"] = "email is required"
	} else if _, err := mail.ParseAddress(req.Email); err != nil {
		errs["email"] = "email must be a valid email address"
	}

	if req.Age < 0 {
		errs["age"] = "age must be at least 0"
	} else if req.Age > 150 {
		errs["age"] = "age must be at most 150"
	}

	return errs
}

// ValidateReflect 使用反射 + struct tag 校验，通用但有反射开销。
func ValidateReflect(v any) map[string]string {
	errs := make(map[string]string)
	val := reflect.ValueOf(v)
	typ := val.Type()

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = val.Type()
	}

	for i := range val.NumField() {
		field := typ.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		fieldVal := val.Field(i)
		name := jsonName(field)

		for _, rule := range strings.Split(tag, ",") {
			if msg := applyRule(rule, fieldVal, name); msg != "" {
				errs[name] = msg
				break
			}
		}
	}
	return errs
}

func jsonName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

func applyRule(rule string, val reflect.Value, name string) string {
	switch {
	case rule == "required":
		if val.IsZero() {
			return fmt.Sprintf("%s is required", name)
		}
	case rule == "email":
		if val.Kind() == reflect.String && val.String() != "" {
			if _, err := mail.ParseAddress(val.String()); err != nil {
				return fmt.Sprintf("%s must be a valid email address", name)
			}
		}
	case strings.HasPrefix(rule, "min="):
		n, _ := strconv.ParseInt(strings.TrimPrefix(rule, "min="), 10, 64)
		switch val.Kind() {
		case reflect.String:
			if int64(len(val.String())) < n {
				return fmt.Sprintf("%s must be at least %d characters", name, n)
			}
		case reflect.Int, reflect.Int64:
			if val.Int() < n {
				return fmt.Sprintf("%s must be at least %d", name, n)
			}
		}
	case strings.HasPrefix(rule, "max="):
		n, _ := strconv.ParseInt(strings.TrimPrefix(rule, "max="), 10, 64)
		switch val.Kind() {
		case reflect.String:
			if int64(len(val.String())) > n {
				return fmt.Sprintf("%s must be at most %d characters", name, n)
			}
		case reflect.Int, reflect.Int64:
			if val.Int() > n {
				return fmt.Sprintf("%s must be at most %d", name, n)
			}
		}
	}
	return ""
}
