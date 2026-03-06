package simplefactory

import (
	"testing"
)

func TestNewSender_AllScenes(t *testing.T) {
	tests := []struct {
		name  string
		scene Scene
	}{
		{"dingding", SceneDingDing},
		{"weixin", SceneWeixin},
		{"feishu", SceneFeishu},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSender(tt.scene)
			if err != nil {
				t.Fatalf("NewSender(%v) unexpected error: %v", tt.scene, err)
			}
			if s == nil {
				t.Fatal("expected non-nil Sender")
			}
		})
	}
}

func TestNewSender_Send(t *testing.T) {
	tests := []struct {
		name  string
		scene Scene
	}{
		{"dingding", SceneDingDing},
		{"weixin", SceneWeixin},
		{"feishu", SceneFeishu},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewSender(tt.scene)
			if err := s.Send("hello"); err != nil {
				t.Fatalf("Send() unexpected error: %v", err)
			}
		})
	}
}

func TestNewSender_UnknownScene(t *testing.T) {
	_, err := NewSender(Scene(999))
	if err == nil {
		t.Fatal("expected error for unknown scene, got nil")
	}
}

func TestNewSender_ZeroValue(t *testing.T) {
	_, err := NewSender(Scene(0))
	if err == nil {
		t.Fatal("expected error for zero-value scene, got nil")
	}
}

func TestNewSender_NegativeScene(t *testing.T) {
	_, err := NewSender(Scene(-1))
	if err == nil {
		t.Fatal("expected error for negative scene, got nil")
	}
}

func TestNewSender_EachCallReturnsNewInstance(t *testing.T) {
	s1, _ := NewSender(SceneDingDing)
	s2, _ := NewSender(SceneDingDing)
	if s1 == nil || s2 == nil {
		t.Fatal("expected non-nil instances")
	}
}

func TestMustNewSender_Success(t *testing.T) {
	for _, scene := range Scenes() {
		t.Run(scene.String(), func(t *testing.T) {
			s := MustNewSender(scene)
			if s == nil {
				t.Fatal("MustNewSender returned nil")
			}
		})
	}
}

func TestMustNewSender_PanicsOnUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown scene")
		}
	}()
	MustNewSender(Scene(999))
}

func TestScene_String(t *testing.T) {
	tests := []struct {
		scene Scene
		want  string
	}{
		{SceneDingDing, "dingding"},
		{SceneWeixin, "weixin"},
		{SceneFeishu, "feishu"},
		{Scene(0), "Scene(0)"},
		{Scene(999), "Scene(999)"},
		{Scene(-1), "Scene(-1)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.scene.String(); got != tt.want {
				t.Errorf("Scene.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScenes(t *testing.T) {
	scenes := Scenes()
	if len(scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(scenes))
	}

	expected := []Scene{SceneDingDing, SceneWeixin, SceneFeishu}
	for i, s := range scenes {
		if s != expected[i] {
			t.Errorf("Scenes()[%d] = %v, want %v", i, s, expected[i])
		}
	}
}

func TestScenes_AllCreateValidSender(t *testing.T) {
	for _, scene := range Scenes() {
		t.Run(scene.String(), func(t *testing.T) {
			s, err := NewSender(scene)
			if err != nil {
				t.Fatalf("NewSender(%v): %v", scene, err)
			}
			if err := s.Send("test"); err != nil {
				t.Fatalf("Send(): %v", err)
			}
		})
	}
}
