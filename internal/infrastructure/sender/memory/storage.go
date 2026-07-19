package memory

type Storage struct {
	Views []string
}

func (s *Storage) AddView(view string) {
	s.Views = append(s.Views, view)
}

func (s *Storage) GetViews() []string {
	return s.Views
}
