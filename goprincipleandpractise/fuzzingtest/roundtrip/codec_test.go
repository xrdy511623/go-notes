package roundtrip

import "testing"

/*
Round-trip Fuzz 模式 — Fuzzing 的黄金模式

核心思路: Encode(record) → bytes → Decode(bytes) → record2 → 断言 record == record2

执行命令:

	go test -fuzz=FuzzRoundTrip -fuzztime=10s

为什么 Round-trip 模式有效:
  - 不需要知道"正确答案"是什么
  - 只需要验证一个数学性质: decode(encode(x)) == x
  - Fuzzing 引擎自动探索各种输入组合（空字符串、超长字符串、特殊字符等）
  - 如果编解码逻辑有任何不一致，都会被捕获

适用场景:
  - JSON/XML/Protobuf 等序列化库
  - 自定义二进制协议
  - 压缩/解压缩（gzip, zstd）
  - 加密/解密
*/

func FuzzRoundTrip(f *testing.F) {
	// 种子语料：覆盖典型值和边界值
	f.Add(uint8(0), "", int32(0))
	f.Add(uint8(1), "hello", int32(100))
	f.Add(uint8(255), "中文名字", int32(-1))
	f.Add(uint8(42), "\x00\xff", int32(2147483647)) // 含零字节和最大 int32
	f.Add(uint8(0), "", int32(-2147483648))         // 最小 int32

	f.Fuzz(func(t *testing.T, typ uint8, name string, score int32) {
		if len(name) > 65535 {
			t.Skip("name too long for uint16 length field")
		}

		original := Record{Type: typ, Name: name, Score: score}

		// Encode
		data, err := Encode(original)
		if err != nil {
			t.Fatalf("Encode(%+v) failed: %v", original, err)
		}

		// Decode
		decoded, err := Decode(data)
		if err != nil {
			t.Fatalf("Decode(Encode(%+v)) failed: %v", original, err)
		}

		// Round-trip 不变性: decoded 必须等于 original
		if decoded.Type != original.Type {
			t.Errorf("Type mismatch: got %d, want %d", decoded.Type, original.Type)
		}
		if decoded.Name != original.Name {
			t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
		}
		if decoded.Score != original.Score {
			t.Errorf("Score mismatch: got %d, want %d", decoded.Score, original.Score)
		}
	})
}

// FuzzDecodeNoPanic 确保任意字节输入不会导致 Decode panic
func FuzzDecodeNoPanic(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 3, 'a', 'b', 'c', 0, 0, 0, 42})
	f.Add([]byte{0xff, 0xff, 0xff}) // nameLen = 65535，但没有足够数据

	f.Fuzz(func(t *testing.T, data []byte) {
		// 不检查返回值，只确保不 panic
		Decode(data) //nolint:errcheck
	})
}
