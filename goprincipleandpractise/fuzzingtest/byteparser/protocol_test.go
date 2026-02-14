package byteparser

import "testing"

/*
[]byte 解析器 Fuzz — 发现 panic 和越界访问

执行命令:

	go test -fuzz=FuzzParseMessage -fuzztime=10s

核心目标:
  对于任意 []byte 输入，ParseMessage 要么返回合法的 Message + nil error，
  要么返回 non-nil error，绝对不能 panic。

  这是 Fuzzing 最基础也最重要的用途：确保解析器对任意输入都是健壮的。
  真实的安全漏洞（如 CVE）很多就是解析器对畸形输入处理不当导致的。
*/

func FuzzParseMessage(f *testing.F) {
	// 种子语料
	f.Add([]byte{1, 0x01, 0, 0})                          // v1 PING 空 payload
	f.Add([]byte{2, 0x02, 0, 5, 'h', 'e', 'l', 'l', 'o'}) // v2 DATA
	f.Add([]byte{1, 0x03, 0, 3, 1, 2, 3})                 // v1 ECHO
	f.Add([]byte{})                                       // 空输入
	f.Add([]byte{0xff})                                   // 非法 version
	f.Add([]byte{1, 0x02, 0xff, 0xff})                    // payload 长度超大

	f.Fuzz(func(t *testing.T, data []byte) {
		msg, err := ParseMessage(data)
		if err != nil {
			return // 解析失败，符合预期
		}

		// 不变性1：解析成功时，version 必须合法
		if msg.Version != 1 && msg.Version != 2 {
			t.Errorf("parsed invalid version %d from %x", msg.Version, data)
		}

		// 不变性2：payload 长度不超过限制
		if len(msg.Payload) > 4096 {
			t.Errorf("payload too large: %d", len(msg.Payload))
		}

		// 不变性3：进一步的业务校验不应 panic
		ValidateMessage(msg) //nolint:errcheck
	})
}

// FuzzParseAndValidate 组合测试：解析 + 校验的完整链路
func FuzzParseAndValidate(f *testing.F) {
	f.Add([]byte{1, 0x01, 0, 0})
	f.Add([]byte{1, 0x02, 0, 1, 42})
	f.Add([]byte{2, 0x03, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		msg, err := ParseMessage(data)
		if err != nil {
			return
		}
		// ValidateMessage 对任意合法 Message 不应 panic
		_ = ValidateMessage(msg)
	})
}
