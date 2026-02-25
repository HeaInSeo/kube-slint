package spec

import "fmt"

// Registry는 SLI 스펙을 저장함.
type Registry struct {
	items map[string]SLISpec
}

// NewRegistry는 새로운 Registry를 생성함.
func NewRegistry() *Registry {
	return &Registry{items: map[string]SLISpec{}}
}

// Register는 SLI 스펙을 레지스트리에 추가함.
func (r *Registry) Register(s SLISpec) error {
	if s.ID == "" {
		return fmt.Errorf("sli spec id is required")
	}
	if _, exists := r.items[s.ID]; exists {
		return fmt.Errorf("sli spec already registered: %s", s.ID)
	}
	r.items[s.ID] = s
	return nil
}

// MustRegister는 SLI 스펙을 추가하며, 실패 시 패닉을 발생시킴.
func (r *Registry) MustRegister(s SLISpec) {
	if err := r.Register(s); err != nil {
		panic(err)
	}
}

// Get은 ID로 SLI 스펙을 조회함.
func (r *Registry) Get(id string) (SLISpec, bool) {
	s, ok := r.items[id]
	return s, ok
}

// List는 등록된 모든 SLI 스펙을 반환함.
func (r *Registry) List() []SLISpec {
	out := make([]SLISpec, 0, len(r.items))
	for _, s := range r.items {
		out = append(out, s)
	}
	return out
}
