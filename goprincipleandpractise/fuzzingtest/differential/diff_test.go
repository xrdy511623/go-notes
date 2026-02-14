package differential

import (
	"net/url"
	"strings"
	"testing"
)

/*
Differential Fuzz（差分模糊测试）

执行命令:

	go test -fuzz=FuzzSplitDiff -fuzztime=10s

核心思路:
  用两个不同的实现处理同一个输入，比较结果是否一致。
  如果不一致，说明至少有一个实现有 Bug。

适用场景:
  - 重构后新旧实现的对比验证
  - 标准库 vs 第三方库的一致性检查
  - 不同版本 API 的行为对比
*/

// ---------- 示例1: 两种字符串分割实现的对比 ----------

// splitSimple 使用 strings.Split 实现
func splitSimple(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	return strings.Split(s, sep)
}

// splitManual 手动实现字符串分割（可能存在 Bug）
func splitManual(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	var result []string
	for {
		idx := strings.Index(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func FuzzSplitDiff(f *testing.F) {
	f.Add("hello,world", ",")
	f.Add("aaa", "a")
	f.Add("", ",")
	f.Add("no-separator", "X")
	f.Add("a::b::c", "::")
	f.Add("边界", "界")

	f.Fuzz(func(t *testing.T, s, sep string) {
		if sep == "" {
			t.Skip("empty separator handled differently")
		}

		got := splitManual(s, sep)
		want := splitSimple(s, sep)

		if len(got) != len(want) {
			t.Fatalf("splitManual(%q, %q) returned %d parts, splitSimple returned %d",
				s, sep, len(got), len(want))
		}

		for i := range got {
			if got[i] != want[i] {
				t.Errorf("part[%d] mismatch: splitManual=%q, splitSimple=%q",
					i, got[i], want[i])
			}
		}
	})
}

// ---------- 示例2: URL 解析的健壮性对比 ----------

// FuzzURLParse 确保 url.Parse 对任意字符串不 panic，
// 并验证 parse → string 的一致性
func FuzzURLParse(f *testing.F) {
	f.Add("https://example.com/path?q=1&b=2#frag")
	f.Add("http://user:pass@host:8080/path")
	f.Add("ftp://192.168.1.1/file.txt")
	f.Add("not-a-url")
	f.Add("")
	f.Add("://missing-scheme")
	f.Add("http://[::1]:8080")

	f.Fuzz(func(t *testing.T, rawURL string) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return
		}

		// 不变性: Parse(u.String()) 应该不报错
		// （注意：u.String() 可能与原始 rawURL 不完全一致，因为 Parse 会规范化）
		u2, err := url.Parse(u.String())
		if err != nil {
			t.Errorf("url.Parse(url.Parse(%q).String()) failed: %v", rawURL, err)
			return
		}

		// 两次解析的关键字段应一致
		if u.Scheme != u2.Scheme {
			t.Errorf("Scheme mismatch after re-parse: %q vs %q", u.Scheme, u2.Scheme)
		}
		if u.Host != u2.Host {
			t.Errorf("Host mismatch after re-parse: %q vs %q", u.Host, u2.Host)
		}
		if u.Path != u2.Path {
			t.Errorf("Path mismatch after re-parse: %q vs %q", u.Path, u2.Path)
		}
	})
}
