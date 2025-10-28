package tools

type Service struct{}

func (s *Service) GetStyleImage(style string) (string, error) {
	return "img/default.jpg", nil
}
