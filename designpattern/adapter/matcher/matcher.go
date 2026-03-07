package matcher

import (
	"regexp"
	"strings"
)

// Matcher 判断一个字符串是否满足某个条件。
// 这个接口演示了 Go 中最重要的适配器模式——函数类型适配器，
// 与 http.Handler / http.HandlerFunc 完全同构。
//
// 核心思想：定义一个小接口，再定义一个函数类型实现该接口，
// 这样函数和结构体都能作为 Matcher 使用——零适配成本。
type Matcher interface {
	Match(s string) bool
}

// MatcherFunc 将任意 func(string) bool 适配为 Matcher 接口。
// 这与标准库的 http.HandlerFunc 完全同一模式：
//
//	type HandlerFunc func(ResponseWriter, *Request)
//	func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }
//
// 函数类型适配器是 Go 独有的优势——Java/C# 无法为函数类型定义方法。
type MatcherFunc func(s string) bool

func (f MatcherFunc) Match(s string) bool { return f(s) }

// --- 工厂函数：返回 MatcherFunc，调用方无需关心类型 ---

// Contains 返回检查子串是否存在的 Matcher。
func Contains(substr string) Matcher {
	return MatcherFunc(func(s string) bool {
		return strings.Contains(s, substr)
	})
}

// HasPrefix 返回检查前缀的 Matcher。
func HasPrefix(prefix string) Matcher {
	return MatcherFunc(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

// HasSuffix 返回检查后缀的 Matcher。
func HasSuffix(suffix string) Matcher {
	return MatcherFunc(func(s string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

// MinLength 返回检查最小长度的 Matcher。
func MinLength(n int) Matcher {
	return MatcherFunc(func(s string) bool {
		return len(s) >= n
	})
}

// --- 结构体适配器：需要状态时使用 ---

// RegexpMatcher 用正则表达式匹配字符串。
// 与 MatcherFunc 不同，它是结构体实现——需要持有编译后的 *regexp.Regexp。
// 函数适配器和结构体适配器都满足 Matcher 接口，调用方完全无感知。
type RegexpMatcher struct {
	re *regexp.Regexp
}

// NewRegexpMatcher 编译正则表达式并返回一个 Matcher。
func NewRegexpMatcher(pattern string) (*RegexpMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexpMatcher{re: re}, nil
}

func (m *RegexpMatcher) Match(s string) bool {
	return m.re.MatchString(s)
}

// --- 组合器：Matcher → Matcher，展示适配器的可组合性 ---

// All 要求所有 Matcher 都匹配（AND 逻辑）。
func All(matchers ...Matcher) Matcher {
	return MatcherFunc(func(s string) bool {
		for _, m := range matchers {
			if !m.Match(s) {
				return false
			}
		}
		return true
	})
}

// Any 要求至少一个 Matcher 匹配（OR 逻辑）。
func Any(matchers ...Matcher) Matcher {
	return MatcherFunc(func(s string) bool {
		for _, m := range matchers {
			if m.Match(s) {
				return true
			}
		}
		return false
	})
}

// Not 取反一个 Matcher。
func Not(m Matcher) Matcher {
	return MatcherFunc(func(s string) bool {
		return !m.Match(s)
	})
}
