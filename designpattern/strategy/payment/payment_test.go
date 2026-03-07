package payment

import (
	"errors"
	"testing"
)

func TestAlipay(t *testing.T) {
	s := &Alipay{MerchantID: "M001"}
	txn, err := s.Pay(99.99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ALIPAY-M001-99.99"; txn != want {
		t.Errorf("got %q, want %q", txn, want)
	}
}

func TestWechatPay(t *testing.T) {
	s := &WechatPay{AppID: "wx123"}
	txn, err := s.Pay(50.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "WECHAT-wx123-50.00"; txn != want {
		t.Errorf("got %q, want %q", txn, want)
	}
}

func TestCreditCard(t *testing.T) {
	s := &CreditCard{Gateway: "stripe"}
	txn, err := s.Pay(120.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "CC-stripe-120.50"; txn != want {
		t.Errorf("got %q, want %q", txn, want)
	}
}

func TestAllStrategies_InvalidAmount(t *testing.T) {
	strategies := []struct {
		name string
		s    Strategy
	}{
		{"alipay", &Alipay{MerchantID: "M001"}},
		{"wechat", &WechatPay{AppID: "wx123"}},
		{"creditcard", &CreditCard{Gateway: "stripe"}},
	}
	for _, tt := range strategies {
		t.Run(tt.name+"/zero", func(t *testing.T) {
			if _, err := tt.s.Pay(0); err == nil {
				t.Error("expected error for zero amount")
			}
		})
		t.Run(tt.name+"/negative", func(t *testing.T) {
			if _, err := tt.s.Pay(-10); err == nil {
				t.Error("expected error for negative amount")
			}
		})
	}
}

func TestCheckout_SwitchStrategy(t *testing.T) {
	co := NewCheckout(&Alipay{MerchantID: "M001"})

	txn1, err := co.Pay(100)
	if err != nil {
		t.Fatal(err)
	}
	if want := "ALIPAY-M001-100.00"; txn1 != want {
		t.Errorf("first Pay got %q, want %q", txn1, want)
	}

	co.SetStrategy(&WechatPay{AppID: "wx123"})

	txn2, err := co.Pay(100)
	if err != nil {
		t.Fatal(err)
	}
	if want := "WECHAT-wx123-100.00"; txn2 != want {
		t.Errorf("second Pay got %q, want %q", txn2, want)
	}
}

func TestCheckout_NilStrategy(t *testing.T) {
	co := &Checkout{}
	if _, err := co.Pay(100); err == nil {
		t.Error("expected error for nil strategy")
	}
}

func TestStrategyFunc_AsStrategy(t *testing.T) {
	mockPay := StrategyFunc(func(amount float64) (string, error) {
		if amount > 1000 {
			return "", errors.New("limit exceeded")
		}
		return "MOCK-OK", nil
	})

	co := NewCheckout(mockPay)

	txn, err := co.Pay(500)
	if err != nil {
		t.Fatal(err)
	}
	if txn != "MOCK-OK" {
		t.Errorf("got %q, want MOCK-OK", txn)
	}

	if _, err := co.Pay(2000); err == nil {
		t.Error("expected error for amount > 1000")
	}
}

func TestStrategyFunc_ImplementsInterface(t *testing.T) {
	var s Strategy = StrategyFunc(func(amount float64) (string, error) {
		return "FUNC", nil
	})
	txn, err := s.Pay(1)
	if err != nil {
		t.Fatal(err)
	}
	if txn != "FUNC" {
		t.Errorf("got %q, want FUNC", txn)
	}
}

func TestCheckout_MultipleSwitch(t *testing.T) {
	strategies := []struct {
		name    string
		s       Strategy
		wantPfx string
	}{
		{"alipay", &Alipay{MerchantID: "M001"}, "ALIPAY-"},
		{"wechat", &WechatPay{AppID: "wx123"}, "WECHAT-"},
		{"creditcard", &CreditCard{Gateway: "stripe"}, "CC-"},
		{"custom_func", StrategyFunc(func(amount float64) (string, error) {
			return "CUSTOM", nil
		}), "CUSTOM"},
	}

	co := NewCheckout(nil)
	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			co.SetStrategy(tt.s)
			txn, err := co.Pay(10)
			if err != nil {
				t.Fatal(err)
			}
			if len(txn) < len(tt.wantPfx) || txn[:len(tt.wantPfx)] != tt.wantPfx {
				t.Errorf("got %q, want prefix %q", txn, tt.wantPfx)
			}
		})
	}
}

func BenchmarkCheckout(b *testing.B) {
	b.Run("interface_strategy", func(b *testing.B) {
		co := NewCheckout(&Alipay{MerchantID: "M001"})
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			co.Pay(99.99)
		}
	})

	b.Run("func_strategy", func(b *testing.B) {
		co := NewCheckout(StrategyFunc(func(amount float64) (string, error) {
			return "FAST", nil
		}))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			co.Pay(99.99)
		}
	})
}
